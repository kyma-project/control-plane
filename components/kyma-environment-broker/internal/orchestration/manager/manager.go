package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/strategies"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type OperationFactory interface {
	NewOperation(o internal.Orchestration, r orchestration.Runtime, i internal.Instance, state domain.LastOperationState) (orchestration.RuntimeOperation, error)
	ResumeOperations(orchestrationID string) ([]orchestration.RuntimeOperation, error)
	CancelOperations(orchestrationID string) error
	RetryOperations(orchestrationID string, schedule orchestration.ScheduleType, policy orchestration.MaintenancePolicy, updateMWindow bool) ([]orchestration.RuntimeOperation, error)
}

type orchestrationManager struct {
	orchestrationStorage storage.Orchestrations
	operationStorage     storage.Operations
	instanceStorage      storage.Instances
	resolver             orchestration.RuntimeResolver
	factory              OperationFactory
	executor             orchestration.OperationExecutor
	log                  logrus.FieldLogger
	pollingInterval      time.Duration
	k8sClient            client.Client
	configNamespace      string
	configName           string
	kymaVersion          string
	kubernetesVersion    string
	bundleBuilder        notification.BundleBuilder
	speedFactor          int
}

const maintenancePolicyKeyName = "maintenancePolicy"
const maintenanceWindowFormat = "150405-0700"

func (m *orchestrationManager) SpeedUp(factor int) {
	m.speedFactor = factor
}

func (m *orchestrationManager) Execute(orchestrationID string) (time.Duration, error) {
	logger := m.log.WithField("orchestrationID", orchestrationID)
	m.log.Infof("Processing orchestration %s", orchestrationID)
	o, err := m.orchestrationStorage.GetByID(orchestrationID)
	if err != nil {
		if o == nil {
			m.log.Errorf("orchestration %s failed: %s", orchestrationID, err)
			return time.Minute, nil
		}
		return m.failOrchestration(o, errors.Wrap(err, "while getting orchestration"))
	}

	maintenancePolicy, err := m.getMaintenancePolicy()
	if err != nil {
		m.log.Warnf("while getting maintenance policy: %s", err)
	}

	operations, err := m.resolveOperations(o, maintenancePolicy)
	if err != nil {
		return m.failOrchestration(o, errors.Wrap(err, "while resolving operations"))
	}

	err = m.orchestrationStorage.Update(*o)
	if err != nil {
		logger.Errorf("while updating orchestration: %v", err)
		return m.pollingInterval, nil
	}
	// do not perform any action if the orchestration is finished
	if o.IsFinished() {
		m.log.Infof("Orchestration was already finished, state: %s", o.State)
		return 0, nil
	}

	strategy := m.resolveStrategy(o.Parameters.Strategy.Type, m.executor, logger)

	// ctreate notification after orchestration resolved
	if !m.bundleBuilder.DisabledCheck() {
		err := m.sendNotificationCreate(o, operations)
		//currently notification error can only be temporary error
		if err != nil && kebError.IsTemporaryError(err) {
			return 5 * time.Second, nil
		}
	}

	execID, err := strategy.Execute(operations, o.Parameters.Strategy)
	if err != nil {
		return 0, errors.Wrap(err, "while executing upgrade strategy")
	}

	o, err = m.waitForCompletion(o, strategy, execID, logger)
	if err != nil && kebError.IsTemporaryError(err) {
		return 5 * time.Second, nil
	} else if err != nil {
		return 0, errors.Wrap(err, "while waiting for orchestration to finish")
	}

	o.UpdatedAt = time.Now()
	err = m.orchestrationStorage.Update(*o)
	if err != nil {
		logger.Errorf("while updating orchestration: %v", err)
		return m.pollingInterval, nil
	}

	logger.Infof("Finished processing orchestration, state: %s", o.State)
	return 0, nil
}

