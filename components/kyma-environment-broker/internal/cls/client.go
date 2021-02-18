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

type ClsOverrides struct {
	FluentdEndPoint string `json:"Fluentd-endpoint"`
	FluentdPassword string `json:"Fluentd-password"`
	FluentdUsername string `json:"Fluentd-username"`
	KibanaUrl       string `json:"Kibana-endpoint"`
}

type BindingRequest struct {
	InstanceKey servicemanager.InstanceKey
	//SKRInstanceID   string
	//Bound bool
	BindingID string
	//ClsOverrides string
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

// CreateInstance creates a CLS Instance
// Instance creation means creation of a cluster, which must be reusable for the same instance/region/project
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

	c.log.Infof("Response from service manager while deprovisioning an instance %s: %#v", instance.InstanceID, resp)

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

type bindParam struct{}

func (c *Client) CreateBinding(smClient servicemanager.Client, request *BindingRequest) (*ClsOverrides, error) {
	var bp bindParam

	respBinding, err := smClient.Bind(request.InstanceKey, request.BindingID, bp, false)
	if err != nil {
		return nil, errors.Wrapf(err, "Bind() call failed")
	}
	// get overrides
	clsOverrides, err := getCredentials(respBinding.Binding)
	if err != nil {
		return nil, errors.Wrapf(err, "getCredentials() call failed")
	}
	return clsOverrides, nil

}

func getCredentials(binding servicemanager.Binding) (*ClsOverrides, error) {
	clsOverrides := ClsOverrides{}
	credentials := binding.Credentials
	clsOverrides.KibanaUrl = credentials["Kibana-endpoint"].(string)
	clsOverrides.FluentdUsername = credentials["Fluentd-username"].(string)
	clsOverrides.FluentdPassword = credentials["Fluentd-password"].(string)
	clsOverrides.FluentdEndPoint = credentials["Fluentd-endpoint"].(string)
	return &clsOverrides, nil
}
