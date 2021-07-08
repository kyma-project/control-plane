package manager

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type upgradeClusterFactory struct {
	operationStorage storage.Operations
}

func NewUpgradeClusterManager(orchestrationStorage storage.Orchestrations, operationStorage storage.Operations, instanceStorage storage.Instances,
	kymaClusterExecutor orchestration.OperationExecutor, resolver orchestration.RuntimeResolver,
	pollingInterval time.Duration, log logrus.FieldLogger, cli client.Client, ctx context.Context, policyNamespace string,
	policyName string) process.Executor {
	return &orchestrationManager{
		orchestrationStorage: orchestrationStorage,
		operationStorage:     operationStorage,
		instanceStorage:      instanceStorage,
		resolver:             resolver,
		factory: &upgradeClusterFactory{
			operationStorage: operationStorage,
		},
		executor:        kymaClusterExecutor,
		pollingInterval: pollingInterval,
		log:             log,
		k8sClient:       cli,
		ctx:             ctx,
		policyNamespace: policyNamespace,
		policyName:      policyName,
	}
}

func (u *upgradeClusterFactory) NewOperation(o internal.Orchestration, r orchestration.Runtime, i internal.Instance) (orchestration.RuntimeOperation, error) {
	id := uuid.New().String()
	op := internal.UpgradeClusterOperation{
		Operation: internal.Operation{
			ID:                     id,
			Version:                0,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Type:                   internal.OperationTypeUpgradeCluster,
			InstanceID:             r.InstanceID,
			State:                  orchestration.Pending,
			Description:            "Operation created",
			OrchestrationID:        o.OrchestrationID,
			ProvisioningParameters: i.Parameters,
			InstanceDetails:        i.InstanceDetails,
		},
		RuntimeOperation: orchestration.RuntimeOperation{
			ID:      id,
			Runtime: r,
			DryRun:  o.Parameters.DryRun,
		},
	}

	err := u.operationStorage.InsertUpgradeClusterOperation(op)
	return op.RuntimeOperation, err
}

func (u *upgradeClusterFactory) ResumeOperations(orchestrationID string) ([]orchestration.RuntimeOperation, error) {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.InProgress, orchestration.Pending}})
	if err != nil {
		return nil, err
	}

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

	return append(inProgress, pending...), nil
}

func (u *upgradeClusterFactory) CancelOperations(orchestrationID string) error {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Pending}})
	if err != nil {
		return errors.Wrap(err, "while listing upgrade operations")
	}
	for _, op := range ops {
		op.State = orchestration.Canceled
		op.Description = "Operation was canceled"
		_, err := u.operationStorage.UpdateUpgradeClusterOperation(op)
		if err != nil {
			return errors.Wrap(err, "while updating upgrade kyma operation")
		}
	}

	return nil
}