func (m *orchestrationManager) getMaintenancePolicy() (orchestration.MaintenancePolicy, error) {
	policy := orchestration.MaintenancePolicy{}
	config := &coreV1.ConfigMap{}
	key := client.ObjectKey{Namespace: m.configNamespace, Name: m.configName}
	if err := m.k8sClient.Get(context.Background(), key, config); err != nil {
		return policy, errors.New("orchestration config is absent")
	}

	if config.Data[maintenancePolicyKeyName] == "" {
		return policy, errors.New("maintenance policy is absent from orchestration config")
	}

	err := json.Unmarshal([]byte(config.Data[maintenancePolicyKeyName]), &policy)
	if err != nil {
		return policy, errors.New("failed to unmarshal the policy config")
	}

	return policy, nil
}

func (m *orchestrationManager) resolveOperations(o *internal.Orchestration, policy orchestration.MaintenancePolicy) ([]orchestration.RuntimeOperation, error) {
	result := []orchestration.RuntimeOperation{}
	if o.State == orchestration.Pending {
		fmt.Println("manager.go resolveOperations() o.State = ", o.State)
		runtimes, err := m.resolver.Resolve(o.Parameters.Targets)
		if err != nil {
			return result, errors.Wrap(err, "while resolving targets")
		}

		for _, r := range runtimes {
			windowBegin := time.Time{}
			windowEnd := time.Time{}
			days := []string{}

			if o.Parameters.Strategy.Schedule == orchestration.MaintenanceWindow {
				windowBegin, windowEnd, days = resolveMaintenanceWindowTime(r, policy)
			}
			r.MaintenanceWindowBegin = windowBegin
			r.MaintenanceWindowEnd = windowEnd
			r.MaintenanceDays = days

			inst, err := m.instanceStorage.GetByID(r.InstanceID)
			if err != nil {
				return nil, errors.Wrapf(err, "while getting instance %s", r.InstanceID)
			}

			op, err := m.factory.NewOperation(*o, r, *inst, orchestration.Pending)
			if err != nil {
				return nil, errors.Wrapf(err, "while creating new operation for runtime id %q", r.RuntimeID)
			}

			result = append(result, op)
		}

		if o.Parameters.Kyma == nil || o.Parameters.Kyma.Version == "" {
			o.Parameters.Kyma = &orchestration.KymaParameters{Version: m.kymaVersion}
		}
		if o.Parameters.Kubernetes == nil || o.Parameters.Kubernetes.KubernetesVersion == "" {
			o.Parameters.Kubernetes = &orchestration.KubernetesParameters{KubernetesVersion: m.kubernetesVersion}
		}

		if len(runtimes) != 0 {
			o.State = orchestration.InProgress
		} else {
			o.State = orchestration.Succeeded
		}
		o.Description = fmt.Sprintf("Scheduled %d operations", len(runtimes))
	} else if o.State == orchestration.Retrying {
		fmt.Println("manager.go resolveOperations() o.Parameters = ", o.Parameters)
		runtimes, err := m.resolver.Resolve(o.Parameters.Targets)
		if err != nil {
			return result, errors.Wrap(err, "while resolving targets")
		}

		for _, r := range runtimes {
			windowBegin := time.Time{}
			windowEnd := time.Time{}
			days := []string{}

			if o.Parameters.Strategy.Schedule == orchestration.MaintenanceWindow {
				windowBegin, windowEnd, days = resolveMaintenanceWindowTime(r, policy)
			}
			r.MaintenanceWindowBegin = windowBegin
			r.MaintenanceWindowEnd = windowEnd
			r.MaintenanceDays = days

			inst, err := m.instanceStorage.GetByID(r.InstanceID)
			if err != nil {
				return nil, errors.Wrapf(err, "while getting instance %s", r.InstanceID)
			}

			op, err := m.factory.NewOperation(*o, r, *inst, orchestration.Retrying)
			if err != nil {
				return nil, errors.Wrapf(err, "while creating new operation for runtime id %q", r.RuntimeID)
			}

			result = append(result, op)
		}

		if o.Parameters.Kyma == nil || o.Parameters.Kyma.Version == "" {
			o.Parameters.Kyma = &orchestration.KymaParameters{Version: m.kymaVersion}
		}
		if o.Parameters.Kubernetes == nil || o.Parameters.Kubernetes.KubernetesVersion == "" {
			o.Parameters.Kubernetes = &orchestration.KubernetesParameters{KubernetesVersion: m.kubernetesVersion}
		}
		// look for the ops with retrying state, then convert the op state to pending and orchestration state to in progress
		_, err = m.factory.RetryOperations(o.OrchestrationID, o.Parameters.Strategy.Schedule, policy, true)

		if err != nil {
			return result, errors.Wrap(err, "while resolving retrying orchestration")
		}

		if len(result) != 0 {
			o.State = orchestration.InProgress
		} else {
			o.State = orchestration.Succeeded
		}
		fmt.Println("manager.go o.State retrying branch o.State = ", o.State)
		o.Description = updateRetryingDescription(o.Description, fmt.Sprintf("retried %d operations", len(result)))
	} else {
		// Resume processing of not finished upgrade operations after restart
		fmt.Println("manager.go others")
		var err error
		result, err = m.factory.ResumeOperations(o.OrchestrationID)
		if err != nil {
			return result, err
		}

		m.log.Infof("Resuming %d operations for orchestration %s", len(result), o.OrchestrationID)
	}

	return result, nil
}

