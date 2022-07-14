package postsql

import (
	dbr "github.com/gocraft/dbr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
)

type GenericRead[T any] struct {
	Session   *dbr.Session
	IdName    string
	TableName string

	NewItem func() T
}

func (r GenericRead[T]) GetInstanceByID(instanceID string) (T, dberr.Error) {
	var instance T

	err := r.Session.
		Select("*").
		From(r.TableName).
		Where(dbr.Eq(r.IdName, instanceID)).
		LoadOne(&instance)

	if err != nil {
		if err == dbr.ErrNotFound {
			return r.NewItem(), dberr.NotFound("Cannot find %s for %s:'%s'", r.TableName, r.IdName, instanceID)
		}
		return r.NewItem(), dberr.Internal("Failed to get %s: %s", r.TableName, err)
	}

	return instance, nil
}
