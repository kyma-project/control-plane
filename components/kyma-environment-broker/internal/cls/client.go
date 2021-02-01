package cls

import (
	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

type client struct {
	log logrus.FieldLogger
	smCli servicemanager.Client
	operation internal.ProvisioningOperation
}

type clsParameters struct {
	RetentionPeriod    int  `json:"retentionPeriod"`
	MaxDataInstances   int  `json:"maxDataInstances"`
	MaxIngestInstances int  `json:"maxIngestInstances"`
	EsAPIEnabled       bool `json:"esApiEnabled"`
}

func NewClient(cfg Config, log logrus.FieldLogger) Client {
	return &client{
		smCli:         cfg.,
		clusterType: cfg.ClusterType,
		environment: cfg.Environment,
		token:       cfg.Token,
		samlTenant:  cfg.SamlTenant,
		log:         log,
	}
}

// CreateTenant create the LMS tenant
// Tenant creation means creation of a cluster, which must be reusable for the same tenant/region/project
func (c *client) CreateInstance(input CreateInstanceInput) (o CreateInstanceOutput, err error) {
	// Check if we already have a cls instance assigned to the GA, if so use it

	// No cls instance assigned to GA provision a new one.

	var smInput servicemanager.ProvisioningInput
	smInput.ID = uuid.New().String()
	smInput.ServiceID = c.operation.Cls.Instance.ServiceID
	smInput.PlanID = c.operation.Cls.Instance.PlanID
	smInput.SpaceGUID = uuid.New().String()
	smInput.OrganizationGUID = uuid.New().String()
	smInput.Context = map[string]interface{}{
		"platform": "kubernetes",
	}
	smInput.Parameters = clsParameters{
		RetentionPeriod:    7,
		MaxDataInstances:   2,
		MaxIngestInstances: 2,
		EsAPIEnabled:       false,
	}

	resp, err := c.smCli.Provision(c.operation.Cls.Instance.BrokerID, smInput, true)
	if err != nil {
		return CreateInstanceOutput{}, errors.Wrapf(err, "Provision() call failed for brokerID: %s; smInput: %#v", c.operation.Cls.Instance.BrokerID, smInput)
	}
	c.log.Infof("response from CLS provisioning call: %#v", resp)

	c.operation.Cls.Instance.InstanceID = smInput.ID

	return CreateInstanceOutput{ID: smInput.ID}, nil
}