func (m *orchestrationManager) resolveStrategy(sType orchestration.StrategyType, executor orchestration.OperationExecutor, log logrus.FieldLogger) orchestration.Strategy {
	switch sType {
	case orchestration.ParallelStrategy:
		s := strategies.NewParallelOrchestrationStrategy(executor, log, 0)
		if m.speedFactor != 0 {
			s.SpeedUp(m.speedFactor)
		}
		return s
	}
	return nil
}

// waitForCompletion waits until processing of given orchestration ends or if it's canceled
func (m *orchestrationManager) waitForCompletion(o *internal.Orchestration, strategy orchestration.Strategy, execID string, log logrus.FieldLogger) (*internal.Orchestration, error) {
	orchestrationID := o.OrchestrationID
	canceled := false
	var err error
	var stats map[string]int
	err = wait.PollImmediateInfinite(m.pollingInterval, func() (bool, error) {
		// check if orchestration wasn't canceled
		o, err = m.orchestrationStorage.GetByID(orchestrationID)
		switch {
		case err == nil:
			if o.State == orchestration.Canceling {
				log.Info("Orchestration was canceled")
				canceled = true
			}
		case dberr.IsNotFound(err):
			log.Errorf("while getting orchestration: %v", err)
			return false, err
		default:
			log.Errorf("while getting orchestration: %v", err)
			return false, nil
		}
		s, err := m.operationStorage.GetOperationStatsForOrchestration(o.OrchestrationID)
		if err != nil {
			log.Errorf("while getting operations: %v", err)
			return false, nil
		}
		stats = s

		numberOfNotFinished := 0
		numberOfInProgress, found := stats[orchestration.InProgress]
		if found {
			numberOfNotFinished += numberOfInProgress
		}
		numberOfPending, found := stats[orchestration.Pending]
		if found {
			numberOfNotFinished += numberOfPending
		}
		numberOfRetrying, found := stats[orchestration.Retrying]
		if found {
			// handle the retrying ops during in progress orchestration
			// use the existing resolved policy in op
			numberOfNotFinished += numberOfRetrying
			ops, err := m.factory.RetryOperations(o.OrchestrationID, o.Parameters.Strategy.Schedule, orchestration.MaintenancePolicy{}, false)
			if err != nil {
				// don't block the polling and cancel signal
				log.Errorf("while handling retrying operations: %v", err)
			} else {
				err := strategy.Insert(execID, ops, o.Parameters.Strategy)
				if err != nil {
					return false, errors.Wrap(err, "while inserting operations to queue")
				}
			}

		}

		// don't wait for pending operations if orchestration was canceled
		if canceled {
			return numberOfInProgress == 0, nil
		} else {
			return numberOfNotFinished == 0, nil
		}
	})
	if err != nil {
		return nil, errors.Wrap(err, "while waiting for scheduled operations to finish")
	}

	return m.resolveOrchestration(o, strategy, execID, stats)
}

