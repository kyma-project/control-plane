package provisioning

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

type ConnectivityOverrides struct {
	OauthClientId      string `json:"oauthClientId"`
	OauthClientSecret  string `json:"oauthClientSecret"`
	OauthTokenEndpoint string `json:"oauthTokenEndpoint"`
	PublishUrl         string `json:"publishUrl"`
	BebNamespace       string `json:"bebNamespace"`
	IsBEBEnabled       string `json:"isBEBEnabled"`
}

type ConnBindStep struct {
	operationManager *process.ProvisionOperationManager
	secretKey        string
}

func NewConnBindStep(os storage.Operations, secretKey string) *ConnBindStep {
	return &ConnBindStep{
		operationManager: process.NewProvisionOperationManager(os),
		secretKey:        secretKey,
	}
}

var _ Step = (*ConnBindStep)(nil)

func (s *ConnBindStep) Name() string {
	return "CONN_Bind"
}

func (s *ConnBindStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if !operation.Conn.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("Connectivity Provisioning step was not triggered"), log, "")
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}
	// test if the provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Conn.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("LastInstanceOperation() call failed"))
	}
	log.Infof("Provisioning Connectivity (instanceID=%s) state: %s", operation.Conn.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Connectivity provisioning failed: %s", resp.Description), log)
	}
	// execute binding
	var connectivityOverrides *ConnectivityOverrides
	if !operation.Conn.Instance.Provisioned {
		if operation.Conn.Binding.BindingID == "" {
			operation.Conn.Binding.BindingID = uuid.New().String()
		}
		respBinding, err := smCli.Bind(operation.Conn.Instance.InstanceKey(), operation.Conn.Binding.BindingID, nil, false)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Bind() call failed"))
		}
		// get overrides
		connectivityOverrides, err = GetConnCredentials(respBinding.Binding)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("getCredentials() call failed"))
		}
		encryptedOverrides, err := EncryptConnOverrides(s.secretKey, connectivityOverrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("encryptOverrides() call failed"))
		}

		// save the status
		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			operation.Conn.Overrides = encryptedOverrides
			operation.Conn.Instance.Provisioned = true
			operation.Conn.Instance.ProvisioningTriggered = false
		}, log)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
		operation = op
	} else {
		// get the credentials from encrypted string in operation.Conn.Instance.
		connectivityOverrides, err = DecryptConnOverrides(s.secretKey, operation.Conn.Overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("decryptOverrides() call failed"))
		}
	}

	// append overrides
	operation.InputCreator.AppendOverrides(components.Connectivity, GetConnOverrides(connectivityOverrides))

	return operation, 0, nil
}

func (s *ConnBindStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}

func GetConnCredentials(binding servicemanager.Binding) (*ConnectivityOverrides, error) {
	return nil, nil
}

//func GetEventingCredentials(binding servicemanager.Binding) (*EventingOverrides, error) {
//	evOverrides := EventingOverrides{}
//	credentials := binding.Credentials
//	evOverrides.BebNamespace = credentials["namespace"].(string)
//	evOverrides.IsBEBEnabled = "true"
//	messaging, ok := credentials["messaging"].([]interface{})
//	if !ok {
//		return nil, fmt.Errorf("false type for %s", "messaging")
//	}
//	for i, m := range messaging {
//		m, ok := m.(map[string]interface{})
//		if !ok {
//			return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d]", i))
//		}
//		p, ok := m["protocol"].([]interface{})
//		if !ok {
//			return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d] -> protocol", i))
//		}
//		if p[0] == "httprest" {
//			evOverrides.PublishUrl = m["uri"].(string)
//			oa2, ok := m["oa2"].(map[string]interface{})
//			if !ok {
//				return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d] -> oa2", i))
//			}
//			evOverrides.OauthClientId = oa2["clientid"].(string)
//			evOverrides.OauthClientSecret = oa2["clientsecret"].(string)
//			evOverrides.OauthTokenEndpoint = oa2["tokenendpoint"].(string)
//			break
//		}
//	}
//	return &evOverrides, nil
//}

func GetConnOverrides(cnOverrides *ConnectivityOverrides) []*gqlschema.ConfigEntryInput {
	return nil
}

//func GetEventingOverrides(evOverrides *EventingOverrides) []*gqlschema.ConfigEntryInput {
//	return []*gqlschema.ConfigEntryInput{
//		{
//			Key:    "authentication.oauthClientId",
//			Value:  evOverrides.OauthClientId,
//			Secret: ptr.Bool(true),
//		},
//		{
//			Key:    "authentication.oauthClientSecret",
//			Value:  evOverrides.OauthClientSecret,
//			Secret: ptr.Bool(true),
//		},
//		{
//			Key:    "authentication.oauthTokenEndpoint",
//			Value:  evOverrides.OauthTokenEndpoint,
//			Secret: ptr.Bool(true),
//		},
//		{
//			Key:    "authentication.publishUrl",
//			Value:  evOverrides.PublishUrl,
//			Secret: ptr.Bool(true),
//		},
//		{
//			Key:    "authentication.bebNamespace",
//			Value:  evOverrides.BebNamespace,
//			Secret: ptr.Bool(true),
//		},
//		{
//			Key:    "global.isBEBEnabled",
//			Value:  evOverrides.IsBEBEnabled,
//			Secret: ptr.Bool(false),
//		},
//		{
//			Key:    "global.eventing.backend",
//			Value:  "beb",
//			Secret: ptr.Bool(false),
//		},
//	}
//}

func EncryptConnOverrides(secretKey string, overrides *ConnectivityOverrides) (string, error) {
	ovrs, err := json.Marshal(*overrides)
	if err != nil {
		return "", errors.Wrap(err, "while encoding connectivity overrides")
	}
	encrypter := storage.NewEncrypter(secretKey)
	encryptedOverrides, err := encrypter.Encrypt(ovrs)
	if err != nil {
		return "", errors.Wrap(err, "while encrypting connectivity overrides")
	}
	return string(encryptedOverrides), nil
}

func DecryptConnOverrides(secretKey string, encryptedOverrides string) (*ConnectivityOverrides, error) {
	encrypter := storage.NewEncrypter(secretKey)
	decryptedOverrides, err := encrypter.Decrypt([]byte(encryptedOverrides))
	if err != nil {
		return nil, errors.Wrap(err, "while decrypting connectivity overrides")
	}
	connectivityOverrides := ConnectivityOverrides{}
	if err := json.Unmarshal(decryptedOverrides, &connectivityOverrides); err != nil {
		return nil, errors.Wrap(err, "while unmarshall connectivity overrides")
	}
	return &connectivityOverrides, nil
}
