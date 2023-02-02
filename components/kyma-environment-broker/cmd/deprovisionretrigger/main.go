package main

import (
	"context"
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"os"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/events"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/schema-migrator/cleaner"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

type BrokerClient interface {
	Deprovision(instance internal.Instance) (string, error)
}

type Config struct {
	Database storage.Config
	Broker   broker.ClientConfig
	DryRun   bool `envconfig:"default=true"`
}

type DeprovisionRetriggerService struct {
	cfg             Config
	filter          dbmodel.InstanceFilter
	instanceStorage storage.Instances
	brokerClient    BrokerClient
}

type instancePredicate func(internal.Instance) bool

func main() {
	time.Sleep(20 * time.Second)

	log.SetFormatter(&log.JSONFormatter{})
	log.Info("Starting deprovision retrigger job!")

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	if cfg.DryRun {
		log.Info("Dry run only - no changes")
	}

	ctx := context.Background()
	brokerClient := broker.NewClient(ctx, cfg.Broker)

	// create storage connection
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher, log.WithField("service", "storage"))
	fatalOnError(err)
	svc := newDeprovisionRetriggerService(cfg, brokerClient, db.Instances())

	err = svc.PerformCleanup()

	fatalOnError(err)

	log.Info("Deprovision retrigger job finished successfully!")

	err = conn.Close()
	if err != nil {
		fatalOnError(err)
	}

	// do not use defer, close must be done before halting
	err = cleaner.Halt()
	fatalOnError(err)

	time.Sleep(5 * time.Second)
}

func newDeprovisionRetriggerService(cfg Config, brokerClient BrokerClient, instances storage.Instances) *DeprovisionRetriggerService {
	return &DeprovisionRetriggerService{
		cfg:             cfg,
		instanceStorage: instances,
		brokerClient:    brokerClient,
	}
}

func (s *DeprovisionRetriggerService) PerformCleanup() error {
	allInstances, _, _, err := s.instanceStorage.List(dbmodel.InstanceFilter{})

	if err != nil {
		log.Error(fmt.Sprintf("while getting not completely deprovisioned instances: %s", err))
		return err
	}

	instancesToDeprovision, _ := s.filterInstances(allInstances,
		func(instance internal.Instance) bool { return !instance.DeletedAt.IsZero() },
	)

	if s.cfg.DryRun {
		s.logInstances(instancesToDeprovision)
		log.Infof("Instances to retrigger deprovisioning: %d", len(instancesToDeprovision))
	} else {
		deprovisioningAccepted, failuresCount := s.retriggerDeprovisioningForInstances(instancesToDeprovision)
		log.Infof("Instances to retrigger deprovisioning: %d, accepted requests: %d, failed requests: %d", len(instancesToDeprovision), deprovisioningAccepted, failuresCount)
	}

	return nil
}

func (s *DeprovisionRetriggerService) filterInstances(instances []internal.Instance, filter instancePredicate) ([]internal.Instance, int) {
	var filteredInstances []internal.Instance
	for _, instance := range instances {
		if filter(instance) {
			filteredInstances = append(filteredInstances, instance)
		}
	}
	return filteredInstances, len(filteredInstances)
}

func (s *DeprovisionRetriggerService) retriggerDeprovisioningForInstances(instances []internal.Instance) (int, int) {
	var failuresCount int
	for _, instance := range instances {
		err := s.deprovisionInstance(instance)
		if err != nil {
			// just counting, logging and ignoring errors
			failuresCount += 1
			log.Error(fmt.Sprintf("while sending deprovision request for instanceID: %s, error: %s", instance.InstanceID, err))
			continue
		}
	}
	return len(instances) - failuresCount, failuresCount
}

func (s *DeprovisionRetriggerService) deprovisionInstance(instance internal.Instance) (err error) {
	log.Infof("About to deprovision instance for instanceId: %+v", instance.InstanceID)
	operationId, err := s.brokerClient.Deprovision(instance)
	if err != nil {
		log.Error(fmt.Sprintf("while sending deprovision request for instance ID %s: %s", instance.InstanceID, err))
		return err
	}
	log.Infof("Deprovision instance for instanceId: %s accepted, operationId: %s", instance.InstanceID, operationId)
	return nil
}

func (s *DeprovisionRetriggerService) logInstances(instances []internal.Instance) {
	for _, instance := range instances {
		log.Infof("instanceId: %s, createdAt: %+v, deletedAt %+v", instance.InstanceID, instance.CreatedAt, instance.DeletedAt)
	}
}

func fatalOnError(err error) {
	if err != nil {
		// temporarily we exit with 0 to avoid any side effects - we ignore all errors only logging those
		//log.Fatal(err)
		log.Error(err)
		os.Exit(0)
	}
}
