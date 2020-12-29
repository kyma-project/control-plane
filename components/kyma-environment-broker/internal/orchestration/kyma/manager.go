package kyma

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration/strategies"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type upgradeKymaManager struct {
	orchestrationStorage storage.Orchestrations
	operationStorage     storage.Operations
	resolver             orchestration.RuntimeResolver
	kymaUpgradeExecutor  process.Executor
	log                  logrus.FieldLogger
	pollingInterval      time.Duration
}

func NewUpgradeKymaManager(orchestrationStorage storage.Orchestrations, operationStorage storage.Operations,
	kymaUpgradeExecutor process.Executor, resolver orchestration.RuntimeResolver,
	pollingInterval time.Duration, log logrus.FieldLogger) process.Executor {
	return &upgradeKymaManager{
		orchestrationStorage: orchestrationStorage,
		operationStorage:     operationStorage,
		resolver:             resolver,
		kymaUpgradeExecutor:  kymaUpgradeExecutor,
		pollingInterval:      pollingInterval,
		log:                  log,
	}
}

// Execute reconciles runtimes for a given orchestration
func (u *upgradeKymaManager) Execute(orchestrationID string) (time.Duration, error) {
	logger := u.log.WithField("orchestrationID", orchestrationID)
	u.log.Infof("Processing orchestration %s", orchestrationID)
	o, err := u.orchestrationStorage.GetByID(orchestrationID)
	if err != nil {
		return u.failOrchestration(o, errors.Wrap(err, "while getting orchestration"))
	}

	operations, err := u.resolveOperations(o, o.Parameters)
	if err != nil {
		return u.failOrchestration(o, errors.Wrap(err, "while resolving operations"))
	}

	err = u.orchestrationStorage.Update(*o)
	if err != nil {
		logger.Errorf("while updating orchestration: %v", err)
		return u.pollingInterval, nil
	}
	// do not perform any action if the orchestration is finished
	if o.IsFinished() {
		u.log.Infof("Orchestration was already finished, state: %s", o.State)
		return 0, nil
	}

	strategy := u.resolveStrategy(o.Parameters.Strategy.Type, u.kymaUpgradeExecutor, logger)
	execID, err := strategy.Execute(u.filterNotFinishedOperations(operations), o.Parameters.Strategy)
	if err != nil {
		return 0, errors.Wrap(err, "while executing upgrade strategy")
	}

	o, err = u.waitForCompletion(o, strategy, execID, logger)
	if err != nil {
		return 0, errors.Wrap(err, "while waiting for orchestration to finish")
	}

	o.UpdatedAt = time.Now()
	err = u.orchestrationStorage.Update(*o)
	if err != nil {
		logger.Errorf("while updating orchestration: %v", err)
		return u.pollingInterval, nil
	}

	logger.Infof("Finished processing orchestration, state: %s", o.State)
	return 0, nil
}

