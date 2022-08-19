package main

import (
	"context"
	"os"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/schema-migrator/cleaner"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

const (
	trialPlanID = broker.TrialPlanID
)

type BrokerClient interface {
	SendExpirationRequest(instance internal.Instance) (string, error)
}

type Config struct {
	Database         storage.Config
	Broker           broker.ClientConfig
	DryRun           bool          `envconfig:"default=true"`
	ExpirationPeriod time.Duration `envconfig:"default=336h"`
}

type TrialCleanupService struct {
	cfg             Config
	filter          dbmodel.InstanceFilter
	instanceStorage storage.Instances
	brokerClient    BrokerClient
}

type instancePredicate func(internal.Instance) bool

func main() {
	time.Sleep(20 * time.Second)

	log.SetFormatter(&log.JSONFormatter{})
	log.Info("Starting trial cleanup job!")

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	if cfg.DryRun {
		log.Info("Dry run only - no changes")
	}

	log.Infof("Expiration period: %+v", cfg.ExpirationPeriod)

	ctx := context.Background()
	brokerClient := broker.NewClient(ctx, cfg.Broker)

	// create storage connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, _, err := storage.NewFromConfig(cfg.Database, cipher, log.WithField("service", "storage"))
	fatalOnError(err)
	svc := newTrialCleanupService(cfg, brokerClient, db.Instances())

	err = svc.PerformCleanup()

	fatalOnError(err)

	log.Info("Trial cleanup job finished successfully!")

	err = cleaner.Halt()
	fatalOnError(err)

	time.Sleep(5 * time.Second)
}

func newTrialCleanupService(cfg Config, brokerClient BrokerClient, instances storage.Instances) *TrialCleanupService {
	return &TrialCleanupService{
		cfg:             cfg,
		instanceStorage: instances,
		brokerClient:    brokerClient,
	}
}

func (s *TrialCleanupService) PerformCleanup() error {

	nonExpiredTrialInstancesFilter := dbmodel.InstanceFilter{PlanIDs: []string{trialPlanID}, Expired: &[]bool{false}[0]}
	nonExpiredTrialInstances, nonExpiredTrialInstancesCount, err := s.getInstances(nonExpiredTrialInstancesFilter)

	if err != nil {
		log.Error(errors.Wrap(err, "while getting non expired trial instances"))
		return err
	}

	log.Infof("Non expired trials to be processed: %+v", nonExpiredTrialInstancesCount)

	instancesToBeCleanedUp := s.filterInstances(nonExpiredTrialInstances, func(instance internal.Instance) bool { return time.Since(instance.CreatedAt) >= s.cfg.ExpirationPeriod })

	log.Infof("Trials to be cleaned up: %+v", len(instancesToBeCleanedUp))
	log.Infof("Trials to be left untouched: %+v", nonExpiredTrialInstancesCount-len(instancesToBeCleanedUp))

	if s.cfg.DryRun {
		s.logInstances(instancesToBeCleanedUp)
	} else {
		s.cleanupInstances(instancesToBeCleanedUp)
	}
	return nil
}

func (s *TrialCleanupService) getInstances(filter dbmodel.InstanceFilter) ([]internal.Instance, int, error) {

	instances, _, totalCount, err := s.instanceStorage.List(filter)
	if err != nil {
		return []internal.Instance{}, 0, err
	}

	return instances, totalCount, nil
}

func (s *TrialCleanupService) filterInstances(instances []internal.Instance, filter instancePredicate) []internal.Instance {
	var filteredInstances []internal.Instance
	for _, instance := range instances {
		if filter(instance) {
			filteredInstances = append(filteredInstances, instance)
		}
	}
	return filteredInstances
}

func (s *TrialCleanupService) cleanupInstances(instances []internal.Instance) {
	var processedInstances int
	var unprocessedInstances int
	totalInstances := len(instances)
	for _, instance := range instances {
		processed, err := s.suspendInstance(instance)
		if err != nil {
			// ignoring errors - only logging
			log.Error(errors.Wrap(err, "while sending expiration request"))
			continue
		}
		if processed {
			processedInstances += 1
		} else {
			unprocessedInstances += 1
		}
	}
	failures := totalInstances - processedInstances - unprocessedInstances
	log.Infof("To suspend: %+v processable: %+v unprocessable: %+v failures: %+v", totalInstances, processedInstances, unprocessedInstances, failures)
}

func (s *TrialCleanupService) logInstances(instances []internal.Instance) {
	for _, instance := range instances {
		log.Infof("instanceId: %+v createdAt: %+v (%.0f days ago) servicePlanID: %+v servicePlanName: %+v",
			instance.InstanceID, instance.CreatedAt, time.Since(instance.CreatedAt).Hours()/24, instance.ServicePlanID, instance.ServicePlanName)
	}
}

func (s *TrialCleanupService) suspendInstance(instance internal.Instance) (processed bool, err error) {
	log.Infof("About to make instance suspended for instanceId: %+v", instance.InstanceID)
	opID, err := s.brokerClient.SendExpirationRequest(instance)
	if err != nil {
		log.Error(errors.Wrapf(err, "while triggering expiration of instance ID %q", instance.InstanceID))
		return false, err
	}
	if len(opID) == 0 {
		log.Info("Request sent successfully to Kyma Environment Broker - got unprocessable entity")
		return false, nil
	}
	log.Infof("Request sent successfully to Kyma Environment Broker - got operation ID %q", opID)
	return true, nil
}

func fatalOnError(err error) {
	if err != nil {
		// temporarily we exit with 0 to avoid any side effects - we ignore all errors only logging those
		//log.Fatal(err)
		log.Error(err)
		os.Exit(0)
	}
}
