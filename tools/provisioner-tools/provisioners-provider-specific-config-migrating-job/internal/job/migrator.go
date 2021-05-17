package job

import (
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/model"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dbsession"
)

type ProviderConfigMigrator interface {
	Do() error
}

type providerConfigMigrator struct {
	dbsFactory dbsession.Factory
}

func (p providerConfigMigrator) Do() error {
	session := p.dbsFactory.NewReadWriteSession()

	configs, err := session.GetProviderSpecificConfigsByProvider(model.AWS)

	if err != nil {
		return err
	}

	for _, config := range configs {

		/*
			1. Unmarshal
			2. Map to new model
			3. Update
		*/
	}

	return nil
}
