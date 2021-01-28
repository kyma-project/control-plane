package environmentscleanup

import (
	"fmt"
	"strings"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hashicorp/go-multierror"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	shootAnnotationRuntimeId = "kcp.provisioner.kyma-project.io/runtime-id"
	shootLabelAccountId      = "account"
)

//go:generate mockery -name=GardenerClient -output=automock
type GardenerClient interface {
	List(opts v1.ListOptions) (*v1beta1.ShootList, error)
}

//go:generate mockery -name=BrokerClient -output=automock
type BrokerClient interface {
	Deprovision(instance internal.Instance) (string, error)
}

//go:generate mockery -name=ProvisionerClient -output=automock
type ProvisionerClient interface {
	DeprovisionRuntime(accountID, runtimeID string) (string, error)
}

type Service struct {
	gardenerService   GardenerClient
	brokerService     BrokerClient
	instanceStorage   storage.Instances
	logger            *log.Logger
	MaxShootAge       time.Duration
	LabelSelector     string
	provisionerClient ProvisionerClient
}

func NewService(gardenerClient GardenerClient, brokerClient BrokerClient, provisionerClient ProvisionerClient, instanceStorage storage.Instances, logger *log.Logger, maxShootAge time.Duration, labelSelector string) *Service {
	return &Service{
		gardenerService:   gardenerClient,
		brokerService:     brokerClient,
		instanceStorage:   instanceStorage,
		logger:            logger,
		MaxShootAge:       maxShootAge,
		LabelSelector:     labelSelector,
		provisionerClient: provisionerClient,
	}
}

func (s *Service) PerformCleanup() error {
	var result *multierror.Error
	formatFunc := func(i []error) string {
		var s []string
		for _, v := range i {
			s = append(s, v.Error())
		}
		return strings.Join(s, ", ")
	}

	shootsToDelete, err := s.getShootsToDelete(s.LabelSelector)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "while getting shoots to delete"))
		return err
	}

	var runtimeIDsToDelete []string
	for _, shoot := range shootsToDelete {
		runtimeID, ok := shoot.Annotations[shootAnnotationRuntimeId]
		if !ok {
			err = errors.New(fmt.Sprintf("shoot %q has no runtime-id annotation", shoot.Name))
			s.logger.Error(err)
			continue
		}
		runtimeIDsToDelete = append(runtimeIDsToDelete, runtimeID)
	}
	s.logger.Infof("Runtime IDs to process: %v", runtimeIDsToDelete)

	if len(runtimeIDsToDelete) == 0 {
		return nil
	}

	deletedInstances, result := s.cleanUpKEBInstances(runtimeIDsToDelete, formatFunc)
	if result != nil {
		result.ErrorFormat = formatFunc
	}

	s.cleanProvisionerInstances(shootsToDelete, deletedInstances, formatFunc)

	return result.ErrorOrNil()
}

func (s *Service) getShootsToDelete(labelSelector string) ([]v1beta1.Shoot, error) {
	opts := v1.ListOptions{
		LabelSelector: labelSelector,
	}
	shootList, err := s.gardenerService.List(opts)
	if err != nil {
		return []v1beta1.Shoot{}, errors.Wrap(err, "while listing Gardener shoots")
	}

	var shoots []v1beta1.Shoot
	for _, shoot := range shootList.Items {
		shootCreationTimestamp := shoot.GetCreationTimestamp()
		shootAge := time.Since(shootCreationTimestamp.Time)

		if shootAge.Hours() >= s.MaxShootAge.Hours() {
			log.Infof("Shoot %q is older than %f hours with age: %f hours", shoot.Name, s.MaxShootAge.Hours(), shootAge.Hours())
			shoots = append(shoots, shoot)
		}
	}

	return shoots, nil
}

func (s *Service) getInstancesForRuntimes(runtimeIDsToDelete []string) ([]internal.Instance, error) {
	instances, err := s.instanceStorage.FindAllInstancesForRuntimes(runtimeIDsToDelete)
	if err != nil {
		return []internal.Instance{}, err
	}

	return instances, nil
}

func (s *Service) triggerEnvironmentDeprovisioning(instance internal.Instance) error {
	opID, err := s.brokerService.Deprovision(instance)
	if err != nil {
		return err
	}
	log.Infof("Successfully send deprovision request, got operation ID %q", opID)
	return nil
}

func (s *Service) cleanUpKEBInstances(runtimeIDsToDelete []string, formatFunc func(i []error) string) ([]internal.Instance, *multierror.Error) {
	var result *multierror.Error

	instancesToDelete, err := s.getInstancesForRuntimes(runtimeIDsToDelete)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "while getting instance IDs for Runtimes"))
		result = multierror.Append(result, err)
		result.ErrorFormat = formatFunc
		return []internal.Instance{}, result
	}

	for _, instance := range instancesToDelete {
		s.logger.Infof("Triggering environment deprovisioning for instance ID %q", instance.InstanceID)
		currentErr := s.triggerEnvironmentDeprovisioning(instance)
		if currentErr != nil {
			s.logger.Error(errors.Wrapf(currentErr, "while triggering deprovisioning for instance ID %q", instance.InstanceID))
			result = multierror.Append(result, currentErr)
		}
	}

	return instancesToDelete, result
}

func (s *Service) cleanProvisionerInstances(shootsToDelete []v1beta1.Shoot, kebInstancesToDelete []internal.Instance, formatFunc func(i []error) string) *multierror.Error {
	kebInstanceExists := func(runtimeID string) bool {
		for _, instance := range kebInstancesToDelete {
			if instance.RuntimeID == runtimeID {
				return true
			}
		}

		return false
	}

	var result *multierror.Error

	for _, shoot := range shootsToDelete {
		runtimeID, ok := shoot.Annotations[shootAnnotationRuntimeId]
		if !ok {
			err := errors.New(fmt.Sprintf("shoot %q has no runtime-id annotation", shoot.Name))
			s.logger.Error(err)
			continue
		}

		accountID, ok := shoot.Labels[shootLabelAccountId]
		if !ok {
			err := errors.New(fmt.Sprintf("shoot %q has no account label", shoot.Name))
			s.logger.Error(err)
			continue
		}

		if !kebInstanceExists(runtimeID) {
			_, err := s.provisionerClient.DeprovisionRuntime(runtimeID, accountID)
			if err != nil {
				s.logger.Error(errors.Wrap(err, "while deprovisioning isntance with Provisioner"))
				result = multierror.Append(result, err)
				result.ErrorFormat = formatFunc
			}
		}
	}

	return result
}
