package dbconnection

import (
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dberrors"

	"github.com/gocraft/dbr/v2"
)

type readSession struct {
	session *dbr.Session
}

type ProviderData struct {
	Id         string
	ClusterId  string
	WorkerCidr string
	Config     string
}

func (r readSession) GetProviderSpecificConfigsByProvider(provider string) ([]ProviderData, dberrors.Error) {
	providerConfigs := make([]ProviderData, 0)

	m, err := r.session.
		Select("id", "cluster_id", "worker_cidr", "provider_specific_config").
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
