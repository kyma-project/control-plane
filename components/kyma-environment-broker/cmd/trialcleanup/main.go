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
	SendExpirationRequest(instance internal.Instance) (bool, error)
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
	db, conn, err := storage.NewFromConfig(cfg.Database, cipher, log.WithField("service", "storage"))
	fatalOnError(err)
	svc := newTrialCleanupService(cfg, brokerClient, db.Instances())

	err = svc.PerformCleanup()

	fatalOnError(err)

	log.Info("Trial cleanup job finished successfully!")

	err = conn.Close()
	if err != nil {
		fatalOnError(err)
	}

	// do not use defer, close must be done before halting
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

	instancesToExpire, instancesToExpireCount := s.filterInstances(
		nonExpiredTrialInstances,
		func(instance internal.Instance) bool { return time.Since(instance.CreatedAt) >= s.cfg.ExpirationPeriod },
	)

	instancesToBeLeftCount := nonExpiredTrialInstancesCount - instancesToExpireCount

	if s.cfg.DryRun {
		s.logInstances(instancesToExpire)
		log.Infof("Trials non-expired: %+v to expire now: %+v to be left untouched: %+v", nonExpiredTrialInstancesCount, instancesToExpireCount, instancesToBeLeftCount)
	} else {
		suspensionsAcceptedCount, onlyMarkedAsExpiredCount, failuresCount := s.cleanupInstances(instancesToExpire)
		log.Infof("Trials non-expired: %+v to expire: %+v left untouched: %+v suspension under way: %+v only marked expired: %+v failures: %+v", nonExpiredTrialInstancesCount, instancesToExpireCount, instancesToBeLeftCount, suspensionsAcceptedCount, onlyMarkedAsExpiredCount, failuresCount)
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

func (s *TrialCleanupService) filterInstances(instances []internal.Instance, filter instancePredicate) ([]internal.Instance, int) {
	var filteredInstances []internal.Instance
	for _, instance := range instances {
		if filter(instance) {
			filteredInstances = append(filteredInstances, instance)
		}
	}
	return filteredInstances, len(filteredInstances)
}

func (s *TrialCleanupService) cleanupInstances(instances []internal.Instance) (int, int, int) {
	var suspensionAccepted int
	var onlyExpirationMarked int
	totalInstances := len(instances)
	for _, instance := range instances {
		suspensionUnderWay, err := s.expireInstance(instance)
		if err != nil {
			// ignoring errors - only logging
			log.Error(errors.Wrapf(err, "while sending expiration request for instanceID: %s", instance.InstanceID))
			continue
		}
		if suspensionUnderWay {
			suspensionAccepted += 1
		} else {
			onlyExpirationMarked += 1
		}
	}
	failures := totalInstances - suspensionAccepted - onlyExpirationMarked
	return suspensionAccepted, onlyExpirationMarked, failures
}

func (s *TrialCleanupService) logInstances(instances []internal.Instance) {
	for _, instance := range instances {
		log.Infof("instanceId: %+v createdAt: %+v (%.0f days ago) servicePlanID: %+v servicePlanName: %+v",
			instance.InstanceID, instance.CreatedAt, time.Since(instance.CreatedAt).Hours()/24, instance.ServicePlanID, instance.ServicePlanName)
	}
}

func (s *TrialCleanupService) expireInstance(instance internal.Instance) (processed bool, err error) {
	log.Infof("About to make instance suspended for instanceId: %+v", instance.InstanceID)
	suspensionUnderWay, err := s.brokerClient.SendExpirationRequest(instance)
	if err != nil {
		log.Error(errors.Wrapf(err, "while sending expiration request for instance ID %q", instance.InstanceID))
		return suspensionUnderWay, err
	}
	return suspensionUnderWay, nil
}

func fatalOnError(err error) {
	if err != nil {
		// temporarily we exit with 0 to avoid any side effects - we ignore all errors only logging those
		//log.Fatal(err)
		log.Error(err)
		os.Exit(0)
	}
}
