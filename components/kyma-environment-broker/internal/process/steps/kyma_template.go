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

type InitKymaTempalate struct {
	operationManager *process.OperationManager
}

var _ process.Step = &InitKymaTempalate{}

func NewInitKymaTempalate(os storage.Operations) *InitKymaTempalate {
	return &InitKymaTempalate{operationManager: process.NewOperationManager(os)}
}

func (s *InitKymaTempalate) Name() string {
	return "Init_Kyma_Template"
}

func (s *InitKymaTempalate) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	obj, tmpl, err := s.createUnstructuredKyma(operation)
	if err != nil {
		logger.Errorf("Unable to create kyma template: %s", err.Error())
		return s.operationManager.OperationFailed(operation, "unable to create a kyma template", err, logger)
	}
	logger.Infof("Decoded kyma template: %v", obj)
	return s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.KymaResourceNamespace = obj.GetNamespace()
		op.KymaTemplate = string(tmpl)
	}, logger)
}

func (s *InitKymaTempalate) createUnstructuredKyma(operation internal.Operation) (*unstructured.Unstructured, []byte, error) {
	tmpl := []byte(operation.InputCreator.Configuration().KymaTemplate)

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(tmpl), 512)
	var rawObj runtime.RawExtension
	if err := decoder.Decode(&rawObj); err != nil {
		return nil, []byte{}, err
	}
	obj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return nil, []byte{}, err
	}

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
	return unstructuredObj, tmpl, nil
}

func (s *InitKymaTempalate) kymaTemplate(operation internal.Operation) []byte {
	return []byte(operation.KymaTemplate)
}

// NOTE: adapter for upgrade_kyma which is currently not using shared staged_manager
type initKymaTempalateUpgradeKyma struct {
	*InitKymaTempalate
}

func InitKymaTempalateUpgradeKyma(os storage.Operations) initKymaTempalateUpgradeKyma {
	return initKymaTempalateUpgradeKyma{NewInitKymaTempalate(os)}
}

func (s initKymaTempalateUpgradeKyma) Run(o internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	operation, w, err := s.InitKymaTempalate.Run(o.Operation, logger)
	return internal.UpgradeKymaOperation{operation}, w, err
}
