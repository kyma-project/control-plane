package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=ClsBindingProvider --output=automock --outpkg=automock --case=underscore
type ClsBindingProvider interface {
	CreateBinding(smClient servicemanager.Client, request *cls.BindingRequest) (*cls.OverrideParams, error)
}

type ClsBindStep struct {
	config           *cls.Config
	operationManager *process.ProvisionOperationManager
	secretKey        string
	bindingProvider  ClsBindingProvider
}

func NewClsBindStep(config *cls.Config, bp ClsBindingProvider, os storage.Operations, secretKey string) *ClsBindStep {
	return &ClsBindStep{
		config:           config,
		operationManager: process.NewProvisionOperationManager(os),
		secretKey:        secretKey,
		bindingProvider:  bp,
	}
}

var _ Step = (*ClsBindStep)(nil)

func (s *ClsBindStep) Name() string {
	return "CLS_Bind"
}

func (s *ClsBindStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if !operation.Cls.Instance.Provisioned {
		failureReason := "CLS instance was not provisioned"
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}

	var overrideParams *cls.OverrideParams
	var err error
	if operation.Cls.Overrides == "" {
		smCredentials, err := cls.FindCredentials(s.config.ServiceManager, operation.Cls.Region)
		if err != nil {
			failureReason := fmt.Sprintf("Unable to find credentials for CLS Service Manager in region %s", operation.Cls.Region)
			log.Errorf("%s: %v", failureReason, err)
			return s.operationManager.OperationFailed(operation, failureReason, log)
		}
		smCli := operation.SMClientFactory.ForCredentials(smCredentials)

		if operation.Cls.BindingID == "" {
			op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
				operation.Cls.BindingID = uuid.New().String()
			}, log)
			if retry > 0 {
				log.Errorf("Unable to update operation")
				return operation, time.Second, nil
			}
			operation = op
		}

		overrideParams, err = s.bindingProvider.CreateBinding(smCli, &cls.BindingRequest{
			InstanceKey: operation.Cls.Instance.InstanceKey(),
			BindingID:   operation.Cls.BindingID,
		})
		if err != nil {
			log.Errorf("Unable to create CLS Binding: %v. Retrying", err)
			return operation, 30 * time.Second, nil
		}

		encryptedOverrideParams, err := cls.EncryptOverrides(s.secretKey, overrideParams)
		if err != nil {
			failureReason := "Unable to encrypt CLS overrides"
			log.Errorf("%s: %v", failureReason, err)
			return s.operationManager.OperationFailed(operation, failureReason, log)
		}

		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			operation.Cls.Overrides = encryptedOverrideParams
		}, log)
		if retry > 0 {
			log.Errorf("Unable to update operation")
			return operation, time.Second, nil
		}
		operation = op
	} else {
		overrideParams, err = cls.DecryptOverrides(s.secretKey, operation.Cls.Overrides)
		if err != nil {
			failureReason := "Unable to decrypt CLS overrides"
			log.Errorf("%s: %v", failureReason, err)
			return s.operationManager.OperationFailed(operation, failureReason, log)
		}
	}

	extraConfTemplate, err := cls.GetExtraConfTemplate()
	if err != nil {
		failureReason := "Unable to get CLS extra config template"
		log.Errorf("%s: %v", failureReason, err)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}

	fluentBitClsOverrides, err := cls.RenderOverrides(overrideParams, extraConfTemplate)
	if err != nil {
		failureReason := "Unable to render CLS overrides"
		log.Errorf("%s: %v", failureReason, err)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}

	// TODO: delete this check (isVersionAtLeast1_20) after all SKR clusters are migrated to 1.20!
	isVersion1_20, err := cls.IsKymaVersionAtLeast_1_20(operation.RuntimeVersion.Version)
	if err != nil {
		failureReason := "Unable to check kyma version"
		log.Errorf("%s: %v", failureReason, err)
		return s.operationManager.OperationFailed(operation, failureReason, log)
	}
	if isVersion1_20 {
		operation.InputCreator.AppendOverrides(components.CLS, []*gqlschema.ConfigEntryInput{
			{
				Key:   "fluent-bit.config.outputs.additional",
				Value: fluentBitClsOverrides,
			}})
	}

	return operation, 0, nil
}
