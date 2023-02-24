package deprovisioning

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/steps"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

const (
	backoffForK8SOperation = time.Second
	timeoutForK8sOperation = 10 * time.Second
)

type DeleteKymaResourceStep struct {
	operationManager *process.OperationManager
	kcpClient        client.Client
}

func NewDeleteKymaResourceStep(operations storage.Operations, kcpClient client.Client) *DeleteKymaResourceStep {
	return &DeleteKymaResourceStep{
		operationManager: process.NewOperationManager(operations),
		kcpClient:        kcpClient,
	}
}

func (step *DeleteKymaResourceStep) Name() string {
	return "Delete_Kyma_Resource"
}

func (step *DeleteKymaResourceStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.KymaResourceNamespace == "" {
		logger.Warnf("namespace for Kyma resource not specified")
		return operation, 0, nil
	}
	kymaResourceName := steps.KymaName(operation)
	if kymaResourceName == "" {
		logger.Infof("Kyma resource name is empty, skipping")
		return operation, 0, nil
	}

	kymaUnstructured := &unstructured.Unstructured{}
	kymaUnstructured.SetName(kymaResourceName)
	kymaUnstructured.SetNamespace(operation.KymaResourceNamespace)
	kymaUnstructured.SetGroupVersionKind(steps.KymaResourceGroupVersionKind())

	err := step.kcpClient.Delete(context.Background(), kymaUnstructured)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("no Kyma resource to delete - ignoring")
		} else {
			logger.Errorf("unable to delete the Kyma resource: %s", err)
			return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to delete the Kyma resource", backoffForK8SOperation, timeoutForK8sOperation, logger)
		}
	}

	return operation, 0, nil
}
