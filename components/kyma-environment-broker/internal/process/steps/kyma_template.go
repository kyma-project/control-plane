package steps

import (
	"bytes"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

type InitKymaTemplate struct {
	operationManager *process.OperationManager
}

var _ process.Step = &InitKymaTemplate{}

func NewInitKymaTemplate(os storage.Operations) *InitKymaTemplate {
	return &InitKymaTemplate{operationManager: process.NewOperationManager(os)}
}

func (s *InitKymaTemplate) Name() string {
	return "Init_Kyma_Template"
}

func (s *InitKymaTemplate) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	tmpl := operation.InputCreator.Configuration().KymaTemplate
	obj, err := DecodeKymaTemplate(tmpl)
	if err != nil {
		logger.Errorf("Unable to create kyma template: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	logger.Infof("Decoded kyma template: %v", obj)
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = obj.GetNamespace()
		op.KymaTemplate = tmpl
	}, logger)
}

// NOTE: adapter for upgrade_kyma which is currently not using shared staged_manager
type initKymaTemplateUpgradeKyma struct {
	*InitKymaTemplate
}

func InitKymaTemplateUpgradeKyma(os storage.Operations) initKymaTemplateUpgradeKyma {
	return initKymaTemplateUpgradeKyma{NewInitKymaTemplate(os)}
}

func (s initKymaTemplateUpgradeKyma) Run(o internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation, w, err := s.InitKymaTemplate.Run(o.Operation, logger)
	return internal.UpgradeKymaOperation{operation}, w, err
}

func DecodeKymaTemplate(template string) (*unstructured.Unstructured, error) {
	tmpl := []byte(template)

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
	return unstructuredObj, err
}
