package cls

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"time"

"regexp"

"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
"github.com/pkg/errors"
"github.com/sirupsen/logrus"
"k8s.io/apimachinery/pkg/util/wait"
)

type InstanceStorage interface {
	FindInstanceByName(name, region string) (internal.CLSInstance, bool, error)
	InsertInstance(tenant internal.CLSInstance) error
}

//go:generate mockery -name=InstanceCreator -output=automock -outpkg=automock -case=underscore

// TODO: not sure if needed or we get instanceID from the service manager
type InstanceCreator interface {
	CreateInstance(om *process.ProvisionOperationManager, smCli servicemanager.Client, op internal.ProvisioningOperation, input CreateInstanceInput) (o internal.ProvisioningOperation, err error)
}

type manager struct {
	instanceStorage InstanceStorage
	clsClient     InstanceCreator
	log           logrus.FieldLogger
}

func NewInstanceManager(storage InstanceStorage, clsClient InstanceCreator, log logrus.FieldLogger) *manager {
	return &manager{
		instanceStorage: storage,
		clsClient:     clsClient,
		log:           log,
	}
}

//Todo: Verify this if we need to change the regex
var instanceNameNormalizationRegexp = regexp.MustCompile("[^a-zA-Z0-9]+")

func (c *manager) ProvideClsInstanceID(om *process.ProvisionOperationManager, smCli servicemanager.Client, op internal.ProvisioningOperation, globalAccountID string, region string) (internal.ProvisioningOperation, error) {
	name := instanceNameNormalizationRegexp.ReplaceAllString(globalAccountID, "")
	if len(name) > 50 {
		name = name[:50]
	}
	instance, exists, err := c.instanceStorage.FindInstanceByName(name, region)
	if err != nil {
		return op, errors.Wrapf(err, "while checking if tenant is already created")
	}

	if !exists {
		output, err := c.clsClient.CreateInstance(om, smCli, op, CreateInstanceInput{
			Name:            name,
			Region:          region,
			GlobalAccountID: globalAccountID,
		})
		if err != nil {
			return op, errors.Wrapf(err, "while creating instance name=%s region=%s in cls", name, region)
		}

		// it is important to save the tenant ID because tenant creation means creation of a cluster.
		err = wait.PollImmediate(3*time.Second, 30*time.Second, func() (bool, error) {
			err := c.instanceStorage.InsertInstance(internal.CLSInstance{
				ID:        output.Cls.Instance.BrokerID,
				Name:      name,
				Region:    region,
				CreatedAt: time.Now(),
			})
			if err != nil {
				c.log.Warn(errors.Wrapf(err, "while saving cls instance %s with ID %s", name, output.ID).Error())
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return op, errors.Wrapf(err, "while saving instance to storage")
		}
		return op, nil
	}
	op.Cls.Instance.InstanceID = instance.ID

	return op, nil
}

