package cls

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:generate mockery --name=InstanceStorage --output=automock --outpkg=automock --case=underscore
type InstanceStorage interface {
	FindInstance(globalAccountID string) (*internal.CLSInstance, bool, error)
	InsertInstance(instance internal.CLSInstance) error
	AddReference(globalAccountID, subAccountID string) error
}

//go:generate mockery --name=InstanceCreator --output=automock --outpkg=automock --case=underscore
type InstanceCreator interface {
	CreateInstance(smClient servicemanager.Client, brokerID, serviceID, planID string) (string, error)
}

type provisioner struct {
	storage InstanceStorage
	creator InstanceCreator
	log     logrus.FieldLogger
}

func NewProvisioner(storage InstanceStorage, creator InstanceCreator, log logrus.FieldLogger) *provisioner {
	return &provisioner{
		storage: storage,
		creator: creator,
		log:     log,
	}
}

type ProvisionRequest struct {
	GlobalAccountID string
	SubAccountID    string
	ServiceID       string
	PlanID          string
	BrokerID        string
}

type ProvisionResult struct {
	InstanceID            string
	ProvisioningTriggered bool
}

func (c *provisioner) ProvisionIfNoneExists(smClient servicemanager.Client, request *ProvisionRequest) (*ProvisionResult, error) {
	instance, exists, err := c.storage.FindInstance(request.GlobalAccountID)
	if err != nil {
		return nil, errors.Wrapf(err, "while checking if instance is already created")
	}

	if exists {
		return &ProvisionResult{
			InstanceID:            instance.ID,
			ProvisioningTriggered: false,
		}, nil
	}

	return c.createNewInstance(smClient, request)
}

func (c *provisioner) createNewInstance(smClient servicemanager.Client, request *ProvisionRequest) (*ProvisionResult, error) {
	instanceID, err := c.creator.CreateInstance(smClient, request.BrokerID, request.ServiceID, request.PlanID)
	if err != nil {
		return nil, errors.Wrapf(err, "while creating instance name=%s", request.GlobalAccountID)
	}

	instance := internal.CLSInstance{
		ID:              instanceID,
		GlobalAccountID: request.GlobalAccountID,
		CreatedAt:       time.Now(),
		SubAccountRefs:  []string{request.SubAccountID},
	}

	// it is important to save the instance ID because instance creation means creation of a cluster.
	err = wait.PollImmediate(3*time.Second, 30*time.Second, func() (bool, error) {
		err := c.storage.InsertInstance(instance)
		if err != nil {
			c.log.Warn(errors.Wrapf(err, "while saving cls instance %s with ID %s", request.GlobalAccountID, instance.ID).Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "while saving instance to storage")
	}

	return &ProvisionResult{
		InstanceID:            instance.ID,
		ProvisioningTriggered: true,
	}, nil
}
