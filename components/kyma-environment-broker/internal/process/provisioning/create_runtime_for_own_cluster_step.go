package provisioning

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type CreateRuntimeForOwnCluster struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
}

func NewCreateRuntimeForOwnClusterStep(os storage.Operations, is storage.Instances) *CreateRuntimeForOwnCluster {
	return &CreateRuntimeForOwnCluster{
		operationManager: process.NewOperationManager(os),
		instanceStorage:  is,
	}
}

func (s *CreateRuntimeForOwnCluster) Name() string {
	return "Create_Runtime_For_Own_Cluster"
}

func (s *CreateRuntimeForOwnCluster) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.RuntimeID != "" {
		log.Infof("RuntimeID already set %s, skipping", operation.RuntimeID)
		return operation, 0, nil
	}
	if time.Since(operation.UpdatedAt) > CreateRuntimeTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CreateRuntimeTimeout), nil, log)
	}

	runtimeID := uuid.New().String()

	operation, repeat, _ := s.operationManager.UpdateOperation(operation, func(operation *internal.Operation) {
		operation.ProvisionerOperationID = ""
		operation.RuntimeID = runtimeID
	}, log)
	if repeat != 0 {
		log.Errorf("cannot save neither runtimeID nor empty provider operation ID")
		return operation, 5 * time.Second, nil
	}

	err := s.updateInstance(operation.InstanceID,
		runtimeID)

	switch {
	case err == nil:
	case dberr.IsConflict(err):
		err := s.updateInstance(operation.InstanceID, runtimeID)
		if err != nil {
			log.Errorf("cannot update instance: %s", err)
			return operation, 1 * time.Minute, nil
		}
	}

	log.Info("runtime created for own cluster plan")
	return operation, 0, nil
}

func (s *CreateRuntimeForOwnCluster) updateInstance(id, runtimeID string) error {
	instance, err := s.instanceStorage.GetByID(id)
	if err != nil {
		return errors.Wrap(err, "while getting instance")
	}
	instance.RuntimeID = runtimeID
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		return errors.Wrap(err, "while updating instance")
	}

	return nil
}
