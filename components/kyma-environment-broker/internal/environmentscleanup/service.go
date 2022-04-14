package environmentscleanup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	shootAnnotationRuntimeId = "kcp.provisioner.kyma-project.io/runtime-id"
	shootLabelAccountId      = "account"
)

//go:generate mockery -name=GardenerClient -output=automock
type GardenerClient interface {
	List(context context.Context, opts v1.ListOptions) (*unstructured.UnstructuredList, error)
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

type runtime struct {
	ID        string
	AccountID string
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

	staleShoots, err := s.getStaleShoots(s.LabelSelector)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "while getting shoots to delete"))
		return err
	}

	runtimesToDelete := s.getRuntimes(staleShoots)

	s.logger.Infof("Runtimes to process: %+v\n", runtimesToDelete)

	if len(runtimesToDelete) == 0 {
		return nil
	}

	return s.cleanUp(runtimesToDelete)
}

func (s *Service) getStaleShoots(labelSelector string) ([]unstructured.Unstructured, error) {
	opts := v1.ListOptions{
		LabelSelector: labelSelector,
	}
	shootList, err := s.gardenerService.List(context.Background(), opts)
	if err != nil {
		return []unstructured.Unstructured{}, errors.Wrap(err, "while listing Gardener shoots")
	}

	var shoots []unstructured.Unstructured
	for _, shoot := range shootList.Items {
		shootCreationTimestamp := shoot.GetCreationTimestamp()
		shootAge := time.Since(shootCreationTimestamp.Time)

		if shootAge.Hours() >= s.MaxShootAge.Hours() {
			log.Infof("Shoot %q is older than %f hours with age: %f hours", shoot.GetName(), s.MaxShootAge.Hours(), shootAge.Hours())
			shoots = append(shoots, shoot)
		}
	}

	return shoots, nil
}

func (s *Service) getRuntimes(shoots []unstructured.Unstructured) []runtime {
	var runtimes []runtime
	for _, st := range shoots {
		shoot := gardener.Shoot{st}
		runtimeID, ok := shoot.GetAnnotations()[shootAnnotationRuntimeId]
		if !ok {
			err := errors.New(fmt.Sprintf("shoot %q has no runtime-id annotation", shoot.GetName()))
			s.logger.Error(err)
			continue
		}

		accountID, ok := shoot.GetLabels()[shootLabelAccountId]
		if !ok {
			err := errors.New(fmt.Sprintf("shoot %q has no account label", shoot.GetName()))
			s.logger.Error(err)
			continue
		}

		runtimes = append(runtimes, runtime{
			ID:        runtimeID,
			AccountID: accountID,
		})
	}

	return runtimes
}

func (s *Service) cleanUp(runtimesToDelete []runtime) error {
	kebInstancesToDelete, err := s.getInstancesForRuntimes(runtimesToDelete)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "while getting instance IDs for Runtimes"))

		return err
	}

	kebResult := s.cleanUpKEBInstances(kebInstancesToDelete)
	provisionerResult := s.cleanUpProvisionerInstances(runtimesToDelete, kebInstancesToDelete)
	result := multierror.Append(kebResult, provisionerResult)

	if result != nil {
		result.ErrorFormat = func(i []error) string {
			var s []string
			for _, v := range i {
				s = append(s, v.Error())
			}
			return strings.Join(s, ", ")
		}
	}

	return result.ErrorOrNil()
}

func (s *Service) getInstancesForRuntimes(runtimesToDelete []runtime) ([]internal.Instance, error) {

	var runtimeIDsToDelete []string
	for _, runtime := range runtimesToDelete {
		runtimeIDsToDelete = append(runtimeIDsToDelete, runtime.ID)
	}

	instances, err := s.instanceStorage.FindAllInstancesForRuntimes(runtimeIDsToDelete)
	if err != nil {
		return []internal.Instance{}, err
	}

	return instances, nil
}

func (s *Service) cleanUpKEBInstances(instancesToDelete []internal.Instance) *multierror.Error {
	var result *multierror.Error

	for _, instance := range instancesToDelete {
		s.logger.Infof("Triggering environment deprovisioning for instance ID %q", instance.InstanceID)
		currentErr := s.triggerEnvironmentDeprovisioning(instance)
		if currentErr != nil {
			result = multierror.Append(result, currentErr)
		}
	}

	return result
}

func (s *Service) cleanUpProvisionerInstances(runtimesToDelete []runtime, kebInstancesToDelete []internal.Instance) *multierror.Error {
	kebInstanceExists := func(runtimeID string) bool {
		for _, instance := range kebInstancesToDelete {
			if instance.RuntimeID == runtimeID {
				return true
			}
		}

		return false
	}

	var result *multierror.Error

	for _, runtime := range runtimesToDelete {
		if !kebInstanceExists(runtime.ID) {
			s.logger.Infof("Triggering runtime deprovisioning for runtimeID ID %q", runtime.ID)
			err := s.triggerRuntimeDeprovisioning(runtime)
			if err != nil {
				result = multierror.Append(result, err)
			}
		}
	}

	return result
}

func (s *Service) triggerRuntimeDeprovisioning(runtime runtime) error {
	operationID, err := s.provisionerClient.DeprovisionRuntime(runtime.AccountID, runtime.ID)
	if err != nil {
		s.logger.Error(errors.Wrap(err, "while deprovisioning runtime with Provisioner"))
		return err
	}

	log.Infof("Successfully send deprovision request to Provisioner, got operation ID %q", operationID)
	return nil
}

func (s *Service) triggerEnvironmentDeprovisioning(instance internal.Instance) error {
	opID, err := s.brokerService.Deprovision(instance)
	if err != nil {
		s.logger.Error(errors.Wrapf(err, "while triggering deprovisioning for instance ID %q", instance.InstanceID))
		return err
	}

	log.Infof("Successfully send deprovision request to Kyma Environment Broker, got operation ID %q", opID)
	return nil
}
