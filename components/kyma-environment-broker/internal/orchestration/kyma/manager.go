package kyma

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v7/domain"
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

	targets := o.Parameters.Targets
	if targets.Include == nil || len(targets.Include) == 0 {
		targets.Include = []internal.RuntimeTarget{{Target: internal.TargetAll}}
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
		return 0, nil
	}

	// TODO(upgrade): support many strategies
	strategy := orchestration.NewInstantOrchestrationStrategy(u.kymaUpgradeExecutor, logger)
	_, err = strategy.Execute(u.filterOperationsInProgress(operations), o.Parameters.Strategy)
	if err != nil {
		return 0, errors.Wrap(err, "while executing instant upgrade strategy")
	}

	err = u.waitForCompletion(o)
	if err != nil {
		return 0, errors.Wrap(err, "while checking operations results")
	}

	err = u.orchestrationStorage.Update(*o)
	if err != nil {
		logger.Errorf("while updating orchestration: %v", err)
		return u.pollingInterval, nil
	}

	logger.Infof("Finished processing orchestration, state: %s", o.State)
	return 0, nil
}

func (u *upgradeKymaManager) resolveOperations(o *internal.Orchestration, params internal.OrchestrationParameters) ([]internal.UpgradeKymaOperation, error) {
	var result []internal.UpgradeKymaOperation
	if o.State == internal.Pending {
		runtimes, err := u.resolver.Resolve(params.Targets)
		if err != nil {
			return result, errors.Wrap(err, "while resolving targets")
		}

		for _, r := range runtimes {
			id := uuid.New().String()
			op := internal.UpgradeKymaOperation{
				RuntimeOperation: internal.RuntimeOperation{
					Operation: internal.Operation{
						ID:          id,
						Version:     0,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
						InstanceID:  r.InstanceID,
						State:       domain.InProgress,
						Description: "Operation created",
					},
					DryRun:          params.DryRun,
					RuntimeID:       r.RuntimeID,
					GlobalAccountID: r.GlobalAccountID,
					SubAccountID:    r.SubAccountID,
					OrchestrationID: o.OrchestrationID,
				},
			}
			result = append(result, op)
			err := u.operationStorage.InsertUpgradeKymaOperation(op)
			if err != nil {
				u.log.Errorf("while inserting UpgradeKymaOperation for runtime id %q", r.RuntimeID)
			}
		}

		if len(runtimes) != 0 {
			o.State = internal.InProgress
		} else {
			o.State = internal.Succeeded
		}
		o.Description = fmt.Sprintf("Scheduled %d operations", len(runtimes))

	}

	return result, nil
}

func (u *upgradeKymaManager) filterOperationsInProgress(ops []internal.UpgradeKymaOperation) []internal.RuntimeOperation {
	result := make([]internal.RuntimeOperation, 0)

	for _, op := range ops {
		if op.State == domain.InProgress {
			result = append(result, op.RuntimeOperation)
		}
	}

	return result
}

func (u *upgradeKymaManager) failOrchestration(o *internal.Orchestration, err error) (time.Duration, error) {
	u.log.Errorf("orchestration %s failed: %s", o.OrchestrationID, err)
	return u.updateOrchestration(o, internal.Failed, err.Error()), nil
}

func (u *upgradeKymaManager) updateOrchestration(o *internal.Orchestration, state, description string) time.Duration {
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

func (u *upgradeKymaManager) waitForCompletion(o *internal.Orchestration) error {
	// todo: use inter al config
	// todo: remove PollInfinite  and introduce some timeout???
	var stats map[domain.LastOperationState]int
	err := wait.PollInfinite(u.pollingInterval, func() (bool, error) {
		s, err := u.operationStorage.GetOperationStatsForOrchestration(o.OrchestrationID)
		if err != nil {
			u.log.Errorf("while getting operations: %v", err)
			return false, nil
		}
		stats = s

		numberOfInProgress, found := stats[domain.InProgress]
		if !found {
			u.log.Warnf("Orchestration %s operation stats does not contain in progress operations", o.OrchestrationID)
			return false, nil
		}

		return numberOfInProgress == 0, nil
	})
	if err != nil {
		return errors.Wrap(err, "while waiting for scheduled operations to finish")
	}

	orchestrationState := internal.Succeeded
	if stats[domain.Failed] > 0 {
		orchestrationState = internal.Failed
	}

	o.State = orchestrationState

	return nil
}
