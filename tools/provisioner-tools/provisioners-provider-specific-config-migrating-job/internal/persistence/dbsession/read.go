package dbsession

import (
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dberrors"

	"github.com/gocraft/dbr/v2"
)

type readSession struct {
	session *dbr.Session
}

type ProviderConfig struct {
	Id     string
	Config string
}

func (r readSession) GetProviderSpecificConfigsByProvider(provider string) ([]ProviderConfig, dberrors.Error) {
	providerConfigs := make([]ProviderConfig, 0)

	m, err := r.session.
		Select("Id", "provider_specific_config").
		From("gardener_config").
		Where(dbr.Eq("provider", provider)).
		Load(&providerConfigs)

	if err != nil {
		return nil, dberrors.Internal("Failed to get configs for provider: %s", provider)
	}

	if m == 0 {
		return nil, dberrors.NotFound("Clusters with provider: %s, not found", provider)
	}

	return providerConfigs, nil
}
