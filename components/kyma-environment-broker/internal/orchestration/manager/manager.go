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
	"github.com/sirupsen/logrus"
)

type OperationFactory interface {
	NewOperation(o internal.Orchestration, r orchestration.Runtime, i internal.Instance, state domain.LastOperationState) (orchestration.RuntimeOperation, error)
	ResumeOperations(orchestrationID string) ([]orchestration.RuntimeOperation, error)
	CancelOperation(orchestrationID string, runtimeID string) error
	CancelOperations(orchestrationID string) error
	RetryOperations(operationIDs []string) ([]orchestration.RuntimeOperation, error)
	QueryOperation(orchestrationID string, r orchestration.Runtime) (bool, orchestration.RuntimeOperation, error)
	QueryOperations(orchestrationID string) ([]orchestration.RuntimeOperation, error)
	NotifyOperation(orchestrationID string, runtimeID string, oState string, notifyState orchestration.NotificationStateType) error
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
		return m.failOrchestration(o, fmt.Errorf("failed to get orchestration: %w", err))
	}

	operations, runtimeNums, err := m.waitForStart(o)
	if err != nil {
		m.failOrchestration(o, fmt.Errorf("failed while waiting start for operations: %w", err))
	}

	if o.Parameters.Kyma == nil || o.Parameters.Kyma.Version == "" {
		o.Parameters.Kyma = &orchestration.KymaParameters{Version: m.kymaVersion}
	}
	if o.Parameters.Kubernetes == nil || o.Parameters.Kubernetes.KubernetesVersion == "" {
		o.Parameters.Kubernetes = &orchestration.KubernetesParameters{KubernetesVersion: m.kubernetesVersion}
	}

	if o.State == orchestration.Pending || o.State == orchestration.Retrying {
		if runtimeNums != 0 {
			o.State = orchestration.InProgress
		} else {
			o.State = orchestration.Succeeded
		}
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

	execID, err := strategy.Execute(operations, o.Parameters.Strategy)
	if err != nil {
		return 0, fmt.Errorf("failed to execute strategy: %w", err)
	}

	o, err = m.waitForCompletion(o, strategy, execID, logger)
	if err != nil && kebError.IsTemporaryError(err) {
		return 5 * time.Second, nil
	} else if err != nil {
		return 0, fmt.Errorf("while waiting for orchestration to finish: %w", err)
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
		return policy, fmt.Errorf("orchestration config is absent")
	}

	if config.Data[maintenancePolicyKeyName] == "" {
		return policy, fmt.Errorf("maintenance policy is absent from orchestration config")
	}

	err := json.Unmarshal([]byte(config.Data[maintenancePolicyKeyName]), &policy)
	if err != nil {
		return policy, fmt.Errorf("failed to unmarshal the policy config")
	}

	return policy, nil
}

// result contains the operations which from `kcp o *** retry` and its label are retrying, runtimes from target parameter
func (m *orchestrationManager) extractRuntimes(o *internal.Orchestration, runtimes []orchestration.Runtime, result []orchestration.RuntimeOperation) []orchestration.Runtime {
	var fileterRuntimes []orchestration.Runtime
	if o.State == orchestration.Pending {
		fileterRuntimes = runtimes
	} else {
		// o.State = retrying / in progress
		for _, retryOp := range result {
			for _, r := range runtimes {
				if retryOp.Runtime.InstanceID == r.InstanceID {
					fileterRuntimes = append(fileterRuntimes, r)
					break
				}
			}
		}
	}
	return fileterRuntimes
}

