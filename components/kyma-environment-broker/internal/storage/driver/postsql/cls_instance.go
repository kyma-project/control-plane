package postsql

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
)

type clsInstances struct {
	postsql.Factory
}

func NewCLSInstances(sess postsql.Factory) *clsInstances {
	return &clsInstances{
		Factory: sess,
	}
}

func (s *clsInstances) FindInstanceByName(name, region string) (internal.CLSInstance, bool, error) {
	sess := s.NewReadSession()
	dto, err := sess.GetCLSTenant(name, region)

	switch {
	case err == nil:
		return internal.CLSInstance{
			CreatedAt: dto.CreatedAt,
			Name:      dto.Name,
			Region:    dto.Region,
			ID:        dto.ID,
		}, true, nil
	case err.Code() == dberr.CodeNotFound:
		return internal.CLSInstance{}, false, nil
	default:
		return internal.CLSInstance{}, false, err
	}
}

func (s *clsInstances) InsertInstance(tenant internal.CLSInstance) error {
	sess := s.NewWriteSession()
	return sess.InsertCLSInstance(dbmodel.CLSTenantDTO{
		Name:      tenant.Name,
		Region:    tenant.Region,
		CreatedAt: tenant.CreatedAt,
		ID:        tenant.ID,
	})
}