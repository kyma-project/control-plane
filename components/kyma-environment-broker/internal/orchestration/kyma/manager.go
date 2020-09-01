package kyma

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type upgradeKymaManager struct {
	db                  storage.Orchestrations
	resolver            orchestration.RuntimeResolver
	kymaUpgradeExecutor process.Executor
	log                 logrus.FieldLogger
}

func NewUpgradeKymaManager(db storage.Orchestrations, kymaUpgradeExecutor process.Executor, resolver orchestration.RuntimeResolver, log logrus.FieldLogger) process.Executor {
	return &upgradeKymaManager{
		db:                  db,
		resolver:            resolver,
		kymaUpgradeExecutor: kymaUpgradeExecutor,
		log:                 log,
	}
}

// Execute reconciles runtimes for a given orchestration
func (u *upgradeKymaManager) Execute(orchestrationID string) (time.Duration, error) {
	u.log.Infof("Processing orchestration %s", orchestrationID)
	o, err := u.db.GetByID(orchestrationID)
	if err != nil {
		return u.failOrchestration(o, errors.Wrap(err, "while getting orchestration"))
	}

	dto := orchestration.Parameters{}
	if o.Parameters.Valid {
		err = json.Unmarshal([]byte(o.Parameters.String), &dto)
		if err != nil {
			return u.failOrchestration(o, errors.Wrap(err, "while unmarshalling parameters"))
		}
	}

	targets := dto.Targets
	if targets.Include == nil || len(targets.Include) == 0 {
		targets.Include = []internal.RuntimeTarget{{Target: internal.TargetAll}}
	}

	operations, err := u.resolveOperations(o, dto)
	if err != nil {
		return u.failOrchestration(o, errors.Wrap(err, "while resolving operations"))
	}

	state := internal.InProgress
	desc := fmt.Sprintf("scheduled %d operations", len(operations))

	isFinished := len(operations) == 0 || dto.DryRun
	if isFinished {
		state = internal.Succeeded
	}

	repeat, err := u.updateOrchestration(o, state, desc, operations)
	switch {
	case err != nil:
		return u.failOrchestration(o, errors.Wrap(err, "while updating orchestration"))
	case repeat != 0:
		return repeat, nil
	case isFinished:
		return 0, nil
	}

	// TODO(upgrade): support many strategies
	strategy := orchestration.NewInstantOrchestrationStrategy(u.kymaUpgradeExecutor, u.log)
	_, err = strategy.Execute(operations, dto.Strategy)
	if err != nil {
		return 0, errors.Wrap(err, "while executing instant upgrade strategy")
	}

	state = internal.Succeeded
	// TODO(upgrade): check UpgradeKymaOperations in the loop to assert orchestration state
	result, err := u.checkOperationsResults(operations)
	if err != nil {
		return 0, errors.Wrap(err, "while checking operations results")
	}
	if !result {
		state = internal.Failed
	}
	repeat, err = u.updateOrchestration(o, state, desc, operations)
	if err != nil {
		return 0, errors.Wrap(err, "while updating orchestration")
	}
	u.log.Infof("Finished processing orchestration %s", orchestrationID)

	return repeat, nil
}

func (u *upgradeKymaManager) resolveOperations(o *internal.Orchestration, dto orchestration.Parameters) ([]internal.RuntimeOperation, error) {
	operations := make([]internal.RuntimeOperation, 0)

	if o.State == internal.InProgress {
		if o.RuntimeOperations.Valid {
			err := json.Unmarshal([]byte(o.RuntimeOperations.String), &operations)
			if err != nil {
				return nil, errors.Wrap(err, "while un-marshalling runtime operations")
			}
		}
	} else {
		runtimes, err := u.resolver.Resolve(dto.Targets)
		if err != nil {
			return nil, errors.Wrap(err, "while resolving targets")
		}
		for _, r := range runtimes {
			// TODO(upgrade): Insert UpgradeKymaOperation to DB
			id := uuid.New().String()
			operations = append(operations, internal.RuntimeOperation{
				Runtime:     r,
				OperationID: id,
				Status:      internal.InProgress,
			})
		}
	}

	return operations, nil
}

func (u *upgradeKymaManager) failOrchestration(o *internal.Orchestration, err error) (time.Duration, error) {
	u.log.Errorf("orchestration %s failed: %s", o.OrchestrationID, err)
	return u.updateOrchestration(o, internal.Failed, err.Error(), nil)
}

func (u *upgradeKymaManager) updateOrchestration(o *internal.Orchestration, state, description string, ops []internal.RuntimeOperation) (time.Duration, error) {
	if len(ops) > 0 {
		result, err := json.Marshal(&ops)
		if err != nil {
			return 0, errors.Wrap(err, "while un-marshalling runtime operations")
		}
		o.RuntimeOperations = sql.NullString{
			String: string(result),
			Valid:  true,
		}
	}
	o.State = state
	o.Description = description
	err := u.db.Update(*o)
	if err != nil {
		if !dberr.IsNotFound(err) {
			u.log.Errorf("while updating orchestration: %v", err)
			return time.Minute, nil
		}
	}
	return 0, nil
}

func (u *upgradeKymaManager) checkOperationsResults(ops []internal.RuntimeOperation) (bool, error) {
	return true, nil
}
