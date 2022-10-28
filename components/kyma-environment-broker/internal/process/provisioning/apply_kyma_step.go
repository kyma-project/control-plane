package provisioning

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kymaResourceNamespace = "kcp-system"
)

type ApplyKymaStep struct {
	operationManager *process.OperationManager
}

var _ process.Step = &ApplyKymaStep{}
var kymaGVK = schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1alpha1", Kind: "Kyma"}

func NewApplyKymaStep(os storage.Operations) *ApplyKymaStep {
	return &ApplyKymaStep{operationManager: process.NewOperationManager(os)}
}

func (a *ApplyKymaStep) Name() string {
	return "Apply_Kyma"
}

func (a *ApplyKymaStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.K8sClient == nil {
		return a.operationManager.OperationFailed(operation, "operation does not contain initialized k8s client", nil, logger)
	}
	k8s := operation.K8sClient

	template, err := a.createUnstructuredKyma(operation)
	if err != nil {
		return a.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)

	}
	a.addLabelsAndName(operation, template)

	var existingKyma unstructured.Unstructured
	existingKyma.SetGroupVersionKind(kymaGVK)
	err = k8s.Get(context.Background(), client.ObjectKey{
		Namespace: template.GetNamespace(),
		Name:      template.GetName(),
	}, &existingKyma)
	switch {
	case err == nil:
		logger.Infof("Kyma resource already exists, updating")
		v, found, err := unstructured.NestedMap(template.Object, "spec")
		if err != nil {
			return a.operationManager.OperationFailed(operation, "unable to get spec from the kyma template", err, logger)
		}
		if !found {
			return a.operationManager.OperationFailed(operation, "unable to find spec in the kyma template", nil, logger)
		}
		unstructured.SetNestedMap(existingKyma.Object, v, "spec")
		a.addLabelsAndName(operation, &existingKyma)
		err = k8s.Update(context.Background(), &existingKyma)
		if err != nil {
			logger.Errorf("unable to update a Kyma resource: %s", err.Error())
			return a.operationManager.RetryOperation(operation, "unable to update the Kyma resource", err, time.Second, 10*time.Second, logger)
		}
	case errors.IsNotFound(err):
		err := k8s.Create(context.Background(), template)
		if err != nil {
			logger.Errorf("unable to create a Kyma resource: %s", err.Error())
			return a.operationManager.RetryOperation(operation, "unable to create the Kyma resource", err, time.Second, 10*time.Second, logger)
		}
	default:
		return a.operationManager.RetryOperation(operation, "unable to get the Kyma resource", err, time.Second, 10*time.Second, logger)
	}

	return operation, 0, nil
}

func (a *ApplyKymaStep) addLabelsAndName(operation internal.Operation, obj *unstructured.Unstructured) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	// todo: define labels
	labels["kyma-project.io/broker-plan-id"] = operation.ProvisioningParameters.PlanID
	labels["kyma-project.io/global-account-id"] = operation.GlobalAccountID
	labels["kyma-project.io/runtime-id"] = operation.RuntimeID
	labels["kyma-project.io/instance-id"] = operation.InstanceID
	// todo: what should we use for Kyma name?
	obj.SetName(strings.ToLower(operation.RuntimeID))
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
