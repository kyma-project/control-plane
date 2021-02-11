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

func (s *clsInstances) FindInstance(globalAccountID string) (*internal.CLSInstance, bool, error) {
	sess := s.NewReadSession()
	dto, err := sess.GetCLSInstance(globalAccountID)

	switch {
	case err == nil:
		return &internal.CLSInstance{
			ID:              dto.ID,
			GlobalAccountID: dto.GlobalAccountID,
			CreatedAt:       dto.CreatedAt,
		}, true, nil
	case err.Code() == dberr.CodeNotFound:
		return nil, false, nil
	default:
		return nil, false, err
	}
}

func (s *clsInstances) InsertInstance(instance internal.CLSInstance) error {
	sess := s.NewWriteSession()
	return sess.InsertCLSInstance(dbmodel.CLSInstanceDTO{
		ID:              instance.ID,
		GlobalAccountID: instance.GlobalAccountID,
		CreatedAt:       instance.CreatedAt,
	})
}