func (m *orchestrationManager) resolveOrchestration(o *internal.Orchestration, strategy orchestration.Strategy, execID string, stats map[string]int) (*internal.Orchestration, error) {
	if o.State == orchestration.Canceling {
		err := m.factory.CancelOperations(o.OrchestrationID)
		if err != nil {
			return nil, errors.Wrap(err, "while resolving canceled operations")
		}
		strategy.Cancel(execID)
		// Send customer notification for cancel
		if !m.bundleBuilder.DisabledCheck() {
			err := m.sendNotificationCancel(o)
			//currently notification error can only be temporary error
			if err != nil && kebError.IsTemporaryError(err) {
				return nil, err
			}
		}
		o.State = orchestration.Canceled
	} else {
		state := orchestration.Succeeded
		if stats[orchestration.Failed] > 0 {
			state = orchestration.Failed
		}
		o.State = state
	}
	return o, nil
}

// resolves the next exact maintenance window time for the runtime
func resolveMaintenanceWindowTime(r orchestration.Runtime, policy orchestration.MaintenancePolicy) (time.Time, time.Time, []string) {
	ruleMatched := false

	for _, p := range policy.Rules {
		if p.Match.Plan != "" {
			matched, err := regexp.MatchString(p.Match.Plan, r.Plan)
			if err != nil || !matched {
				continue
			}
		}

		if p.Match.GlobalAccountID != "" {
			matched, err := regexp.MatchString(p.Match.GlobalAccountID, r.GlobalAccountID)
			if err != nil || !matched {
				continue
			}
		}

		if p.Match.Region != "" {
			matched, err := regexp.MatchString(p.Match.Region, r.Region)
			if err != nil || !matched {
				continue
			}
		}

		// We have a rule match here, either by one or all of the rule match options. Let's override maintenance attributes.
		ruleMatched = true
		if len(p.Days) > 0 {
			r.MaintenanceDays = p.Days
		}
		if p.TimeBegin != "" {
			if maintenanceWindowBegin, err := time.Parse(maintenanceWindowFormat, p.TimeBegin); err == nil {
				r.MaintenanceWindowBegin = maintenanceWindowBegin
			}
		}
		if p.TimeEnd != "" {
			if maintenanceWindowEnd, err := time.Parse(maintenanceWindowFormat, p.TimeEnd); err == nil {
				r.MaintenanceWindowEnd = maintenanceWindowEnd
			}
		}
		break
	}

	// If non of the rules matched, try to apply the default rule
	if !ruleMatched {
		if len(policy.Default.Days) > 0 {
			r.MaintenanceDays = policy.Default.Days
		}
		if policy.Default.TimeBegin != "" {
			if maintenanceWindowBegin, err := time.Parse(maintenanceWindowFormat, policy.Default.TimeBegin); err == nil {
				r.MaintenanceWindowBegin = maintenanceWindowBegin
			}
		}
		if policy.Default.TimeEnd != "" {
			if maintenanceWindowEnd, err := time.Parse(maintenanceWindowFormat, policy.Default.TimeEnd); err == nil {
				r.MaintenanceWindowEnd = maintenanceWindowEnd
			}
		}
	}

	n := time.Now()
	availableDays := orchestration.ConvertSliceOfDaysToMap(r.MaintenanceDays)
	start := time.Date(n.Year(), n.Month(), n.Day(), r.MaintenanceWindowBegin.Hour(), r.MaintenanceWindowBegin.Minute(), r.MaintenanceWindowBegin.Second(), r.MaintenanceWindowBegin.Nanosecond(), r.MaintenanceWindowBegin.Location())
	end := time.Date(n.Year(), n.Month(), n.Day(), r.MaintenanceWindowEnd.Hour(), r.MaintenanceWindowEnd.Minute(), r.MaintenanceWindowEnd.Second(), r.MaintenanceWindowEnd.Nanosecond(), r.MaintenanceWindowEnd.Location())
	// Set start/end date to the first available day (including today)
	diff := orchestration.FirstAvailableDayDiff(n.Weekday(), availableDays)
	start = start.AddDate(0, 0, diff)
	end = end.AddDate(0, 0, diff)

	// if the window end slips through the next day, adjust the date accordingly
	if end.Before(start) || end.Equal(start) {
		end = end.AddDate(0, 0, 1)
	}

	// if time window has already passed we wait until next available day
	if start.Before(n) && end.Before(n) {
		diff := orchestration.NextAvailableDayDiff(n.Weekday(), availableDays)
		start = start.AddDate(0, 0, diff)
		end = end.AddDate(0, 0, diff)
	}

	return start, end, r.MaintenanceDays
}

