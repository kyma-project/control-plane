package provisioning

import (
	"bytes"
	"context"
	"reflect"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/steps"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplyKymaStep struct {
	operationManager *process.OperationManager
	k8sClient        client.Client
}

var _ process.Step = &ApplyKymaStep{}

func NewApplyKymaStep(os storage.Operations, cli client.Client) *ApplyKymaStep {
	return &ApplyKymaStep{operationManager: process.NewOperationManager(os), k8sClient: cli}
}

func (a *ApplyKymaStep) Name() string {
	return "Apply_Kyma"
}

func (a *ApplyKymaStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	template, err := steps.DecodeKymaTemplate(operation.KymaTemplate)
	if err != nil {
		return a.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	a.addLabelsAndName(operation, template)
	operation, backoff, _ := a.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceName = template.GetName()
	}, logger)
	if backoff != 0 {
		logger.Errorf("cannot save the operation")
		return operation, 5 * time.Second, nil
	}

	var existingKyma unstructured.Unstructured
	existingKyma.SetGroupVersionKind(steps.KymaResourceGroupVersionKind())
	err = a.k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: operation.KymaResourceNamespace,
		Name:      template.GetName(),
	}, &existingKyma)
	switch {
	case err == nil:
		logger.Infof("Kyma resource already exists, updating Kyma resource: %s in namespace %s", existingKyma.GetName(), existingKyma.GetNamespace())
		changed := a.addLabelsAndName(operation, &existingKyma)
		if !changed {
			logger.Infof("Kyma resource does not need any change")
		}
		err = a.k8sClient.Update(context.Background(), &existingKyma)
		if err != nil {
			logger.Errorf("unable to update a Kyma resource: %s", err.Error())
			return a.operationManager.RetryOperation(operation, "unable to update the Kyma resource", err, time.Second, 10*time.Second, logger)
		}
	case errors.IsNotFound(err):
		logger.Infof("creating Kyma resource: % in namespace: %s", template.GetName(), template.GetNamespace())
		err := a.k8sClient.Create(context.Background(), template)
		if err != nil {
			logger.Errorf("unable to create a Kyma resource: %s", err.Error())
			return a.operationManager.RetryOperation(operation, "unable to create the Kyma resource", err, time.Second, 10*time.Second, logger)
		}
	default:
		logger.Errorf("Unable to get Kyma: %s", err.Error())
		return a.operationManager.RetryOperation(operation, "unable to get the Kyma resource", err, time.Second, 10*time.Second, logger)
	}

	return operation, 0, nil
}

func (a *ApplyKymaStep) addLabelsAndName(operation internal.Operation, obj *unstructured.Unstructured) bool {
	oldLabels := obj.GetLabels()
	steps.ApplyLabelsAndAnnotationsForLM(obj, operation)
	obj.SetName(steps.KymaName(operation))
	return !reflect.DeepEqual(obj.GetLabels(), oldLabels)
}

func (a *ApplyKymaStep) createUnstructuredKyma(operation internal.Operation) (*unstructured.Unstructured, error) {
	tmpl := a.kymaTemplate(operation)

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(tmpl), 512)
	var rawObj runtime.RawExtension
	if err := decoder.Decode(&rawObj); err != nil {
		return nil, err
	}
	obj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return nil, err
	}

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
	return unstructuredObj, nil
}

func (a *ApplyKymaStep) kymaTemplate(operation internal.Operation) []byte {
	return []byte(operation.KymaTemplate)
}
