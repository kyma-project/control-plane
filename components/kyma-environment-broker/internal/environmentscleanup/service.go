package environmentscleanup

import (
	"context"
	"fmt"
	"strings"
	"time"

	error2 "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"

	"github.com/hashicorp/go-multierror"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	run "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	shootAnnotationRuntimeId = "kcp.provisioner.kyma-project.io/runtime-id"
	shootLabelAccountId      = "account"
)

//go:generate mockery --name=GardenerClient --output=automock
type GardenerClient interface {
	List(context context.Context, opts v1.ListOptions) (*unstructured.UnstructuredList, error)
	Get(ctx context.Context, name string, options v1.GetOptions, subresources ...string) (*unstructured.Unstructured, error)
}

//go:generate mockery --name=BrokerRuntimesClient --output=automock
type BrokerRuntimesClient interface {
	ListRuntimes(params run.ListParameters) (run.RuntimesPage, error)
}

//go:generate mockery --name=BrokerClient --output=automock
type BrokerClient interface {
	Deprovision(instance internal.Instance) (string, error)
}

//go:generate mockery --name=ProvisionerClient --output=automock
type ProvisionerClient interface {
	DeprovisionRuntime(accountID, runtimeID string) (string, error)
}

type Service struct {
	gardenerService   GardenerClient
	brokerService     BrokerClient
	brokerRuntimes    run.Client
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

func NewService(gardenerClient GardenerClient, brokerClient BrokerClient, brokerRuntimesClient run.Client, provisionerClient ProvisionerClient, instanceStorage storage.Instances, logger *log.Logger, maxShootAge time.Duration, labelSelector string) *Service {
	return &Service{
		gardenerService:   gardenerClient,
		brokerService:     brokerClient,
		brokerRuntimes:    brokerRuntimesClient,
		instanceStorage:   instanceStorage,
		logger:            logger,
		MaxShootAge:       maxShootAge,
		LabelSelector:     labelSelector,
		provisionerClient: provisionerClient,
	}
}

func (s *Service) PerformCleanup() error {
	// err := s.cleanupByRuntimes()
	// if err != nil {
	// 	s.logger.Error(fmt.Errorf("while cleanning up stale shoots: %w", err))
	// 	return err
	// }

	return s.cleanupByShoots()
}

func (s *Service) cleanupByRuntimes() error {
	runtimes, err := s.getStaleRuntimes(s.LabelSelector)
	if err != nil {
		s.logger.Error(fmt.Errorf("while getting stale shoots to delete: %w", err))
		return err
	}

	return s.cleanupRuntimes(runtimes)

}

func (s *Service) cleanupByShoots() error {
	runtimesToDelete, err := s.getStaleRuntimesByShoots(s.LabelSelector)
	if err != nil {
		s.logger.Error(fmt.Errorf("while getting stale shoots to delete: %w", err))
		return err
	}

	return s.cleanupRuntimes(runtimesToDelete)
}

func (s *Service) cleanupRuntimes(runtimes []runtime) error {
	s.logger.Infof("Runtimes to process: %+v\n", runtimes)

	if len(runtimes) == 0 {
		return nil
	}

	return s.cleanUp(runtimes)
}

func (s *Service) getStaleRuntimesByShoots(labelSelector string) ([]runtime, error) {
	opts := v1.ListOptions{
		LabelSelector: labelSelector,
	}
	shootList, err := s.gardenerService.List(context.Background(), opts)
	if err != nil {
		return []runtime{}, fmt.Errorf("while listing Gardener shoots: %w", err)
	}

	var runtimes []runtime
	for _, shoot := range shootList.Items {
		shootCreationTimestamp := shoot.GetCreationTimestamp()
		shootAge := time.Since(shootCreationTimestamp.Time)

		if shootAge.Hours() < s.MaxShootAge.Hours() {
			continue
		}

		_, toRemove := shoot.GetLabels()[notRemoveLabel]
		if toRemove {

			log.Infof("Shoot %q is older than %f hours with age: %f hours", shoot.GetName(), s.MaxShootAge.Hours(), shootAge.Hours())
			staleRuntime, err := s.shootToRuntime(shoot)
			if err != nil {
				s.logger.Error(err)
				continue
			}

			runtimes = append(runtimes, *staleRuntime)
		}
	}

	return runtimes, nil
}

func (s *Service) getStaleRuntimes(labelSelector string) ([]runtime, error) {
	page := 0
	var stale []runtime
	for {
		runtimesPage, err := s.brokerRuntimes.ListRuntimes(run.ListParameters{Page: page, PageSize: 3})
		if err != nil {
			return []runtime{}, fmt.Errorf("while listing Broker shoots: %w", err)
		}

		if runtimesPage.Count <= 0 {
			break
		}

		for _, loadedRuntime := range runtimesPage.Data {
			createdAt := loadedRuntime.Status.Provisioning.CreatedAt
			shootAge := time.Since(createdAt).Hours()
			maxAge := s.MaxShootAge.Hours()

			if shootAge >= maxAge {
				log.Infof("Runtime %q [shoot: %q] is older than %f hours with age: %f hours", loadedRuntime.RuntimeID, loadedRuntime.ShootName, maxAge, shootAge)

				shoot, err := s.gardenerService.Get(context.Background(), loadedRuntime.ShootName, v1.GetOptions{}, "shoot")
				// TODO: check not found error
				if err != nil {
					return []runtime{}, fmt.Errorf("while listing Gardener shoots: %w", err)
				}

				_, keep := shoot.GetLabels()["owner.do-not-delete"]
				if !keep {
					stale = append(stale, runtime{
						loadedRuntime.RuntimeID,
						loadedRuntime.SubAccountID,
					})
				}
			}
		}

		page += 1
	}

	return stale, nil
}

func (s *Service) getRuntimes(shoots []unstructured.Unstructured) []runtime {
	var stale []runtime
	for _, st := range shoots {
		runtime, err := s.shootToRuntime(st)
		if err != nil {
			s.logger.Error(err)
			continue
		}

		stale = append(stale, *runtime)
	}

	return stale
}

func (s *Service) shootToRuntime(st unstructured.Unstructured) (*runtime, error) {
	shoot := gardener.Shoot{st}
	runtimeID, ok := shoot.GetAnnotations()[shootAnnotationRuntimeId]
	if !ok {
		return nil, fmt.Errorf("shoot %q has no runtime-id annotation", shoot.GetName())
	}

	accountID, ok := shoot.GetLabels()[shootLabelAccountId]
	if !ok {
		return nil, fmt.Errorf("shoot %q has no account label", shoot.GetName())
	}

	return &runtime{
		ID:        runtimeID,
		AccountID: accountID,
	}, nil
}

func (s *Service) shootToRuntime(st unstructured.Unstructured) (*runtime, error) {
	shoot := gardener.Shoot{st}
	runtimeID, ok := shoot.GetAnnotations()[shootAnnotationRuntimeId]
	if !ok {
		return nil, fmt.Errorf("shoot %q has no runtime-id annotation", shoot.GetName())
	}

	accountID, ok := shoot.GetLabels()[shootLabelAccountId]
	if !ok {
		return nil, fmt.Errorf("shoot %q has no account label", shoot.GetName())
	}

	return &runtime{
		ID:        runtimeID,
		AccountID: accountID,
	}, nil
}

func (s *Service) cleanUp(runtimesToDelete map[string]runtime) error {
	kebInstancesToDelete, err := s.getInstancesForRuntimes(runtimesToDelete)
	if err != nil {
		err = fmt.Errorf("while getting instance IDs for Runtimes: %w", err)
		s.logger.Error(err)
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

func (s *Service) cleanUpShoots(shoots []unstructured.Unstructured) {
	for _, shoot := range shoots {
		err := s.gardenerService.Delete(context.Background(), shoot.GetName(), v1.DeleteOptions{}, "shoot")
		s.logger.Warn(fmt.Errorf("cannot remove a shoot %w", err))
		continue
	}
}

func (s *Service) getInstancesForRuntimes(runtimesToDelete map[string]runtime) ([]internal.Instance, error) {

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

func (s *Service) cleanUpProvisionerInstances(runtimesToDelete map[string]runtime, kebInstancesToDelete []internal.Instance) *multierror.Error {
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
	if error2.IsNotFoundError(err) {
		s.logger.Warnf("Runtime %s does not exists in the provisioner, skipping", runtime.ID)
		return nil
	}
	if err != nil {
		err = fmt.Errorf("while deprovisioning runtime with Provisioner: %w", err)
		s.logger.Error(err)
		return err
	}

	log.Infof("Successfully send deprovision request to Provisioner, got operation ID %q", operationID)
	return nil
}

func (s *Service) triggerEnvironmentDeprovisioning(instance internal.Instance) error {
	opID, err := s.brokerService.Deprovision(instance)
	if err != nil {
		err = fmt.Errorf("while triggering deprovisioning for instance ID %q: %w", instance.InstanceID, err)
		s.logger.Error(err)
		return err
	}

	log.Infof("Successfully send deprovision request to Kyma Environment Broker, got operation ID %q", opID)
	return nil
}
