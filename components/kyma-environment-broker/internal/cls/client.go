package cls

import (
	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type client struct {
	log logrus.FieldLogger
}

func NewClient(log logrus.FieldLogger) Client {
	return &client{
		log: log,
	}
}

type Client interface {
	CreateInstance(om *process.ProvisionOperationManager, smCli servicemanager.Client, op internal.ProvisioningOperation, input CreateInstanceInput) (o internal.ProvisioningOperation, err error)
}

//type clsParameters struct {
//	RetentionPeriod    int  `json:"retentionPeriod"`
//	MaxDataInstances   int  `json:"maxDataInstances"`
//	MaxIngestInstances int  `json:"maxIngestInstances"`
//	EsAPIEnabled       bool `json:"esApiEnabled"`
//}

// CreateTenant create the LMS tenant
// Tenant creation means creation of a cluster, which must be reusable for the same tenant/region/project
func (c *client) CreateInstance(om *process.ProvisionOperationManager, smCli servicemanager.Client, op internal.ProvisioningOperation, input CreateInstanceInput) (o internal.ProvisioningOperation, err error) {
	// Check if we already have a cls instance assigned to the GA, if so use it

	// No cls instance assigned to GA provision a new one.

	var smInput servicemanager.ProvisioningInput
	smInput.ID = uuid.New().String()
	smInput.ServiceID = op.Cls.Instance.ServiceID
	smInput.PlanID = op.Cls.Instance.PlanID
	smInput.SpaceGUID = uuid.New().String()
	smInput.OrganizationGUID = uuid.New().String()
	smInput.Context = map[string]interface{}{
		"platform": "kubernetes",
	}
	// TODO: Add Paramertes
	//smInput.Parameters = clsParameters{
	//	RetentionPeriod:    7,
	//	MaxDataInstances:   2,
	//	MaxIngestInstances: 2,
	//	EsAPIEnabled:       false,
	//}

	resp, err := smCli.Provision(op.Cls.Instance.BrokerID, smInput, true)
	if err != nil {
		return op, errors.Wrapf(err, "Provision() call failed for brokerID: %s; service manager : %#v", op.Cls.Instance.BrokerID, smInput)
	}
	c.log.Infof("response from CLS provisioning call: %#v", resp)

	op.Cls.Instance.ProvisioningTriggered = true
	_, retry := om.UpdateOperation(op)
	if retry > 0 {
		c.log.Errorf("unable to update operation")
		return op, errors.Wrapf(err, "Unable to update operation for brokerID: %s, service manger: %#v", op.Cls.Instance.BrokerID, smInput)
	}

	op.Cls.Instance.InstanceID = smInput.ID

	return op, nil
}
