package provisioning

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ConnectivityConfig struct {
	ClientId            string `json:"clientid"`
	ClientSecret        string `json:"clientsecret"`
	ConnectivityService struct {
		CAsPath        string `json:"CAs_path"`
		CAsSigningPath string `json:"CAs_signing_path"`
		ApiPath        string `json:"api_path"`
		TunnelPath     string `json:"tunnel_path"`
		Url            string `json:"url"`
	} `json:"connectivity_service"`
	SubaccountId                    string `json:"subaccount_id"`
	SubaccountSubdomain             string `json:"subaccount_subdomain"`
	TokenServiceDomain              string `json:"token_service_domain"`
	TokenServiceUrl                 string `json:"token_service_url"`
	TokenServiceUrlPattern          string `json:"token_service_url_pattern"`
	TokenServiceUrlPatternTenantKey string `json:"token_service_url_pattern_tenant_key"`
	Xsappname                       string `json:"xsappname"`
}

type ConnectivityBindStep struct {
	operationManager *process.ProvisionOperationManager
	secretKey        string
}

func NewConnectivityBindStep(os storage.Operations, secretKey string) *ConnectivityBindStep {
	return &ConnectivityBindStep{
		operationManager: process.NewProvisionOperationManager(os),
		secretKey:        secretKey,
	}
}

var _ Step = (*ConnectivityBindStep)(nil)

func (s *ConnectivityBindStep) Name() string {
	return "Connectivity_Bind"
}

func (s *ConnectivityBindStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if !operation.Connectivity.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("connectivity Provisioning step was not triggered"), log, "")
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manager client"))
	}
	// test if the provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Connectivity.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("LastInstanceOperation() call failed"))
	}
	log.Infof("Provisioning Connectivity (instanceID=%s) state: %s", operation.Connectivity.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Connectivity provisioning failed: %s", resp.Description), log)
	}
	// execute binding
	var connectivityOverrides *ConnectivityConfig
	if !operation.Connectivity.Instance.Provisioned {
		if operation.Connectivity.BindingID == "" {
			operation.Connectivity.BindingID = uuid.New().String()
		}
		respBinding, err := smCli.Bind(operation.Connectivity.Instance.InstanceKey(), operation.Connectivity.BindingID, nil, false)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Bind() call failed"))
		}
		// get overrides
		connectivityOverrides, err = GetConnectivityCredentials(respBinding.Binding)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("unable to load config"))
		}
		encryptedOverrides, err := EncryptConnectivityConfig(s.secretKey, connectivityOverrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("unable to encrypt config"))
		}

		// save the status
		op, retry := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			operation.Connectivity.Overrides = encryptedOverrides
			operation.Connectivity.Instance.Provisioned = true
			operation.Connectivity.Instance.ProvisioningTriggered = false
		}, log)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
		operation = op
	} else {
		// get the credentials from encrypted string in operation.Connectivity.Instance.
		connectivityOverrides, err = DecryptConnectivityConfig(s.secretKey, operation.Connectivity.Overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("unable to decrypt configs"))
		}
	}
	log.Errorf("1OVERRIDES: %+v", connectivityOverrides)
	log.Errorf("2OVERRIDES: %v", connectivityOverrides)
	logrus.Errorf("3OVERRIDES: %+v", connectivityOverrides)
	logrus.Errorf("4OVERRIDES: %v", connectivityOverrides)

	// TODO: Decide how we want to pass this data to the SKR. Currently,
	//       credentials are prepared as a ConnectivityConfig structure.
	//       See the github card - https://github.com/orgs/kyma-project/projects/6#card-56776111
	//       ...
	//       - [ ] define what changes need to be done in KEB to
	//             allow passing secrets data to the Provisioner
	log.Debugf("Got Connectivity Service credentials from the binding.")

	return operation, 0, nil
}

func (s *ConnectivityBindStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg, log)
}

func GetConnectivityCredentials(binding servicemanager.Binding) (*ConnectivityConfig, error) {
	credentials := binding.Credentials
	csMap, ok := credentials["connectivity_service"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(
			"failed to convert connectivity_service part of the credentials to map[string]interface{}")
	}

	return &ConnectivityConfig{
		ClientId:     credentials["clientid"].(string),
		ClientSecret: credentials["clientsecret"].(string),
		ConnectivityService: struct {
			CAsPath        string `json:"CAs_path"`
			CAsSigningPath string `json:"CAs_signing_path"`
			ApiPath        string `json:"api_path"`
			TunnelPath     string `json:"tunnel_path"`
			Url            string `json:"url"`
		}{
			CAsPath:        csMap["CAs_path"].(string),
			CAsSigningPath: csMap["CAs_signing_path"].(string),
			ApiPath:        csMap["CAs_signing_path"].(string),
			TunnelPath:     csMap["tunnel_path"].(string),
			Url:            csMap["url"].(string),
		},
		SubaccountId:                    credentials["subaccount_id"].(string),
		SubaccountSubdomain:             credentials["subaccount_subdomain"].(string),
		TokenServiceDomain:              credentials["token_service_domain"].(string),
		TokenServiceUrl:                 credentials["token_service_url"].(string),
		TokenServiceUrlPattern:          credentials["token_service_url_pattern"].(string),
		TokenServiceUrlPatternTenantKey: credentials["token_service_url_pattern_tenant_key"].(string),
		Xsappname:                       credentials["xsappname"].(string),
	}, nil
}

func EncryptConnectivityConfig(secretKey string, overrides *ConnectivityConfig) (string, error) {
	marshalledOverrides, err := json.Marshal(*overrides)
	if err != nil {
		return "", errors.Wrap(err, "while encoding connectivity overrides")
	}
	encryptedOverrides, err := storage.NewEncrypter(secretKey).Encrypt(marshalledOverrides)
	if err != nil {
		return "", errors.Wrap(err, "while encrypting connectivity overrides")
	}
	return string(encryptedOverrides), nil
}

func DecryptConnectivityConfig(secretKey string, encryptedOverrides string) (*ConnectivityConfig, error) {
	decryptedOverrides, err := storage.NewEncrypter(secretKey).Decrypt([]byte(encryptedOverrides))
	if err != nil {
		return nil, errors.Wrap(err, "while decrypting connectivity overrides")
	}
	connectivityOverrides := ConnectivityConfig{}
	if err := json.Unmarshal(decryptedOverrides, &connectivityOverrides); err != nil {
		return nil, errors.Wrap(err, "while unmarshalling connectivity overrides")
	}
	return &connectivityOverrides, nil
}
