package job

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/avast/retry-go"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/model"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dbconnection"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/util"
	log "github.com/sirupsen/logrus"
)

const maxErrors = 10

type ProviderConfigMigrator interface {
	Do() error
}

type providerConfigMigrator struct {
	dbsFactory dbconnection.Factory
	errors     []string
}

func NewProviderConfigMigrator(dbsFactory dbconnection.Factory) ProviderConfigMigrator {
	return &providerConfigMigrator{
		dbsFactory: dbsFactory,
		errors:     make([]string, 0),
	}
}

func (p *providerConfigMigrator) Do() error {
	session := p.dbsFactory.NewReadWriteSession()

	data, dberr := session.GetProviderSpecificConfigsByProvider(model.AWS)

	if dberr != nil {
		return dberr
	}

	for _, d := range data {
		log.Infof("Migrating Provider Config for runtime: %s", d.ClusterId)
		oldConfig := p.decodeAWSConfig(d)
		newConfig := mapAWSToNewModel(oldConfig, d.WorkerCidr)
		jsonConfig := p.marshalToJson(newConfig)
		p.updateConfig(session, d.ClusterId, jsonConfig)
		if len(p.errors) > maxErrors {
			log.Error("Too many errors!")
			return fmt.Errorf(strings.Join(p.errors, "\n"))
		}
	}
	return nil
}

func (p *providerConfigMigrator) decodeAWSConfig(data dbconnection.ProviderData) model.OldAWSProviderConfigInput {
	var oldConfigInput model.OldAWSProviderConfigInput

	err := util.DecodeJson(data.Config, &oldConfigInput)
	//It is expected that some of configs may be already provided by updated mutation, thus it is not treated as a real error
	if err != nil {
		log.Debugf("Cannot decode config for runtime %s: %s. Json to decode: %s", data.ClusterId, err.Error(), data.Config)
	}

	return oldConfigInput
}

func mapAWSToNewModel(old model.OldAWSProviderConfigInput, workerCidr string) model.AWSProviderConfigInput {
	return model.AWSProviderConfigInput{
		VpcCidr: old.VpcCidr,
		Zones: []*model.AWSZoneInput{
			{
				Name:         old.Zone,
				PublicCidr:   old.PublicCidr,
				InternalCidr: old.InternalCidr,
				WorkerCidr:   workerCidr,
			},
		},
	}
}

func (p *providerConfigMigrator) marshalToJson(obj interface{}) string {
	marshal, err := json.Marshal(obj)

	if err != nil {
		p.errors = append(p.errors, err.Error())
		log.Errorf("error marshalling new provider config: %s. Config to marshal: %s", err.Error(), obj)
		return ""
	}
	return string(marshal)
}

func (p *providerConfigMigrator) updateConfig(session dbconnection.ReadWriteSession, clusterID, config string) {
	err := retry.Do(func() error {
		err := session.UpdateProviderSpecificConfig(clusterID, config)
		return err
	}, retry.Attempts(5))
	if err != nil {
		p.errors = append(p.errors, err.Error())
		log.Errorf("failed to update config for runtime: %s: %s", clusterID, err.Error())
	}
}
