package provisioning

import (
	"bytes"
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
	CreateBinding(smClient servicemanager.Client, request *cls.BindingRequest) (*cls.ClsOverrideParams, error)
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
	if !operation.Cls.Instance.ProvisioningTriggered {
		failureReason := fmt.Sprintf("cls provisioning step was not triggered")
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason)
	}

	smCredentials, err := cls.FindCredentials(s.config.ServiceManager, operation.Cls.Region)
	if err != nil {
		failureReason := fmt.Sprintf("Unable to find credentials for cls service manager in region %s: %s", operation.Cls.Region, err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason)
	}
	smCli := operation.SMClientFactory.ForCredentials(smCredentials)
	var overrides *cls.ClsOverrideParams

	if !operation.Cls.Binding.Bound {
		// test if the provisioning is finished, if not, retry after 10s
		resp, err := smCli.LastInstanceOperation(operation.Cls.Instance.InstanceKey(), "")
		if err != nil {
			log.Errorf("Unable to fetch LastInstanceOperation()")
			return operation, 10 * time.Second, nil
		}
		log.Debug("Provisioning Cls (instanceID=%s) state: %s", operation.Cls.Instance.InstanceID, resp.State)
		switch resp.State {
		case servicemanager.InProgress:
			return operation, 10 * time.Second, nil
		case servicemanager.Failed:
			failureReason := fmt.Sprintf("Cls instance is state failed")
			log.Errorf("%s: %s", failureReason, resp.Description)
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("Cls provisioning failed: %s", resp.Description))
		case servicemanager.Succeeded:
			operation.Cls.Instance.Provisioned = true
			operation.Cls.Instance.ProvisioningTriggered = false
			log.Info("Cls instance is provisioned.")
		}

		if operation.Cls.Binding.BindingID == "" {
			operation.Cls.Binding.BindingID = uuid.New().String()
		}

		// Create a binding
		overrides, err = s.bindingProvider.CreateBinding(smCli, &cls.BindingRequest{
			InstanceKey: operation.Cls.Instance.InstanceKey(),
			BindingID:   operation.Cls.Binding.BindingID,
		})

		if err != nil {
			failureReason := fmt.Sprintf("Cls Binding failed")
			log.Errorf("%s: %s", failureReason, err)
			return s.operationManager.OperationFailed(operation, failureReason)
		}

		encryptedOverrides, err := cls.EncryptOverrides(s.secretKey, overrides)
		if err != nil {
			failureReason := fmt.Sprintf("encryptClsOverrides() call failed")
			log.Errorf("%s: %s", failureReason, err)
			return s.operationManager.OperationFailed(operation, failureReason)
		}

		operation.Cls.Overrides = encryptedOverrides
		operation.Cls.Binding.Bound = true

		// save the status
		op, retry := s.operationManager.UpdateOperation(operation)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
		operation = op
	} else {
		// fetch existing overrides
		overrides, err = cls.DecryptOverrides(s.secretKey, operation.Cls.Overrides)
		if err != nil {
			failureReason := fmt.Sprintf("decryptClsOverrides() call failed")
			log.Errorf("%s: %s", failureReason, err)
			return s.operationManager.OperationFailed(operation, failureReason)
		}
	}

	operation.InputCreator.SetLabel(kibanaURLLabelKey, overrides.KibanaUrl)
	flOverride, err := s.injectOverrides(overrides, log)
	if err != nil {
		log.Errorf("Unable to generate forward plugin to push logs: %v", err)
		return operation, time.Second, nil
	}

	isVersion1_20, err := cls.IsKymaVersionAtLeast_1_20(operation.RuntimeVersion.Version)
	if err != nil {
		failureReason := fmt.Sprintf("unable to check kyma version: %v", err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason)
	}
	if isVersion1_20 {
		operation.InputCreator.AppendOverrides(components.CLS, getClsOverrides(flOverride))
	}

	return operation, 0, nil
}

func (s *ClsBindStep) injectOverrides(overrides *cls.ClsOverrideParams, log logrus.FieldLogger) (string, error) {
	tmpl, err := cls.GetExtraConfTemplate()
	if err != nil {
		log.Errorf("Template error: %v", err)
		return "", err
	}
	var flOutputs bytes.Buffer
	err = tmpl.Execute(&flOutputs, overrides)
	if err != nil {
		log.Errorf("Template error: %v", err)
		return "", err
	}
	return flOutputs.String(), nil
}

func getClsOverrides(flInputsAdditional string) []*gqlschema.ConfigEntryInput {
	return []*gqlschema.ConfigEntryInput{
		{
			Key:   "fluent-bit.config.outputs.additional",
			Value: flInputsAdditional,
		},
	}
}
