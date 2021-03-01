package cls

import (
	"github.com/google/uuid"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type parameters struct {
	RetentionPeriod    int  `json:"retentionPeriod"`
	MaxDataInstances   int  `json:"maxDataInstances"`
	MaxIngestInstances int  `json:"maxIngestInstances"`
	EsAPIEnabled       bool `json:"esApiEnabled"`
	SAML               struct {
		Enabled     bool   `json:"enabled"`
		AdminGroup  string `json:"admin_group"`
		Initiated   bool   `json:"initiated"`
		ExchangeKey string `json:"exchange_key"`
		RolesKey    string `json:"roles_key"`
		Idp         struct {
			MetadataURL string `json:"metadata_url"`
			EntityID    string `json:"entity_id"`
		} `json:"idp"`
		Sp struct {
			EntityID            string `json:"entity_id"`
			SignaturePrivateKey string `json:"signature_private_key"`
		} `json:"sp"`
	} `json:"saml"`
}

type ClsOverrideParams struct {
	FluentdEndPoint string `json:"Fluentd-endpoint"`
	FluentdPassword string `json:"Fluentd-password"`
	FluentdUsername string `json:"Fluentd-username"`
	KibanaUrl       string `json:"Kibana-endpoint"`
}

type BindingRequest struct {
	InstanceKey servicemanager.InstanceKey
	BindingID   string
}

// Client wraps a generic servicemanager.Client an performs CLS specific calls
type Client struct {
	config *Config
	log    logrus.FieldLogger
}

//NewClient creates a new Client instance
func NewClient(config *Config, log logrus.FieldLogger) *Client {
	return &Client{
		config: config,
		log:    log,
	}
}

// CreateInstance sends a request to Service Manager to create a CLS Instance
func (c *Client) CreateInstance(smClient servicemanager.Client, instance servicemanager.InstanceKey) error {
	var input servicemanager.ProvisioningInput
	input.ID = instance.InstanceID
	input.ServiceID = instance.ServiceID
	input.PlanID = instance.PlanID
	input.SpaceGUID = uuid.New().String()
	input.OrganizationGUID = uuid.New().String()
	input.Context = map[string]interface{}{
		"platform": "kubernetes",
	}
	input.Parameters = createParameters(c.config)

	resp, err := smClient.Provision(instance.BrokerID, input, true)
	if err != nil {
		return errors.Wrapf(err, "while provisioning a cls instance %s", instance.InstanceID)
	}

	c.log.Infof("Response from service manager while provisioning an instance %s: %#v", instance.InstanceID, resp)

	return nil
}

func createParameters(config *Config) parameters {
	params := parameters{
		RetentionPeriod:    config.RetentionPeriod,
		MaxDataInstances:   config.MaxDataInstances,
		MaxIngestInstances: config.MaxIngestInstances,
		EsAPIEnabled:       false,
	}
	params.SAML.Enabled = true
	params.SAML.AdminGroup = config.SAML.AdminGroup
	params.SAML.Initiated = config.SAML.Initiated
	params.SAML.ExchangeKey = config.SAML.ExchangeKey
	params.SAML.RolesKey = config.SAML.RolesKey
	params.SAML.Idp.EntityID = config.SAML.Idp.EntityID
	params.SAML.Idp.MetadataURL = config.SAML.Idp.MetadataURL
	params.SAML.Sp.EntityID = config.SAML.Sp.EntityID
	params.SAML.Sp.SignaturePrivateKey = config.SAML.Sp.SignaturePrivateKey
	return params
}

func (c *Client) CreateBinding(smClient servicemanager.Client, request *BindingRequest) (*ClsOverrideParams, error) {
	var emptyParams struct{}

	resp, err := smClient.Bind(request.InstanceKey, request.BindingID, emptyParams, false)
	if err != nil {
		return nil, errors.Wrapf(err, "Bind() call failed")
	}

	return &ClsOverrideParams{
		KibanaUrl:       resp.Credentials["Kibana-endpoint"].(string),
		FluentdUsername: resp.Credentials["Fluentd-username"].(string),
		FluentdPassword: resp.Credentials["Fluentd-password"].(string),
		FluentdEndPoint: resp.Credentials["Fluentd-endpoint"].(string),
	}, nil
}

// RemoveInstance sends a request to Service Manager to remove a CLS Instance
func (c *Client) RemoveInstance(smClient servicemanager.Client, instance servicemanager.InstanceKey) error {
	resp, err := smClient.Deprovision(instance, true)
	if err != nil {
		return errors.Wrapf(err, "while deprovisioning a cls instance %s", instance.InstanceID)
	}

	c.log.Infof("Response from service manager while deprovisioning an instance %s: %#v", instance.InstanceID, resp)

	return nil
}
