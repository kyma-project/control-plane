package deprovisioning

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"

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
	operationManager   *process.OperationManager
	kcpClient          client.Client
	configProvider     input.ConfigurationProvider
	defaultKymaVersion string
}

func NewDeleteKymaResourceStep(operations storage.Operations, kcpClient client.Client, configProvider input.ConfigurationProvider, defaultKymaVersion string) *DeleteKymaResourceStep {
	return &DeleteKymaResourceStep{
		operationManager:   process.NewOperationManager(operations),
		kcpClient:          kcpClient,
		configProvider:     configProvider,
		defaultKymaVersion: defaultKymaVersion,
	}
}

func (step *DeleteKymaResourceStep) Name() string {
	return "Delete_Kyma_Resource"
}

func (step *DeleteKymaResourceStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	// read the KymaTemplate from the config if needed
	if operation.KymaTemplate == "" {
		cfg, err := step.configProvider.ProvideForGivenVersionAndPlan(step.defaultKymaVersion, broker.PlanNamesMapping[operation.Plan])
		if err != nil {
			return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to get config for given version and plan", 5*time.Second, 30*time.Second, logger)
		}
		modifiedOperation, backoff, err := step.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.KymaTemplate = cfg.KymaTemplate
		}, logger)
		if backoff > 0 {
			return operation, backoff, err
		}
		operation = modifiedOperation
	}
	obj, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		return step.operationManager.RetryOperationWithoutFail(operation, step.Name(), "unable to decode kyma template", 5*time.Second, 30*time.Second, logger)
	}

	if operation.KymaResourceNamespace == "" {
		logger.Warnf("namespace for Kyma resource not specified")
		return operation, 0, nil
	}
	kymaResourceName := steps.KymaName(operation)
	if kymaResourceName == "" {
		logger.Infof("Kyma resource name is empty, skipping")
		return operation, 0, nil
	}

	logger.Infof("Deleting Kyma resource: %s in namespace:%s", kymaResourceName, operation.KymaResourceNamespace)

	kymaUnstructured := &unstructured.Unstructured{}
	kymaUnstructured.SetName(kymaResourceName)
	kymaUnstructured.SetNamespace(operation.KymaResourceNamespace)
	kymaUnstructured.SetGroupVersionKind(obj.GroupVersionKind())

	err = step.kcpClient.Delete(context.Background(), kymaUnstructured)
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
