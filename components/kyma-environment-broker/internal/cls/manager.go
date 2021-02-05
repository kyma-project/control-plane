package cls

import (
	"regexp"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type InstanceStorage interface {
	FindInstance(globalAccountID string) (internal.CLSInstance, bool, error)
	InsertInstance(instance internal.CLSInstance) error
}

type InstanceCreator interface {
	CreateInstance(smClient servicemanager.Client, serviceID, planID, brokerID string) (string, error)
}

type manager struct {
	storage InstanceStorage
	creator InstanceCreator
	log     logrus.FieldLogger
}

func NewInstanceManager(storage InstanceStorage, creator InstanceCreator, log logrus.FieldLogger) *manager {
	return &manager{
		storage: storage,
		creator: creator,
		log:     log,
	}
}

func (c *manager) CreateInstanceIfNoneExists(om *process.ProvisionOperationManager, smCli servicemanager.Client, op internal.ProvisioningOperation, globalAccountID string) (internal.ProvisioningOperation, error) {
	normalizedGlobalAccountID := normalize(globalAccountID)

	instance, exists, err := c.storage.FindInstance(normalizedGlobalAccountID)
	if err != nil {
		return op, errors.Wrapf(err, "while checking if instance is already created")
	}

	if exists {
		op.Cls.Instance.InstanceID = instance.ID
		return op, nil
	}

	output, err := c.creator.CreateInstance(smCli, op.Cls.Instance.ServiceID, op.Cls.Instance.PlanID, op.Cls.Instance.BrokerID)
	if err != nil {
		return op, errors.Wrapf(err, "while creating instance name=%s", normalizedGlobalAccountID)
	}

	op.Cls.Instance.ProvisioningTriggered = true

	// it is important to save the instance ID because instance creation means creation of a cluster.
	err = wait.PollImmediate(3*time.Second, 30*time.Second, func() (bool, error) {
		err := c.storage.InsertInstance(internal.CLSInstance{
			ID:        output,
			Name:      normalizedGlobalAccountID,
			CreatedAt: time.Now(),
		})
		if err != nil {
			c.log.Warn(errors.Wrapf(err, "while saving cls instance %s with ID %s", normalizedGlobalAccountID, output).Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return op, errors.Wrapf(err, "while saving instance to storage")
	}
	return op, nil
}

func normalize(s string) string {
	instanceNameNormalizationRegexp := regexp.MustCompile("[^a-zA-Z0-9]+")
	normalized := instanceNameNormalizationRegexp.ReplaceAllString(s, "")
	if len(normalized) > 50 {
		normalized = normalized[:50]
	}

	return normalized
}