func (m *orchestrationManager) failOrchestration(o *internal.Orchestration, err error) (time.Duration, error) {
	m.log.Errorf("orchestration %s failed: %s", o.OrchestrationID, err)
	return m.updateOrchestration(o, orchestration.Failed, err.Error()), nil
}

func (m *orchestrationManager) updateOrchestration(o *internal.Orchestration, state, description string) time.Duration {
	o.UpdatedAt = time.Now()
	o.State = state
	o.Description = description
	err := m.orchestrationStorage.Update(*o)
	if err != nil {
		if !dberr.IsNotFound(err) {
			m.log.Errorf("while updating orchestration: %v", err)
			return time.Minute
		}
	}
	return 0
}

func (m *orchestrationManager) sendNotificationCreate(o *internal.Orchestration, operations []orchestration.RuntimeOperation) error {
	if o.State == orchestration.InProgress {
		if o.Parameters.NotificationState == "" {
			m.log.Info("Initialize notification status")
			o.Parameters.NotificationState = orchestration.NotificationPending
		}
		//Skip sending create signal if notification already existed
		if o.Parameters.NotificationState == orchestration.NotificationPending {
			eventType := ""
			tenants := []notification.NotificationTenant{}
			if o.Type == orchestration.UpgradeKymaOrchestration {
				eventType = notification.KymaMaintenanceNumber
			} else if o.Type == orchestration.UpgradeClusterOrchestration {
				eventType = notification.KubernetesMaintenanceNumber
			}
			for _, op := range operations {
				startDate := ""
				endDate := ""
				if o.Parameters.Strategy.Schedule == orchestration.MaintenanceWindow {
					startDate = op.Runtime.MaintenanceWindowBegin.String()
					endDate = op.Runtime.MaintenanceWindowEnd.String()
				} else {
					startDate = time.Now().Format("2006-01-02 15:04:05")
				}
				tenant := notification.NotificationTenant{
					InstanceID: op.Runtime.InstanceID,
					StartDate:  startDate,
					EndDate:    endDate,
				}
				tenants = append(tenants, tenant)
			}
			notificationParams := notification.NotificationParams{
				OrchestrationID: o.OrchestrationID,
				EventType:       eventType,
				Tenants:         tenants,
			}
			m.log.Info("Start to create notification")
			notificationBundle, err := m.bundleBuilder.NewBundle(o.OrchestrationID, notificationParams)
			if err != nil {
				m.log.Errorf("%s: %s", "failed to create Notification Bundle", err)
				return err
			}
			err = notificationBundle.CreateNotificationEvent()
			if err != nil {
				m.log.Errorf("%s: %s", "cannot send notification", err)
				return err
			}
			m.log.Info("Creating notification succedded")
			o.Parameters.NotificationState = orchestration.NotificationCreated
		}
	}
	return nil
}

func (m *orchestrationManager) sendNotificationCancel(o *internal.Orchestration) error {
	if o.Parameters.NotificationState == orchestration.NotificationCreated {
		notificationParams := notification.NotificationParams{
			OrchestrationID: o.OrchestrationID,
		}
		m.log.Info("Start to cancel notification")
		notificationBundle, err := m.bundleBuilder.NewBundle(o.OrchestrationID, notificationParams)
		if err != nil {
			m.log.Errorf("%s: %s", "failed to create Notification Bundle", err)
			return err
		}
		err = notificationBundle.CancelNotificationEvent()
		if err != nil {
			m.log.Errorf("%s: %s", "cannot cancel notification", err)
			return err
		}
		m.log.Info("Cancelling notification succedded")
		o.Parameters.NotificationState = orchestration.NotificationCancelled
	}
	return nil
}

func updateRetryingDescription(desc string, newDesc string) string {
	if strings.Contains(desc, "retrying") {
		return strings.Replace(desc, "retrying", newDesc, -1)
	}

	return desc + ", " + newDesc
}