func (m *orchestrationManager) NewOperationForPendingRetrying(o *internal.Orchestration, policy orchestration.MaintenancePolicy, retryRT []orchestration.RuntimeOperation, updateWindow bool) ([]orchestration.RuntimeOperation, *internal.Orchestration, []orchestration.Runtime, error) {
	result := []orchestration.RuntimeOperation{}
	runtimes, err := m.resolver.Resolve(o.Parameters.Targets)
	if err != nil {
		return result, o, runtimes, fmt.Errorf("while resolving targets: %w", err)
	}

	fileterRuntimes := m.extractRuntimes(o, runtimes, retryRT)

	for _, r := range fileterRuntimes {
		var op orchestration.RuntimeOperation
		if o.State == orchestration.Pending {
			exist, op, err := m.factory.QueryOperation(o.OrchestrationID, r)
			if err != nil {
				return nil, o, runtimes, fmt.Errorf("while quering operation for runtime id %q: %w", r.RuntimeID, err)
			}
			if exist {
				result = append(result, op)
				continue
			}
		}
		if updateWindow {
			windowBegin := time.Time{}
			windowEnd := time.Time{}
			days := []string{}

			if o.State == orchestration.Pending && o.Parameters.Strategy.MaintenanceWindow {
				windowBegin, windowEnd, days = resolveMaintenanceWindowTime(r, policy, o.Parameters.Strategy.ScheduleTime)
			}
			if o.State == orchestration.Retrying && bool(o.Parameters.RetryOperation.Immediate) && o.Parameters.Strategy.MaintenanceWindow {
				windowBegin, windowEnd, days = resolveMaintenanceWindowTime(r, policy, o.Parameters.Strategy.ScheduleTime)
			}

			r.MaintenanceWindowBegin = windowBegin
			r.MaintenanceWindowEnd = windowEnd
			r.MaintenanceDays = days
		} else {
			if o.Parameters.RetryOperation.Immediate {
				r.MaintenanceWindowBegin = time.Time{}
				r.MaintenanceWindowEnd = time.Time{}
				r.MaintenanceDays = []string{}
			}
		}

		inst, err := m.instanceStorage.GetByID(r.InstanceID)
		if err != nil {
			return nil, o, runtimes, fmt.Errorf("while getting instance %s: %w", r.InstanceID, err)
		}

		op, err = m.factory.NewOperation(*o, r, *inst, orchestration.Pending)
		if err != nil {
			return nil, o, runtimes, fmt.Errorf("while creating new operation for runtime id %q: %w", r.RuntimeID, err)
		}

		result = append(result, op)

	}

	return result, o, fileterRuntimes, nil
}

func (m *orchestrationManager) cancelOperationForNonExistent(o *internal.Orchestration, resolvedOperations []orchestration.RuntimeOperation) error {
	storageOperations, err := m.factory.QueryOperations(o.OrchestrationID)
	if err != nil {
		return fmt.Errorf("while quering operations for orchestration %s: %w", o.OrchestrationID, err)
	}
	var storageOpIDs []string
	for _, storageOperation := range storageOperations {
		storageOpIDs = append(storageOpIDs, storageOperation.Runtime.RuntimeID)
	}
	var resolvedOpIDs []string
	for _, resolvedOperation := range resolvedOperations {
		resolvedOpIDs = append(resolvedOpIDs, resolvedOperation.Runtime.RuntimeID)
	}

	//find diffs that exist in storageOperations but not in resolvedOperations
	operationIdMap := make(map[string]struct{}, len(resolvedOpIDs))
	for _, resolvedOpID := range resolvedOpIDs {
		operationIdMap[resolvedOpID] = struct{}{}
	}
	var nonExistentIDs []string
	for _, storageOpID := range storageOpIDs {
		if _, found := operationIdMap[storageOpID]; !found {
			nonExistentIDs = append(nonExistentIDs, storageOpID)
		}
	}

	//cancel operations for non existent runtimes
	for _, nonExistentID := range nonExistentIDs {
		err := m.factory.CancelOperation(o.OrchestrationID, nonExistentID)
		if err != nil {
			return fmt.Errorf("while resolving canceled operations for runtime id %q: %w", nonExistentID, err)
		}
	}

	return nil
}

