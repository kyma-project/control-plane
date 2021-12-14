package manager

import (
	"fmt"
	"time"

	internalOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"

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
	kymaClusterExecutor orchestration.OperationExecutor, resolver orchestration.RuntimeResolver, pollingInterval time.Duration,
	log logrus.FieldLogger, cli client.Client, cfg internalOrchestration.Config) process.Executor {
	return &orchestrationManager{
		orchestrationStorage: orchestrationStorage,
		operationStorage:     operationStorage,
		instanceStorage:      instanceStorage,
		resolver:             resolver,
		factory: &upgradeClusterFactory{
			operationStorage: operationStorage,
		},
		executor:          kymaClusterExecutor,
		pollingInterval:   pollingInterval,
		log:               log,
		k8sClient:         cli,
		configNamespace:   cfg.Namespace,
		configName:        cfg.Name,
		kymaVersion:       cfg.KymaVersion,
		kubernetesVersion: cfg.KubernetesVersion,
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

func (u *upgradeClusterFactory) ResumeOperations(orchestrationID string, states []string) ([]orchestration.RuntimeOperation, error) {
	result := []orchestration.RuntimeOperation{}

	ops, _, len, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: states})
	if err != nil {
		return nil, err
	}

	fmt.Printf("---------resume------ %+v\n", len)

	for _, op := range ops {
		for _, state := range states {
			if string(op.State) == state {
				result = append(result, op.RuntimeOperation)
				break
			}
		}
	}

	return result, nil
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
			return errors.Wrap(err, "while updating upgrade cluster operation")
		}
	}

	return nil
}

func (u *upgradeClusterFactory) UpdateRetryingOperations(rt orchestration.RuntimeOperation) (orchestration.RuntimeOperation, error) {
	fmt.Println("------------in UpdateRetryingOperations----------")
	op, err := u.operationStorage.GetUpgradeClusterOperationByID(rt.ID)
	if err != nil {
		return orchestration.RuntimeOperation{}, err
	}

	op.MaintenanceWindowBegin = rt.MaintenanceWindowBegin
	op.MaintenanceWindowEnd = rt.MaintenanceWindowEnd
	op.MaintenanceDays = rt.MaintenanceDays
	op.UpdatedAt = time.Now()
	op.Description = "Operation retry triggered"
	op.State = orchestration.Pending

	opUpdated, err := u.operationStorage.UpdateUpgradeClusterOperation(*op)
	if err != nil {
		return orchestration.RuntimeOperation{}, errors.Wrapf(err, "while updating (retrying) operation %s in storage", rt.ID)
	}

	return opUpdated.RuntimeOperation, nil
}

func (u *upgradeClusterFactory) ConvertRetryingToPendingOperations(orchestrationID string) error {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Retrying}})
	if err != nil {
		return errors.Wrap(err, "while listing retrying operations")
	}

	fmt.Printf("---------ConvertRetryingToPendingOperations------ %+v\n", ops)

	for _, op := range ops {
		op.State = orchestration.Pending
		op.Description = "Operation retry triggered"
		_, err := u.operationStorage.UpdateUpgradeClusterOperation(op)
		if err != nil {
			return errors.Wrap(err, "while updating upgrade cluster operation")
		}
	}

	return nil
}
