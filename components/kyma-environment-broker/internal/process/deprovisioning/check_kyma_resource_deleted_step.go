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

type CheckKymaResourceDeletedStep struct {
	operationManager *process.OperationManager
	kcpClient        client.Client
}

func NewCheckKymaResourceDeletedStep(operations storage.Operations, kcpClient client.Client) *CheckKymaResourceDeletedStep {
	return &CheckKymaResourceDeletedStep{
		operationManager: process.NewOperationManager(operations),
		kcpClient:        kcpClient,
	}
}

func (step *CheckKymaResourceDeletedStep) Name() string {
	return "Check_Kyma_Resource_Deleted"
}

func (step *CheckKymaResourceDeletedStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.KymaResourceNamespace == "" {
		logger.Warnf("namespace for Kyma resource not specified")
		return operation, 0, nil
	}

	kymaUnstructured := &unstructured.Unstructured{}
	kymaUnstructured.SetGroupVersionKind(steps.KymaResourceGroupVersionKind())
	err := step.kcpClient.Get(context.Background(), client.ObjectKey{
		Namespace: operation.KymaResourceNamespace,
		Name:      steps.KymaName(operation),
	}, kymaUnstructured)

	if err == nil {
		logger.Error("Kyma resource still exists")
		return step.operationManager.RetryOperationWithoutFail(operation, "Kyma resource still exists", backoffForK8SOperation, timeoutForK8sOperation, logger)
	}

	if !errors.IsNotFound(err) {
		logger.Error("unable to check Kyma resource existence")
		return step.operationManager.RetryOperationWithoutFail(operation, "unable to check Kyma resource existence", backoffForK8SOperation, timeoutForK8sOperation, logger)
	}

	return operation, 0, nil
}