func (u *upgradeKymaManager) resolveOperations(o *internal.Orchestration, params orchestration.Parameters) ([]internal.UpgradeKymaOperation, error) {
	var result []internal.UpgradeKymaOperation
	if o.State == orchestration.Pending {
		runtimes, err := u.resolver.Resolve(params.Targets)
		if err != nil {
			return result, errors.Wrap(err, "while resolving targets")
		}

		for _, r := range runtimes {
			// we set planID fetched from provisioning parameters
			po, err := u.operationStorage.GetProvisioningOperationByInstanceID(r.InstanceID)
			if err != nil {
				return nil, errors.Wrapf(err, "while getting provisioning operation for instance id %s", r.InstanceID)
			}
			windowBegin := time.Time{}
			windowEnd := time.Time{}
			if params.Strategy.Schedule == orchestration.MaintenanceWindow {
				windowBegin, windowEnd = u.resolveWindowTime(r.MaintenanceWindowBegin, r.MaintenanceWindowEnd)
			}

			id := uuid.New().String()
			op := internal.UpgradeKymaOperation{
				Operation: internal.Operation{
					ID:                     id,
					Version:                0,
					CreatedAt:              time.Now(),
					UpdatedAt:              time.Now(),
					InstanceID:             r.InstanceID,
					State:                  orchestration.Pending,
					Description:            "Operation created",
					OrchestrationID:        o.OrchestrationID,
					ProvisioningParameters: po.ProvisioningParameters,
				},
				RuntimeOperation: orchestration.RuntimeOperation{
					ID: id,
					Runtime: orchestration.Runtime{
						ShootName:              r.ShootName,
						MaintenanceWindowBegin: windowBegin,
						MaintenanceWindowEnd:   windowEnd,
						RuntimeID:              r.RuntimeID,
						GlobalAccountID:        r.GlobalAccountID,
						SubAccountID:           r.SubAccountID,
					},
					DryRun: params.DryRun,
				},
			}
			result = append(result, op)
			err = u.operationStorage.InsertUpgradeKymaOperation(op)
			if err != nil {
				u.log.Errorf("while inserting UpgradeKymaOperation for runtime id %q", r.RuntimeID)
			}
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
		result, _, _, err = u.operationStorage.ListUpgradeKymaOperationsByOrchestrationID(o.OrchestrationID, dbmodel.OperationFilter{States: []string{orchestration.InProgress, orchestration.Pending}})
		if err != nil {
			return result, err
		}
		u.log.Infof("Resuming %d operations for orchestration %s", len(result), o.OrchestrationID)
	}

	return result, nil
}

func (u *upgradeKymaManager) resolveStrategy(sType orchestration.StrategyType, executor process.Executor, log logrus.FieldLogger) orchestration.Strategy {
	switch sType {
	case orchestration.ParallelStrategy:
		return strategies.NewParallelOrchestrationStrategy(executor, log)
	}
	return nil
}

func (u *upgradeKymaManager) filterNotFinishedOperations(ops []internal.UpgradeKymaOperation) []orchestration.RuntimeOperation {
	pending := make([]orchestration.RuntimeOperation, 0)
	inProgress := make([]orchestration.RuntimeOperation, 0)

	for _, op := range ops {
		if op.State == orchestration.Pending {
			pending = append(pending, op.RuntimeOperation)
		}
		if op.State == orchestration.InProgress {
			inProgress = append(inProgress, op.RuntimeOperation)
		}
	}
	return append(inProgress, pending...)
}

// waitForCompletion waits until processing of given orchestration ends or if it's canceled
func (u *upgradeKymaManager) waitForCompletion(o *internal.Orchestration, strategy orchestration.Strategy, execID string, log logrus.FieldLogger) (*internal.Orchestration, error) {
	canceled := false
	var err error
	var stats map[string]int
	err = wait.PollImmediateInfinite(u.pollingInterval, func() (bool, error) {
		// check if orchestration wasn't canceled
		o, err = u.orchestrationStorage.GetByID(o.OrchestrationID)
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
		s, err := u.operationStorage.GetOperationStatsForOrchestration(o.OrchestrationID)
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

	return u.resolveOrchestration(o, strategy, execID, stats)
}
func (u *upgradeKymaManager) resolveOrchestration(o *internal.Orchestration, strategy orchestration.Strategy, execID string, stats map[string]int) (*internal.Orchestration, error) {
	if o.State == orchestration.Canceling {
		err := u.resolveCanceledOperations(o)
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

func (u *upgradeKymaManager) resolveCanceledOperations(o *internal.Orchestration) error {
	ops, _, _, err := u.operationStorage.ListUpgradeKymaOperationsByOrchestrationID(o.OrchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Pending}})
	if err != nil {
		return errors.Wrap(err, "while listing upgrade operations")
	}
	for _, op := range ops {
		op.State = orchestration.Canceled
		op.Description = "Operation was canceled"
		_, err := u.operationStorage.UpdateUpgradeKymaOperation(op)
		if err != nil {
			return errors.Wrap(err, "while updating upgrade kyma operation")
		}
	}
	return nil
}

// resolves when is the next occurrence of the time window
func (u *upgradeKymaManager) resolveWindowTime(beginTime, endTime time.Time) (time.Time, time.Time) {
	n := time.Now()
	start := time.Date(n.Year(), n.Month(), n.Day(), beginTime.Hour(), beginTime.Minute(), beginTime.Second(), beginTime.Nanosecond(), beginTime.Location())
	end := time.Date(n.Year(), n.Month(), n.Day(), endTime.Hour(), endTime.Minute(), endTime.Second(), endTime.Nanosecond(), endTime.Location())

	// if the window end slips through the next day, adjust the date accordingly
	if end.Before(start) {
		end = end.AddDate(0, 0, 1)
	}

	// if time window has already passed we wait until next day
	if start.Before(n) && end.Before(n) {
		start = start.AddDate(0, 0, 1)
		end = end.AddDate(0, 0, 1)
	}

	return start, end
}

func (u *upgradeKymaManager) failOrchestration(o *internal.Orchestration, err error) (time.Duration, error) {
	u.log.Errorf("orchestration %s failed: %s", o.OrchestrationID, err)
	return u.updateOrchestration(o, orchestration.Failed, err.Error()), nil
}

func (u *upgradeKymaManager) updateOrchestration(o *internal.Orchestration, state, description string) time.Duration {
	o.UpdatedAt = time.Now()
	o.State = state
	o.Description = description
	err := u.orchestrationStorage.Update(*o)
	if err != nil {
		if !dberr.IsNotFound(err) {
			u.log.Errorf("while updating orchestration: %v", err)
			return time.Minute
		}
	}
	return 0
}
