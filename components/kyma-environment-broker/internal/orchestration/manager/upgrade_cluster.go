package manager

import (
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

func (u *upgradeClusterFactory) CancelOperations(orchestrationID string) error {
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Pending}})
	if err != nil {
		return errors.Wrap(err, "while listing upgrade cluster operations")
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

// get current retrying operations, update state to pending and update other required params to storage
func (u *upgradeClusterFactory) RetryOperations(orchestrationID string, schedule orchestration.ScheduleType, policy orchestration.MaintenancePolicy, updateMWindow bool) ([]orchestration.RuntimeOperation, error) {
	result := []orchestration.RuntimeOperation{}
	ops, _, _, err := u.operationStorage.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{States: []string{orchestration.Retrying}})
	if err != nil {
		return nil, errors.Wrap(err, "while listing retrying operations")
	}

	for _, op := range ops {
		if updateMWindow {
			windowBegin := time.Time{}
			windowEnd := time.Time{}
			days := []string{}

			// use the latest policy
			if schedule == orchestration.MaintenanceWindow {
				windowBegin, windowEnd, days = resolveMaintenanceWindowTime(op.RuntimeOperation.Runtime, policy)
			}
			op.MaintenanceWindowBegin = windowBegin
			op.MaintenanceWindowEnd = windowEnd
			op.MaintenanceDays = days
		}

		runtimeop, err := u.updateRetryingOperation(op)
		if err != nil {
			return nil, err
		}

		result = append(result, runtimeop)
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
		return orchestration.RuntimeOperation{}, errors.Wrapf(err, "while updating (retrying) upgrade cluster operation %s in storage", op.Operation.ID)
	}

	return opUpdated.RuntimeOperation, nil
}