func (m *orchestrationManager) resolveOperations(o *internal.Orchestration, policy orchestration.MaintenancePolicy) ([]orchestration.RuntimeOperation, []orchestration.Runtime, error) {
	result := []orchestration.RuntimeOperation{}
	filterRuntimes := []orchestration.Runtime{}
	if o.State == orchestration.Pending {
		var err error
		result, o, filterRuntimes, err = m.NewOperationForPendingRetrying(o, policy, result, true)
		if err != nil {
			return nil, filterRuntimes, fmt.Errorf("while creating new operation for pending: %w", err)
		}
		//cancel operations that no longer exist, and cancel their notification
		err = m.cancelOperationForNonExistent(o, result)
		if err != nil {
			return nil, filterRuntimes, fmt.Errorf("while canceling non existent operation for pending: %w", err)
		}

		o.Description = fmt.Sprintf("Scheduled %d operations", len(filterRuntimes))
	} else if o.State == orchestration.Retrying {
		//check retry operation list, if empty return error
		if len(o.Parameters.RetryOperation.RetryOperations) == 0 {
			return nil, filterRuntimes, fmt.Errorf("while retrying operations: %w",
				fmt.Errorf("o.Parameters.RetryOperation.RetryOperations is empty"))
		}
		retryRuntimes, err := m.factory.RetryOperations(o.Parameters.RetryOperation.RetryOperations)
		if err != nil {
			return retryRuntimes, filterRuntimes, fmt.Errorf("while resolving retrying orchestration: %w", err)
		}

		result, o, filterRuntimes, err = m.NewOperationForPendingRetrying(o, policy, retryRuntimes, true)

		if err != nil {
			return nil, filterRuntimes, fmt.Errorf("while NewOperationForPendingRetrying: %w", err)
		}

		o.Description = updateRetryingDescription(o.Description, fmt.Sprintf("retried %d operations", len(filterRuntimes)))
		o.Parameters.RetryOperation.RetryOperations = nil
		o.Parameters.RetryOperation.Immediate = false
		m.log.Infof("Resuming %d operations for orchestration %s", len(result), o.OrchestrationID)
	} else {
		// Resume processing of not finished upgrade operations after restart
		var err error
		result, err = m.factory.ResumeOperations(o.OrchestrationID)
		if err != nil {
			return result, filterRuntimes, fmt.Errorf("while resuming operation: %w", err)
		}

		m.log.Infof("Resuming %d operations for orchestration %s", len(result), o.OrchestrationID)
	}

	return result, filterRuntimes, nil
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
	execIDs := []string{execID}

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
			numberOfNotFinished += numberOfRetrying
		}

		if len(o.Parameters.RetryOperation.RetryOperations) > 0 {
			ops, err := m.factory.RetryOperations(o.Parameters.RetryOperation.RetryOperations)
			if err != nil {
				// don't block the polling and cancel signal
				log.Errorf("PollImmediateInfinite() while handling retrying operations: %v", err)
			}

			result, o, _, err := m.NewOperationForPendingRetrying(o, orchestration.MaintenancePolicy{}, ops, false)
			if err != nil {
				log.Errorf("PollImmediateInfinite() while new operation for retrying instanceid : %v", err)
			}

			err = strategy.Insert(execID, result, o.Parameters.Strategy)
			if err != nil {
				retryExecID, err := strategy.Execute(result, o.Parameters.Strategy)
				if err != nil {
					return false, fmt.Errorf("while executing upgrade strategy during retrying: %w", err)
				}
				execIDs = append(execIDs, retryExecID)
				execID = retryExecID
			}
			o.Description = updateRetryingDescription(o.Description, fmt.Sprintf("retried %d operations", len(o.Parameters.RetryOperation.RetryOperations)))
			o.Parameters.RetryOperation.RetryOperations = nil
			o.Parameters.RetryOperation.Immediate = false

			err = m.orchestrationStorage.Update(*o)
			if err != nil {
				log.Errorf("PollImmediateInfinite() while updating orchestration: %v", err)
				return false, nil
			}
			m.log.Infof("PollImmediateInfinite() while resuming %d operations for orchestration %s", len(result), o.OrchestrationID)
		}

		// don't wait for pending operations if orchestration was canceled
		if canceled {
			return numberOfInProgress == 0, nil
		} else {
			return numberOfNotFinished == 0, nil
		}
	})
	if err != nil {
		return nil, fmt.Errorf("while waiting for scheduled operations to finish: %w", err)
	}

	return m.resolveOrchestration(o, strategy, execIDs, stats)
}

