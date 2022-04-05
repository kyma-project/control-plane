package deprovisioning

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceInstanceName      = "uaa-issuer"
	serviceInstanceNamespace = "kyma-system"
	svcatObjectKey           = "serviceinstances.servicecatalog.k8s.io"
	btpOperatorObjectKey     = "serviceinstances.services.cloud.sap.com"
	allowedRetries           = 5
)

type RemoveServiceInstanceStep struct {
	operationManager *process.DeprovisionOperationManager
}

func NewRemoveServiceInstanceStep(os storage.Operations) *RemoveServiceInstanceStep {
	return &RemoveServiceInstanceStep{
		operationManager: process.NewDeprovisionOperationManager(os),
	}
}

func (s *RemoveServiceInstanceStep) Name() string {
	return "Remove_ServiceInstance"
}

func (s *RemoveServiceInstanceStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.IsServiceInstanceDeleted {
		return operation, 0, nil
	}

	if operation.K8sClient == nil {
		log.Errorf("k8s client must be provided")
		return s.operationManager.OperationFailed(operation, "k8s client must be provided", nil, log)
	}

	si, err := s.getServiceInstance(operation.K8sClient)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			operation.IsServiceInstanceDeleted = true
			return operation, 0, nil
		}
		log.Errorf("while getting %s service instance from the cluster", serviceInstanceName)
		return s.operationManager.OperationFailed(operation, "could not get service instance to be deleted", nil, log)
	}

	err = s.deleteServiceInstance(operation.K8sClient, si)

	switch err.(type) {
	case *k8serrors.UnexpectedObjectError:
		log.Errorf("could not delete %s service instance, unknown status: %s", serviceInstanceName, err)
		return s.operationManager.OperationFailed(operation, "could not delete service instance", nil, log)
	case *k8serrors.StatusError:
		operation.Retries++
		log.Warnf("could not delete %s service instance, status: %s", serviceInstanceName, err)
		return s.retryOrFail(&operation, &log)
	case nil:
		return operation, 0, nil
	}

	log.Errorf("could not delete %s service instance, %s", serviceInstanceName, err)
	return s.operationManager.OperationFailed(operation, "could not delete service instance", nil, log)
}

func (s *RemoveServiceInstanceStep) getServiceInstance(k8sClient client.Client) (*unstructured.Unstructured, error) {
	si := &unstructured.Unstructured{}
	si.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "services.cloud.sap.com",
		Version: "v1",
		Kind:    "ServiceInstance",
	})

	err := k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: serviceInstanceNamespace,
		Name:      serviceInstanceName,
	}, si)
	if err == nil {
		return si, nil
	} else if client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	err = k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: serviceInstanceNamespace,
		Name:      serviceInstanceName,
	}, si)
	if err == nil {
		return si, nil
	} else if client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return nil, err
}

func (s *RemoveServiceInstanceStep) deleteServiceInstance(k8sClient client.Client, si *unstructured.Unstructured) error {
	err := k8sClient.Delete(context.Background(), si)
	return err
}

func (s *RemoveServiceInstanceStep) retryOrFail(operation *internal.DeprovisioningOperation, log *logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Retries > allowedRetries {
		(*log).Errorf("could not delete %s service instance, timeout reached", serviceInstanceName)
		return s.operationManager.OperationFailed(*operation, "could not delete service instance", nil, *log)
	}
	return *operation, time.Second * 20, nil
}
