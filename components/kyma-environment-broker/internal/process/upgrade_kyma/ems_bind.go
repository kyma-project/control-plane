package upgrade_kyma

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

// TODO common code
type EventingOverrides struct {
	OauthClientId      string `json:"oauthClientId"`
	OauthClientSecret  string `json:"oauthClientSecret"`
	OauthTokenEndpoint string `json:"oauthTokenEndpoint"`
	PublishUrl         string `json:"publishUrl"`
	BebNamespace       string `json:"bebNamespace"`
}

type EmsUpgradeBindStep struct {
	operationManager *process.UpgradeKymaOperationManager
	secretKey        string
}

func NewEmsUpgradeBindStep(os storage.Operations, secretKey string) *EmsUpgradeBindStep {
	return &EmsUpgradeBindStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
		secretKey:        secretKey,
	}
}

var _ Step = (*EmsUpgradeBindStep)(nil)

func (s *EmsUpgradeBindStep) Name() string {
	return "EMS_UpgradeBind"
}

func (s *EmsUpgradeBindStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if operation.Ems.BindingID != "" {
		log.Infof("Ems Upgrade-Bind was already done")
		return operation, 0, nil
	}
	if !operation.Ems.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("Ems Provisioning step was not triggered"), log, "")
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}
	// test if the provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Ems.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("LastInstanceOperation() call failed"))
	}
	log.Infof("Provisioning Ems (instanceID=%s) state: %s", operation.Ems.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Ems provisioning failed: %s", resp.Description))
	}
	// execute binding
	var eventingOverrides *EventingOverrides
	if !operation.Ems.Instance.Provisioned {
		if operation.Ems.BindingID == "" {
			operation.Ems.BindingID = uuid.New().String()
		}
		respBinding, err := smCli.Bind(operation.Ems.Instance.InstanceKey(), operation.Ems.BindingID, nil, false)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Bind() call failed"))
		}
		// get overrides
		eventingOverrides, err = getCredentials(respBinding.Binding)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("getCredentials() call failed"))
		}
		encryptedOverrides, err := encryptOverrides(s.secretKey, eventingOverrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("encryptOverrides() call failed"))
		}
		operation.Ems.Overrides = encryptedOverrides
		operation.Ems.Instance.Provisioned = true
		operation.Ems.Instance.ProvisioningTriggered = false
		// save the status
		operation, retry := s.operationManager.UpdateOperation(operation)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
	} else {
		// get the credentials from encrypted string in operation.Ems.Instance.
		eventingOverrides, err = decryptOverrides(s.secretKey, operation.Ems.Overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("decryptOverrides() call failed"))
		}
	}

	// append overrides
	operation.InputCreator.AppendOverrides(components.Eventing, getEventingOverrides(eventingOverrides))

	return operation, 0, nil
}

func (s *EmsUpgradeBindStep) handleError(operation internal.UpgradeKymaOperation, err error, log logrus.FieldLogger, msg string) (internal.UpgradeKymaOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}

func getCredentials(binding servicemanager.Binding) (*EventingOverrides, error) {
	evOverrides := EventingOverrides{}
	credentials := binding.Credentials
	evOverrides.BebNamespace = credentials["namespace"].(string)
	messaging, ok := credentials["messaging"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("false type for %s", "messaging")
	}
	for i, m := range messaging {
		m, ok := m.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d]", i))
		}
		p, ok := m["protocol"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d] -> protocol", i))
		}
		if p[0] == "httprest" {
			evOverrides.PublishUrl = m["uri"].(string)
			oa2, ok := m["oa2"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d] -> oa2", i))
			}
			evOverrides.OauthClientId = oa2["clientid"].(string)
			evOverrides.OauthClientSecret = oa2["clientsecret"].(string)
			evOverrides.OauthTokenEndpoint = oa2["tokenendpoint"].(string)
			break
		}
	}
	return &evOverrides, nil
}

func getEventingOverrides(evOverrides *EventingOverrides) []*gqlschema.ConfigEntryInput {
	return []*gqlschema.ConfigEntryInput{
		{
			Key:    "authentication.oauthClientId",
			Value:  evOverrides.OauthClientId,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "authentication.oauthClientSecret",
			Value:  evOverrides.OauthClientSecret,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "authentication.oauthTokenEndpoint",
			Value:  evOverrides.OauthTokenEndpoint,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "authentication.publishUrl",
			Value:  evOverrides.PublishUrl,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "authentication.bebNamespace",
			Value:  evOverrides.BebNamespace,
			Secret: ptr.Bool(true),
		},
	}
}

func encryptOverrides(secretKey string, overrides *EventingOverrides) (string, error) {
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

func decryptOverrides(secretKey string, encryptedOverrides string) (*EventingOverrides, error) {
	encrypter := storage.NewEncrypter(secretKey)
	decryptedOverrides, err := encrypter.Decrypt([]byte(encryptedOverrides))
	if err != nil {
		return nil, errors.Wrap(err, "while decrypting eventing overrides")
	}
	eventingOverrides := EventingOverrides{}
	if err := json.Unmarshal(decryptedOverrides, &eventingOverrides); err != nil {
		return nil, errors.Wrap(err, "while unmarshall eventing overrides")
	}
	return &eventingOverrides, nil
}
