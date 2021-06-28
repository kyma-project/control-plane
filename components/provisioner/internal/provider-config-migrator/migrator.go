//TODO: Remove this package after schema migration
package migrator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

type ProviderConfigMigrator interface {
	Do() error
}

type providerConfigMigrator struct {
	dbsFactory      dbsession.Factory
	errorsThreshold int
	errors          []string
}

func NewProviderConfigMigrator(dbsFactory dbsession.Factory, errorsThreshold int) ProviderConfigMigrator {
	return &providerConfigMigrator{
		dbsFactory:      dbsFactory,
		errorsThreshold: errorsThreshold,
		errors:          make([]string, 0),
	}
}

func (p *providerConfigMigrator) Do() error {
	session := p.dbsFactory.NewReadWriteSession()

	data, dberr := session.GetProviderSpecificConfigsByProvider(model.AWS)

	if dberr != nil {
		if dberr.Code() == dberrors.CodeNotFound {
			log.Info("Could not find Gardener configs for aws. Skipping")
			return nil
		}
		return dberr
	}

	for _, d := range data {
		oldConfig, err := p.decodeAWSConfig(d)
		if err != nil {
			continue
		}
		log.Infof("Migrating Provider Config for runtime: %s", d.ClusterId)
		newConfig := mapAWSToNewModel(oldConfig, d.WorkerCidr)
		jsonConfig := p.marshalToJson(newConfig)
		p.updateConfig(session, d.Id, d.ClusterId, jsonConfig)
		if len(p.errors) > p.errorsThreshold {
			log.Error("Too many errors!")
			return fmt.Errorf(strings.Join(p.errors, "\n"))
		}
	}
	return nil
}

func (p *providerConfigMigrator) decodeAWSConfig(data dbsession.ProviderData) (model.SingleZoneAWSProviderConfigInput, error) {
	var oldConfigInput model.SingleZoneAWSProviderConfigInput

	err := util.DecodeJson(data.ProviderSpecificConfig, &oldConfigInput)
	//It is expected that some of configs may be already provided by updated mutation, thus it is not treated as a real error
	if err != nil {
		log.Debugf("Cannot decode config for runtime %s: %s. Json to decode: %s", data.ClusterId, err.Error(), data.ProviderSpecificConfig)
		return model.SingleZoneAWSProviderConfigInput{}, err
	}

	return oldConfigInput, nil
}

func mapAWSToNewModel(old model.SingleZoneAWSProviderConfigInput, workerCidr string) gqlschema.AWSProviderConfigInput {
	return gqlschema.AWSProviderConfigInput{
		VpcCidr: old.VpcCidr,
		AwsZones: []*gqlschema.AWSZoneInput{
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

func (p *providerConfigMigrator) updateConfig(session dbsession.ReadWriteSession, id, clusterID, config string) {
	err := retry.Do(func() error {
		err := session.UpdateProviderSpecificConfig(id, config)
		return err
	}, retry.Attempts(5))
	if err != nil {
		p.errors = append(p.errors, err.Error())
		log.Errorf("failed to update config for runtime: %s: %s", clusterID, err.Error())
	}
}
