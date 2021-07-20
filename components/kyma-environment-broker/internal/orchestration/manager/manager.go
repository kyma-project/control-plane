package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

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
	ctx                  context.Context
	policyNamespace      string
	policyName           string
}

const maintenancePolicyKeyName = "maintenancePolicy"

func (m *orchestrationManager) Execute(orchestrationID string) (time.Duration, error) {
	logger := m.log.WithField("orchestrationID", orchestrationID)
	m.log.Infof("Processing orchestration %s", orchestrationID)
	o, err := m.orchestrationStorage.GetByID(orchestrationID)
	if err != nil {
		return m.failOrchestration(o, errors.Wrap(err, "while getting orchestration"))
	}

	config := &coreV1.ConfigMap{}
	key := client.ObjectKey{Namespace: m.policyNamespace, Name: m.policyName}
	if err := m.k8sClient.Get(m.ctx, key, config); err != nil {
		m.log.Info("Orchestration Config is absent")
	}
	if config.Data[maintenancePolicyKeyName] == "" {
		m.log.Info("Maintenance policy is set to Gardener defaults")
	}

	var policies []orchestration.MaintenancePolicyEntry
	err = json.Unmarshal([]byte(config.String()), &policies)
	if err != nil {
		m.log.Info("Unable to unmarshal the policies config")
	}
	operations, err := m.resolveOperations(o, policies)
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

func (m *orchestrationManager) resolveOperations(o *internal.Orchestration, policies []orchestration.MaintenancePolicyEntry) ([]orchestration.RuntimeOperation, error) {
	result := []orchestration.RuntimeOperation{}
	if o.State == orchestration.Pending {
		runtimes, err := m.resolver.Resolve(o.Parameters.Targets)
		if err != nil {
			return result, errors.Wrap(err, "while resolving targets")
		}

		for _, r := range runtimes {
			windowBegin := time.Time{}
			windowEnd := time.Time{}

			maintenanceDays := r.MaintenanceDays
			maintenanceWindowBegin := r.MaintenanceWindowBegin
			maintenanceWindowEnd := r.MaintenanceWindowEnd

			for _, p := range policies {
				if p.Match.Plan != "" && p.Match.Plan != r.Plan {
					continue
				}
				if p.Match.GlobalAccountID != "" {
					matched, err := regexp.MatchString(p.Match.GlobalAccountID, r.GlobalAccountID)
					if err != nil || !matched {
						continue
					}
				}

				// We have a rule match here, either be one or all of the rule match options. Let's override maintenance attributes.
				if len(p.Days) > 0 {
					maintenanceDays = p.Days
				}
				if !p.TimeBegin.IsZero() {
					maintenanceWindowBegin = p.TimeBegin
				}
				if !p.TimeEnd.IsZero() {
					maintenanceWindowEnd = p.TimeEnd
				}
			}

			if o.Parameters.Strategy.Schedule == orchestration.MaintenanceWindow {
				windowBegin, windowEnd = m.resolveWindowTime(maintenanceWindowBegin, maintenanceWindowEnd, maintenanceDays)
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

// resolves when is the next occurrence of the time window
func (m *orchestrationManager) resolveWindowTime(beginTime, endTime time.Time, availableDays []time.Weekday) (time.Time, time.Time) {
	n := time.Now()
	start := time.Date(n.Year(), n.Month(), n.Day(), beginTime.Hour(), beginTime.Minute(), beginTime.Second(), beginTime.Nanosecond(), beginTime.Location())
	end := time.Date(n.Year(), n.Month(), n.Day(), endTime.Hour(), endTime.Minute(), endTime.Second(), endTime.Nanosecond(), endTime.Location())

	// if the window end slips through the next day, adjust the date accordingly
	if end.Before(start) {
		end = end.AddDate(0, 0, 1)
	}

	// if time window has already passed we wait until next day
	if start.Before(n) && end.Before(n) {
		currentDay := n.Day()
		nextDay := orchestration.FirstAvailableDay(currentDay, orchestration.ConvertSliceOfDaysToMap(availableDays))
		diff := (7 - currentDay + nextDay) % 7
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
