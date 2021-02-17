package provisioning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)


//go:generate mockery --name=ClsBindingProvider --output=automock --outpkg=automock --case=underscore
type ClsBindingProvider interface {
	CreateBinding(smClient servicemanager.Client, request *cls.BindingRequest) (*cls.ClsOverrides, error)
}

type ClsBindStep struct {
	config           *cls.Config
	operationManager *process.ProvisionOperationManager
	secretKey        string
	bindingProvider ClsBindingProvider

}

func NewClsBindStep(config *cls.Config, bp ClsBindingProvider, os storage.Operations, secretKey string) *ClsBindStep {
	return &ClsBindStep{
		config:config,
		operationManager: process.NewProvisionOperationManager(os),
		secretKey:        secretKey,
		bindingProvider: bp,
	}
}

var _ Step = (*ClsBindStep)(nil)

func (s *ClsBindStep) Name() string {
	return "CLS_Bind"
}

func (s *ClsBindStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if !operation.Cls.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("cls Provisioning step was not triggered"), log, "")
	}

	skrRegion := operation.ProvisioningParameters.Parameters.Region
	smCli, err := cls.ServiceManagerClient(operation.SMClientFactory, s.config.ServiceManager, skrRegion)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}
	// test if the provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Cls.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("LastInstanceOperation() call failed"))
	}
	log.Infof("Provisioning Cls (instanceID=%s) state: %s", operation.Cls.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Cls provisioning failed: %s", resp.Description))
	case servicemanager.Succeeded:
		operation.Cls.Instance.Provisioned = true
		operation.Cls.Instance.ProvisioningTriggered = false
		log.Info("Cls instance is provisioned.")
	}

	var overrides *cls.ClsOverrides

	if !operation.Cls.Binding.Bound {
		if operation.Cls.Binding.BindingID == "" {
			operation.Cls.Binding.BindingID = uuid.New().String()
		}

	  // Create a binding
		overrides, err = s.bindingProvider.CreateBinding(smCli, &cls.BindingRequest{
			InstanceKey:   operation.Cls.Instance.InstanceKey(),
			BindingID:     operation.Cls.Binding.BindingID,
		})

		if err != nil {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("Cls Binding failed: %s", err))
		}

		encryptedOverrides, err := encryptClsOverrides(s.secretKey, overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("encryptOverrides() call failed"))
		}

		operation.Cls.Overrides = encryptedOverrides
		operation.Cls.Binding.Bound = true

		// save the status
		operation, retry := s.operationManager.UpdateOperation(operation)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
	} else {
		// fetch existing overrides
		overrides, err = decryptClsOverrides(s.secretKey, operation.Cls.Overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("encryptOverrides() call failed"))
		}
	}

	log.Infof("overrides: %v", overrides)

	operation.InputCreator.SetLabel(kibanaURLLabelKey, overrides.KibanaUrl)
	flOverride, err := s.injectOverrides(overrides, log)
	if err != nil {
		log.Errorf("Unable to generate forward plugin to push logs: %v", err)
		return  operation, time.Second, nil
	}
	log.Infof("flOverrides: %v", flOverride)

	operation.InputCreator.AppendOverrides(components.CLS, getClsOverrides(flOverride))

	return operation, 0, nil
}

func encryptClsOverrides(secretKey string, overrides *cls.ClsOverrides) (string, error) {
	ovrs, err := json.Marshal(*overrides)
	if err != nil {
		return "", errors.Wrap(err, "while encoding eventing overrides")
	}
	encrypter := storage.NewEncrypter(secretKey)
	encryptedOverrides, err := encrypter.Encrypt(ovrs)
	if err != nil {
		return "", errors.Wrap(err, "while encrypting eventing overrides")
	}
	return string(encryptedOverrides), nil
}

func decryptClsOverrides(secretKey string, encryptedOverrides string) (*cls.ClsOverrides, error) {
	encrypter := storage.NewEncrypter(secretKey)
	decryptedOverrides, err := encrypter.Decrypt([]byte(encryptedOverrides))
	if err != nil {
		return nil, errors.Wrap(err, "while decrypting eventing overrides")
	}
	clsOverrides := cls.ClsOverrides{}
	if err := json.Unmarshal(decryptedOverrides, &clsOverrides); err != nil {
		return nil, errors.Wrap(err, "while unmarshall eventing overrides")
	}
	return &clsOverrides, nil
}

func (s *ClsBindStep) injectOverrides(overrides *cls.ClsOverrides, log logrus.FieldLogger) (string,error){
	tmpl, err := template.New("test").Parse("    [OUTPUT]\n        Name              http\n        Match             *\n        Host              {{.FluentdEndPoint}}\n        Port              443\n        HTTP_User         {{.FluentdUsername}}\n        HTTP_Passwd       {{.FluentdPassword}}\n        tls               true\n        tls.verify        true\n        tls.debug         1\n        URI               /\n        Format            json")
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
			Key:    "fluent-bit.config.outputs.additional:",
			Value:  flInputsAdditional,
			Secret: ptr.Bool(true),
		},
	}
}

func (s *ClsBindStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}