func (m *orchestrationManager) resolveOrchestration(o *internal.Orchestration, strategy orchestration.Strategy, execIDs []string, stats map[string]int) (*internal.Orchestration, error) {
	if o.State == orchestration.Canceling {
		err := m.factory.CancelOperations(o.OrchestrationID)
		if err != nil {
			return nil, fmt.Errorf("while resolving canceled operations: %w", err)
		}
		for _, execID := range execIDs {
			strategy.Cancel(execID)
		}
		// Send customer notification for cancel
		if o.Parameters.Notification {
			operations, err := m.factory.QueryOperations(o.OrchestrationID)
			if err != nil {
				return nil, fmt.Errorf("while quering operations for orchestration %s: %w", o.OrchestrationID, err)
			}
			err = m.sendNotificationCancel(o, operations)
			//currently notification error can only be temporary error
			if err != nil && kebError.IsTemporaryError(err) {
				return nil, err
			}
			//update notification state for notified operations
			for _, operation := range operations {
				runtimeID := operation.Runtime.RuntimeID
				err = m.factory.NotifyOperation(o.OrchestrationID, runtimeID, orchestration.Canceling, orchestration.NotificationCancelled)
				if err != nil {
					return nil, fmt.Errorf("while updaring operation for runtime id %q: %w", runtimeID, err)
				}
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
func resolveMaintenanceWindowTime(r orchestration.Runtime, policy orchestration.MaintenancePolicy, after time.Time) (time.Time, time.Time, []string) {
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
	// If 'after' is in the future, set it as timepoint for the maintenance window calculation
	if after.After(n) {
		n = after
	}
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
	eventType := ""
	tenants := []notification.NotificationTenant{}
	if o.Type == orchestration.UpgradeKymaOrchestration {
		eventType = notification.KymaMaintenanceNumber
	} else if o.Type == orchestration.UpgradeClusterOrchestration {
		eventType = notification.KubernetesMaintenanceNumber
	}

	for _, operation := range operations {
		startDate := ""
		endDate := ""
		if o.Parameters.Strategy.MaintenanceWindow {
			startDate = operation.Runtime.MaintenanceWindowBegin.String()
			endDate = operation.Runtime.MaintenanceWindowEnd.String()
		} else {
			startDate = time.Now().Format("2006-01-02 15:04:05")
		}
		tenant := notification.NotificationTenant{
			InstanceID: operation.Runtime.InstanceID,
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

	return nil
}

func (m *orchestrationManager) sendNotificationCancel(o *internal.Orchestration, ops []orchestration.RuntimeOperation) error {
	eventType := ""
	tenants := []notification.NotificationTenant{}
	if o.Type == orchestration.UpgradeKymaOrchestration {
		eventType = notification.KymaMaintenanceNumber
	} else if o.Type == orchestration.UpgradeClusterOrchestration {
		eventType = notification.KubernetesMaintenanceNumber
	}
	for _, op := range ops {
		if op.NotificationState == orchestration.NotificationCreated {
			tenant := notification.NotificationTenant{
				InstanceID: op.Runtime.InstanceID,
			}
			tenants = append(tenants, tenant)
		}
	}
	notificationParams := notification.NotificationParams{
		OrchestrationID: o.OrchestrationID,
		EventType:       eventType,
		Tenants:         tenants,
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

	return nil
}

func updateRetryingDescription(desc string, newDesc string) string {
	if strings.Contains(desc, "retrying") {
		return strings.Replace(desc, "retrying", newDesc, -1)
	}

	return desc + ", " + newDesc
}

func (m *orchestrationManager) waitForStart(o *internal.Orchestration) ([]orchestration.RuntimeOperation, int, error) {
	maintenancePolicy, err := m.getMaintenancePolicy()
	if err != nil {
		m.log.Warnf("while getting maintenance policy: %s", err)
	}

	//polling every 5 min until ochestration start
	pollingInterval := 5 * time.Minute
	var operations, unnotified_operations []orchestration.RuntimeOperation
	var filterRuntimes []orchestration.Runtime
	err = wait.PollImmediateInfinite(pollingInterval, func() (bool, error) {
		//resolve operations, cancel non existent ones
		operations, filterRuntimes, err = m.resolveOperations(o, maintenancePolicy)
		if err != nil {
			return true, err
		}

		//send notification for each operation which doesn't have one
		if o.Parameters.Notification && o.State == orchestration.Pending {
			for _, operation := range operations {
				if operation.NotificationState == "" {
					unnotified_operations = append(unnotified_operations, operation)
				}
			}
			err = m.sendNotificationCreate(o, unnotified_operations)
			//currently notification error can only be temporary error
			if err != nil && kebError.IsTemporaryError(err) {
				return true, err
			}

			//update notification state for notified operations
			for _, operation := range unnotified_operations {
				runtimeID := operation.Runtime.RuntimeID
				err = m.factory.NotifyOperation(o.OrchestrationID, runtimeID, orchestration.Pending, orchestration.NotificationCreated)
				if err != nil {
					return true, fmt.Errorf("while updaring operation for runtime id %q: %w", runtimeID, err)
				}
			}
		}

		//leave polling when ochestration starts
		if time.Now().After(o.Parameters.Strategy.ScheduleTime) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return []orchestration.RuntimeOperation{}, len(filterRuntimes), fmt.Errorf("while waiting for orchestration start: %w", err)
	}
	return operations, len(filterRuntimes), nil
}
