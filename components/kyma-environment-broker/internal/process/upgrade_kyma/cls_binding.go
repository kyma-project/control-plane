package upgrade_kyma

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const (
	kibanaURLLabelKey = "operator_lmsUrl"
)

type ClsUpgradeBindStep struct {
	config           *cls.Config
	operationManager *process.UpgradeKymaOperationManager
	secretKey        string
	bindingProvider  provisioning.ClsBindingProvider
}

func NewClsUpgradeBindStep(config *cls.Config, bp provisioning.ClsBindingProvider, os storage.Operations, secretKey string) *ClsUpgradeBindStep {
	return &ClsUpgradeBindStep{
		config:           config,
		operationManager: process.NewUpgradeKymaOperationManager(os),
		secretKey:        secretKey,
		bindingProvider:  bp,
	}
}

var _ Step = (*ClsUpgradeBindStep)(nil)

func (s *ClsUpgradeBindStep) Name() string {
	return "CLS_UpgradeBind"
}

func (s *ClsUpgradeBindStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
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

	var overrideParams *cls.OverrideParams
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
			failureReason := fmt.Sprintf("Cls instance is in failed state")
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
		overrideParams, err = s.bindingProvider.CreateBinding(smCli, &cls.BindingRequest{
			InstanceKey: operation.Cls.Instance.InstanceKey(),
			BindingID:   operation.Cls.Binding.BindingID,
		})
		if err != nil {
			failureReason := fmt.Sprintf("Cls Binding failed")
			log.Errorf("%s: %s", failureReason, err)
			return s.operationManager.OperationFailed(operation, failureReason)
		}

		encryptedOverrideParams, err := cls.EncryptOverrides(s.secretKey, overrideParams)
		if err != nil {
			failureReason := fmt.Sprintf("encryptClsOverrides() call failed")
			log.Errorf("%s: %s", failureReason, err)
			return s.operationManager.OperationFailed(operation, failureReason)
		}

		operation.Cls.Overrides = encryptedOverrideParams
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
		overrideParams, err = cls.DecryptOverrides(s.secretKey, operation.Cls.Overrides)
		if err != nil {
			failureReason := fmt.Sprintf("decryptClsOverrides() call failed")
			log.Errorf("%s: %s", failureReason, err)
			return s.operationManager.OperationFailed(operation, failureReason)
		}
	}

	operation.InputCreator.SetLabel(kibanaURLLabelKey, overrideParams.KibanaUrl)

	extraConfTemplate, err := cls.GetExtraConfTemplate()
	if err != nil {
		log.Errorf("Unable to get extra config template: %v", err)
		return operation, time.Second, nil
	}

	fluentBitClsOverrides, err := cls.RenderOverrides(overrideParams, extraConfTemplate)
	if err != nil {
		log.Errorf("Unable to render overrides: %v", err)
		return operation, time.Second, nil
	}

	isVersion1_20, err := cls.IsKymaVersionAtLeast_1_20(operation.RuntimeVersion.Version)
	if err != nil {
		failureReason := fmt.Sprintf("unable to check kyma version: %v", err)
		log.Error(failureReason)
		return s.operationManager.OperationFailed(operation, failureReason)
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
