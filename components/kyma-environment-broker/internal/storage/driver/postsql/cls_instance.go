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

func (s *clsInstances) FindInstance(name string) (internal.CLSInstance, bool, error) {
	sess := s.NewReadSession()
	dto, err := sess.GetCLSInstance(name)

	switch {
	case err == nil:
		return internal.CLSInstance{
			CreatedAt: dto.CreatedAt,
			Name:      dto.Name,
			ID:        dto.ID,
		}, true, nil
	case err.Code() == dberr.CodeNotFound:
		return internal.CLSInstance{}, false, nil
	default:
		return internal.CLSInstance{}, false, err
	}
}

func (s *clsInstances) InsertInstance(instance internal.CLSInstance) error {
	sess := s.NewWriteSession()
	return sess.InsertCLSInstance(dbmodel.CLSInstanceDTO{
		Name:      instance.Name,
		CreatedAt: instance.CreatedAt,
		ID:        instance.ID,
	})
}
