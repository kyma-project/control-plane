package main

import (
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

type Config struct {
	Database         storage.Config
	DryRun           bool          `envconfig:"default=true"`
	ExpirationPeriod time.Duration `envconfig:"default=14d"`
}

type TrialCleanupService struct {
	cfg             Config
	instanceStorage storage.Instances
	logger          *log.Logger
	filter          dbmodel.InstanceFilter
}

type instancePredicate func(internal.Instance) bool

type instanceVisitor func(internal.Instance) error

func main() {
	time.Sleep(20 * times.Second)

	log.Info("Starting trial cleanup job!")

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	if cfg.DryRun {
		log.Info("Dry run only - no changes")
	}

	log.Infof("Expiration period: %+v", cfg.ExpirationPeriod)

	// create storage connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, _, err := storage.NewFromConfig(cfg.Database, cipher, log.WithField("service", "storage"))
	fatalOnError(err)

	logger := log.New()

	svc := newTrialCleanupService(cfg, db.Instances(), logger)

	err = svc.PerformCleanup()

	fatalOnError(err)

	log.Info("Trial cleanup job finished successfully!")

	err = cleaner.Halt()
	fatalOnError(err)

	time.Sleep(5 * time.Second)
}

func newTrialCleanupService(cfg Config, instances storage.Instances, logger *log.Logger) *TrialCleanupService {
	return &TrialCleanupService{
		cfg:             cfg,
		instanceStorage: instances,
		logger:          logger,
	}
}

func (s *TrialCleanupService) PerformCleanup() error {

	nonExpiredTrialInstancesFilter := dbmodel.InstanceFilter{PlanIDs: []string{trialPlanID}, Expired: &[]bool{false}[0]}
	nonExpiredTrialInstances, nonExpiredTrialInstancesCount, err := s.getInstances(nonExpiredTrialInstancesFilter)

	if err != nil {
		s.logger.Error(errors.Wrap(err, "while getting non expired trial instances"))
		return err
	}

	s.logger.Infof("Non expired trials to be processed: %+v", nonExpiredTrialInstancesCount)

	instancesToBeCleanedUp := s.filterInstances(nonExpiredTrialInstances, func(instance internal.Instance) bool { return time.Since(instance.CreatedAt) >= s.cfg.ExpirationPeriod })

	s.logger.Infof("Trials to be cleaned up: %+v", len(instancesToBeCleanedUp))

	if s.cfg.DryRun {
		return s.visitInstances(instancesToBeCleanedUp, s.executeDryRun)
	} else {
		return s.visitInstances(instancesToBeCleanedUp, s.suspendInstance)
	}
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

func (s *TrialCleanupService) visitInstances(instances []internal.Instance, visit instanceVisitor) error {
	for _, instance := range instances {
		err := visit(instance)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TrialCleanupService) executeDryRun(instance internal.Instance) error {
	s.logger.Infof("instanceId: %+v createdAt: %+v (so %.0f days ago) servicePlanID: %+v servicePlanName: %+v\n",
		instance.InstanceID, instance.CreatedAt, time.Since(instance.CreatedAt).Hours()/24, instance.ServicePlanID, instance.ServicePlanName)
	return nil
}

func (s *TrialCleanupService) suspendInstance(instance internal.Instance) error {
	s.logger.Infof("About to suspend instanceId: %+v - to be implemented", instance.InstanceID)
	return nil
}

func fatalOnError(err error) {
	if err != nil {
		// temporarily we exit with 0 to avoid any side effects - we ignore all errors only logging those
		//log.Fatal(err)
		log.Error(err)
		os.Exit(0)
	}
}
