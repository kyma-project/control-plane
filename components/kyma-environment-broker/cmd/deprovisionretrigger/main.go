package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

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

func main() {
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
}

func newDeprovisionRetriggerService(cfg Config, brokerClient BrokerClient, instances storage.Instances) *DeprovisionRetriggerService {
	return &DeprovisionRetriggerService{
		cfg:             cfg,
		instanceStorage: instances,
		brokerClient:    brokerClient,
	}
}

func (s *DeprovisionRetriggerService) PerformCleanup() error {
	notCompletelyDeletedFilter := dbmodel.InstanceFilter{DeletionAttempted: &[]bool{true}[0]}
	instancesToDeprovisionAgain, _, _, err := s.instanceStorage.List(notCompletelyDeletedFilter)

	if err != nil {
		log.Error(fmt.Sprintf("while getting not completely deprovisioned instances: %s", err))
		return err
	}

	if s.cfg.DryRun {
		s.logInstances(instancesToDeprovisionAgain)
		log.Infof("Instances to retrigger deprovisioning: %d", len(instancesToDeprovisionAgain))
	} else {
		deprovisioningAccepted, failuresCount := s.retriggerDeprovisioningForInstances(instancesToDeprovisionAgain)
		log.Infof("Instances to retrigger deprovisioning: %d, accepted requests: %d, failed requests: %d", len(instancesToDeprovisionAgain), deprovisioningAccepted, failuresCount)
	}

	return nil
}

func (s *DeprovisionRetriggerService) retriggerDeprovisioningForInstances(instances []internal.Instance) (int, int) {
	var failuresCount int
	for _, instance := range instances {
		err := s.deprovisionInstance(instance)
		if err != nil {
			// just counting, logging and ignoring errors
			failuresCount += 1
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
