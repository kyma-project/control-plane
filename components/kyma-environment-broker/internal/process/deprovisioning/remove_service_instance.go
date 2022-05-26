package deprovisioning

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8serrors2 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceInstanceName      = "uaa-issuer"
	serviceInstanceNamespace = "kyma-system"
	k8sResourceType          = "ServiceInstance"
	svcatGroup               = "servicecatalog.k8s.io"
	svcatApiVer              = "v1beta1"
	btpOperatorGroup         = "services.cloud.sap.com"
	btpOperatorApiVer        = "v1"
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
		log.Errorf("k8s client is not provided, service instance %s may need manual removal", serviceInstanceName)
		return operation, 0, nil
	}

	si, err := s.getServiceInstance(operation.K8sClient)
	if err != nil {
		if k8serrors.IsNotFound(err) || k8serrors2.IsNoMatchError(err) {
			operation.IsServiceInstanceDeleted = true
			log.Infof("%s Service Instance is not present in the cluster, skipping step", serviceInstanceName)
			return operation, 0, nil
		}
		log.Errorf("while getting %s service instance from the cluster: %s", serviceInstanceName, err)
		return s.operationManager.OperationFailed(operation, "could not get service instance to be deleted", err, log)
	}

	err = s.deleteServiceInstance(operation.K8sClient, si)

	switch err.(type) {
	case *k8serrors.UnexpectedObjectError:
		log.Errorf("could not delete %s service instance, unknown status: %s", serviceInstanceName, err)
		return s.operationManager.OperationFailed(operation, "could not delete service instance", err, log)
	case *k8serrors.StatusError:
		operation.Retries++
		log.Warnf("could not delete %s service instance, status: %s", serviceInstanceName, err)
		return s.retryOrFail(&operation, err, &log)
	case nil:
		operation.IsServiceInstanceDeleted = true
		return operation, 0, nil
	}

	log.Errorf("could not delete %s service instance, %s", serviceInstanceName, err)
	return s.operationManager.OperationFailed(operation, "could not delete service instance", err, log)
}

func (s *RemoveServiceInstanceStep) getServiceInstance(k8sClient client.Client) (*unstructured.Unstructured, error) {
	si := &unstructured.Unstructured{}
	si.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   svcatGroup,
		Version: svcatApiVer,
		Kind:    k8sResourceType,
	})

	err := k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: serviceInstanceNamespace,
		Name:      serviceInstanceName,
	}, si)
	if err == nil {
		return si, nil
	} else if client.IgnoreNotFound(err) != nil && !k8serrors2.IsNoMatchError(err) {
		return nil, err
	}

	si.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   btpOperatorGroup,
		Version: btpOperatorApiVer,
		Kind:    k8sResourceType,
	})

	err = k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: serviceInstanceNamespace,
		Name:      serviceInstanceName,
	}, si)
	if err == nil {
		return si, nil
	} else if client.IgnoreNotFound(err) != nil && !k8serrors2.IsNoMatchError(err) {
		return nil, err
	}

	return nil, err
}

func (s *RemoveServiceInstanceStep) deleteServiceInstance(k8sClient client.Client, si *unstructured.Unstructured) error {
	err := k8sClient.Delete(context.Background(), si)
	return err
}

func (s *RemoveServiceInstanceStep) retryOrFail(operation *internal.DeprovisioningOperation, err error, log *logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	if operation.Retries > allowedRetries {
		(*log).Errorf("could not delete %s service instance, timeout reached", serviceInstanceName)
		return s.operationManager.OperationFailed(*operation, "could not delete service instance", err, *log)
	}
	return *operation, time.Second * 20, nil
}
