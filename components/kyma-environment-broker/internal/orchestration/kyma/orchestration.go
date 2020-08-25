package kyma

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type upgradeKymaOrchestration struct {
	db                  storage.Orchestration
	resolver            orchestration.RuntimeResolver
	kymaUpgradeExecutor process.Executor
	log                 logrus.FieldLogger
}

func NewUpgradeKymaOrchestration(db storage.Orchestration, kymaUpgradeExecutor process.Executor, resolver orchestration.RuntimeResolver, log logrus.FieldLogger) process.Executor {
	return &upgradeKymaOrchestration{
		db:                  db,
		resolver:            resolver,
		kymaUpgradeExecutor: kymaUpgradeExecutor,
		log:                 log,
	}
}

// Execute reconciles runtimes for a given orchestration
func (u *upgradeKymaOrchestration) Execute(orchestrationID string) (time.Duration, error) {
	o, err := u.db.GetOrchestrationByID(orchestrationID)
	if err != nil {
		return 0, errors.Wrapf(err, "while getting orchestration %s", orchestrationID)
	}

	dto := orchestration.UpgradeOrchestrationDTO{}
	if o.Parameters.Valid {
		err = json.Unmarshal([]byte(o.Parameters.String), &dto)
		if err != nil {
			return 0, errors.Wrap(err, "while unmarshalling parameters")
		}
	}

	targets := dto.Targets
	if targets.Include == nil || len(targets.Include) == 0 {
		targets.Include = []internal.RuntimeTarget{{Target: internal.TargetAll}}
	}
	operations := make([]internal.RuntimeOperation, 0)

	if o.State == internal.InProgress {
		if o.RuntimeOperations.Valid {
			err = json.Unmarshal([]byte(o.RuntimeOperations.String), &operations)
			if err != nil {
				return 0, errors.Wrap(err, "while un-marshalling runtime operations")
			}
		}
	} else {
		runtimes, err := u.resolver.Resolve(dto.Targets)
		if err != nil {
			return 0, errors.Wrap(err, "while resolving targets")
		}
		for range runtimes {
			// TODO(upgrade): Insert UpgradeKymaOperation to DB; write unit test for o.State cases
			id := uuid.New().String()
			operations = append(operations, internal.RuntimeOperation{OperationID: id})
		}
	}

	strategy := orchestration.NewInstantOrchestrationStrategy(process.NewQueue(u.kymaUpgradeExecutor, u.log), u.log)
	_, err = strategy.Execute(operations, dto.Strategy)
	if err != nil {
		return 0, errors.Wrap(err, "while executing instant upgrade strategy")
	}

	return 0, nil
}
