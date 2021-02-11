package postsql

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/pkg/errors"
)

type clsInstances struct {
	postsql.Factory
}

func NewCLSInstances(sess postsql.Factory) *clsInstances {
	return &clsInstances{
		Factory: sess,
	}
}

func (s *clsInstances) FindInstance(globalAccountID string) (internal.CLSInstance, bool, error) {
	sess := s.NewReadSession()
	_, err := sess.GetCLSInstance(globalAccountID)

	return internal.CLSInstance{}, false, errors.Wrapf(err, "needs to be implemented")
}

func (s *clsInstances) InsertInstance(instance internal.CLSInstance) error {
	return errors.New("needs to be implemented")
}

func (s *clsInstances) AddReference(instance internal.CLSInstance, subAccountID string) error {
	return errors.New("needs to be implemented")
}
