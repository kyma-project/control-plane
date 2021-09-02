package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/strategies"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OperationFactory interface {
	NewOperation(o internal.Orchestration, r orchestration.Runtime, i internal.Instance) (orchestration.RuntimeOperation, error)
	ResumeOperations(orchestrationID string) ([]orchestration.RuntimeOperation, error)
	CancelOperations(orchestrationID string) error
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
	cfg                  *broker.KEBConfig
}

const maintenancePolicyKeyName = "maintenancePolicy"
const maintenanceWindowFormat = "150405-0700"

func (m *orchestrationManager) Execute(orchestrationID string) (time.Duration, error) {
	logger := m.log.WithField("orchestrationID", orchestrationID)
	m.log.Infof("Processing orchestration %s", orchestrationID)
	o, err := m.orchestrationStorage.GetByID(orchestrationID)
	if err != nil {
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

	o.Parameters.Kyma.Version = m.cfg.KymaVersion
	o.Parameters.Kubernetes.Version = m.cfg.Provisioner.KubernetesVersion

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
		return 0, errors.Wrap(err, "while executing upgrade strategy")
	}

	o, err = m.waitForCompletion(o, strategy, execID, logger)
	if err != nil {
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
		runtimes, err := m.resolver.Resolve(o.Parameters.Targets)
		if err != nil {
			return result, errors.Wrap(err, "while resolving targets")
		}

		for _, r := range runtimes {
			windowBegin := time.Time{}
			windowEnd := time.Time{}

			if o.Parameters.Strategy.Schedule == orchestration.MaintenanceWindow {
				windowBegin, windowEnd = m.resolveMaintenanceWindowTime(r, policy)
			}
			r.MaintenanceWindowBegin = windowBegin
			r.MaintenanceWindowEnd = windowEnd

			inst, err := m.instanceStorage.GetByID(r.InstanceID)
			if err != nil {
				return nil, errors.Wrapf(err, "while getting instance %s", r.InstanceID)
			}

			op, err := m.factory.NewOperation(*o, r, *inst)
			if err != nil {
				return nil, errors.Wrapf(err, "while creating new operation for runtime id %q", r.RuntimeID)
			}

			result = append(result, op)
		}

		if len(runtimes) != 0 {
			o.State = orchestration.InProgress
		} else {
			o.State = orchestration.Succeeded
		}
		o.Description = fmt.Sprintf("Scheduled %d operations", len(runtimes))
	} else {
		// Resume processing of not finished upgrade operations after restart
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
		return strategies.NewParallelOrchestrationStrategy(executor, log, 24*time.Hour)
	}
	return nil
}

// waitForCompletion waits until processing of given orchestration ends or if it's canceled
func (m *orchestrationManager) waitForCompletion(o *internal.Orchestration, strategy orchestration.Strategy, execID string, log logrus.FieldLogger) (*internal.Orchestration, error) {
	canceled := false
	var err error
	var stats map[string]int
	err = wait.PollImmediateInfinite(m.pollingInterval, func() (bool, error) {
		// check if orchestration wasn't canceled
		o, err = m.orchestrationStorage.GetByID(o.OrchestrationID)
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
func (m *orchestrationManager) resolveMaintenanceWindowTime(r orchestration.Runtime, policy orchestration.MaintenancePolicy) (time.Time, time.Time) {
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

	return start, end
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
