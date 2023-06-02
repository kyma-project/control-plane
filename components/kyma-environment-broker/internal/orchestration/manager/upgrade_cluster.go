package manager

import (
	"fmt"
	"time"

	internalOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

type upgradeClusterFactory struct {
	operationStorage storage.Operations
}

func NewUpgradeClusterManager(orchestrationStorage storage.Orchestrations, operationStorage storage.Operations, instanceStorage storage.Instances,
	kymaClusterExecutor orchestration.OperationExecutor, resolver orchestration.RuntimeResolver, pollingInterval time.Duration,
	log logrus.FieldLogger, cli client.Client, cfg internalOrchestration.Config, bundleBuilder notification.BundleBuilder, speedFactor int) process.Executor {
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
		bundleBuilder:     bundleBuilder,
		speedFactor:       speedFactor,
	}
}

func (u *upgradeClusterFactory) NewOperation(o internal.Orchestration, r orchestration.Runtime, i internal.Instance, state domain.LastOperationState) (orchestration.RuntimeOperation, error) {
	id := uuid.New().String()
	op := internal.UpgradeClusterOperation{
		Operation: internal.Operation{
			ID:                     id,
			Version:                0,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Type:                   internal.OperationTypeUpgradeCluster,
			InstanceID:             r.InstanceID,
			State:                  state,
			Description:            "Operation created",
			OrchestrationID:        o.OrchestrationID,
			ProvisioningParameters: i.Parameters,
			InstanceDetails:        i.InstanceDetails,
			RuntimeOperation: orchestration.RuntimeOperation{
				ID:      id,
				Runtime: r,
				DryRun:  o.Parameters.DryRun,
			},
		},
	}

	err := u.operationStorage.InsertUpgradeClusterOperation(op)
	return op.RuntimeOperation, err
}

func (u *upgradeClusterFactory) ResumeOperations(orchestrationID string) ([]orchestration.RuntimeOperation, error) {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.InProgress, orchestration.Retrying, orchestration.Pending}})
	if err != nil {
		return nil, err
	}

	pending := make([]orchestration.RuntimeOperation, 0)
	retrying := make([]orchestration.RuntimeOperation, 0)
	inProgress := make([]orchestration.RuntimeOperation, 0)
	for _, op := range ops {
		if op.State == orchestration.Pending {
			pending = append(pending, op.RuntimeOperation)
		}
		if op.State == orchestration.Retrying {
			runtimeop, err := u.updateRetryingOperation(op)
			if err != nil {
				return nil, err
			}
			retrying = append(retrying, runtimeop)
		}
		if op.State == orchestration.InProgress {
			inProgress = append(inProgress, op.RuntimeOperation)
		}
	}

	return append(inProgress, append(retrying, pending...)...), nil
}

func (u *upgradeClusterFactory) CancelOperation(orchestrationID string, runtimeID string) error {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Pending}})
	if err != nil {
		return fmt.Errorf("while listing upgrade cluster operations: %w", err)
	}
	for _, op := range ops {
		if op.InstanceDetails.RuntimeID == runtimeID {
			op.State = orchestration.Canceled
			op.Description = "Operation was canceled"
			_, err := u.operationStorage.UpdateUpgradeClusterOperation(op)
			if err != nil {
				return fmt.Errorf("while updating upgrade cluster operation: %w", err)
			}
		}
	}

	return nil
}

func (u *upgradeClusterFactory) CancelOperations(orchestrationID string) error {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Pending}})
	if err != nil {
		return fmt.Errorf("while listing upgrade cluster operations: %w", err)
	}
	for _, op := range ops {
		op.State = orchestration.Canceled
		op.Description = "Operation was canceled"
		_, err := u.operationStorage.UpdateUpgradeClusterOperation(op)
		if err != nil {
			return fmt.Errorf("while updating upgrade cluster operation: %w", err)
		}
	}

	return nil
}

// get current retrying operations
func (u *upgradeClusterFactory) RetryOperations(retryOps []string) ([]orchestration.RuntimeOperation, error) {

	result := []orchestration.RuntimeOperation{}
	for _, opId := range retryOps {
		runtimeop, err := u.operationStorage.GetUpgradeClusterOperationByID(opId)
		if err != nil {
			return nil, fmt.Errorf("while geting (retrying) upgrade cluster operation %s in storage: %w", opId, err)

		}
		result = append(result, runtimeop.RuntimeOperation)
	}

	return result, nil
}

// update storage in corresponding upgrade factory to avoid too many storage read and write
func (u *upgradeClusterFactory) updateRetryingOperation(op internal.UpgradeClusterOperation) (orchestration.RuntimeOperation, error) {
	op.UpdatedAt = time.Now()
	op.State = orchestration.Pending
	op.Description = "Operation retry triggered"
	op.ProvisionerOperationID = ""

	opUpdated, err := u.operationStorage.UpdateUpgradeClusterOperation(op)
	if err != nil {
		return orchestration.RuntimeOperation{}, fmt.Errorf("while updating (retrying) upgrade cluster operation %s in storage: %w", op.Operation.ID, err)
	}

	return opUpdated.RuntimeOperation, nil
}

func (u *upgradeClusterFactory) QueryOperation(orchestrationID string, r orchestration.Runtime) (bool, orchestration.RuntimeOperation, error) {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Pending}})
	if err != nil {
		return false, orchestration.RuntimeOperation{}, fmt.Errorf("while listing upgrade cluster operations: %w", err)
	}
	for _, op := range ops {
		if op.InstanceDetails.RuntimeID == r.RuntimeID {
			return true, op.RuntimeOperation, nil
		}
	}

	return false, orchestration.RuntimeOperation{}, nil
}

func (u *upgradeClusterFactory) QueryOperations(orchestrationID string) ([]orchestration.RuntimeOperation, error) {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Pending}})
	if err != nil {
		return []orchestration.RuntimeOperation{}, fmt.Errorf("while listing upgrade cluster operations: %w", err)
	}
	result := []orchestration.RuntimeOperation{}
	for _, op := range ops {
		result = append(result, op.RuntimeOperation)
	}

	return result, nil
}

func (u *upgradeClusterFactory) NotifyOperation(orchestrationID string, runtimeID string, oState string, notifyState orchestration.NotificationStateType) error {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{oState}})
	if err != nil {
		return fmt.Errorf("while listing upgrade cluster operations: %w", err)
	}
	for _, op := range ops {
		if op.InstanceDetails.RuntimeID == runtimeID {
			op.RuntimeOperation.NotificationState = notifyState
			_, err := u.operationStorage.UpdateUpgradeClusterOperation(op)
			if err != nil {
				return fmt.Errorf("while updating pending upgrade cluster operation %s in storage: %w", op.Operation.ID, err)
			}
		}
	}
	return nil
}
