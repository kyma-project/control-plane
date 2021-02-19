package postsql

import (
	"errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
)

type clsInstances struct {
	postsql.Factory
}

func NewCLSInstances(sessionFactory postsql.Factory) *clsInstances {
	return &clsInstances{
		Factory: sessionFactory,
	}
}

func (s *clsInstances) FindActiveByGlobalAccountID(globalAccountID string) (*internal.CLSInstance, bool, error) {
	return s.find(func(session postsql.ReadSession) ([]dbmodel.CLSInstanceDTO, dberr.Error) {
		return session.GetCLSInstanceByGlobalAccountID(globalAccountID)
	})
}

func (s *clsInstances) FindByID(clsInstanceID string) (*internal.CLSInstance, bool, error) {
	return s.find(func(session postsql.ReadSession) ([]dbmodel.CLSInstanceDTO, dberr.Error) {
		return session.GetCLSInstanceByID(clsInstanceID)
	})
}

type findFunc func(session postsql.ReadSession) ([]dbmodel.CLSInstanceDTO, dberr.Error)

func (s *clsInstances) find(f findFunc) (*internal.CLSInstance, bool, error) {
	session := s.NewReadSession()
	dtos, err := f(session)
	if err != nil {
		if err.Code() == dberr.CodeNotFound {
			return nil, false, nil
		}

		return nil, false, err
	}

	if len(dtos) == 0 {
		return nil, false, nil
	}

	first := dtos[0]
	model := &internal.CLSInstance{
		Version:         first.Version,
		ID:              first.ID,
		GlobalAccountID: first.GlobalAccountID,
		Region:          first.Region,
		CreatedAt:       first.CreatedAt,
	}
	for _, dto := range dtos {
		model.ReferencedSKRInstanceIDs = append(model.ReferencedSKRInstanceIDs, dto.SKRInstanceID)
	}

	return model, true, nil
}

func (s *clsInstances) Insert(instance internal.CLSInstance) error {
	session, err := s.NewSessionWithinTransaction()
	if err != nil {
		return err
	}
	defer session.RollbackUnlessCommitted()

	if err := session.InsertCLSInstance(dbmodel.CLSInstanceDTO{
		Version:         0,
		ID:              instance.ID,
		GlobalAccountID: instance.GlobalAccountID,
		Region:          instance.Region,
		CreatedAt:       instance.CreatedAt,
	}); err != nil {
		return err
	}

	if len(instance.ReferencedSKRInstanceIDs) != 1 {
		return errors.New("must have a single skr reference")
	}

	if err := session.InsertCLSInstanceReference(dbmodel.CLSInstanceReferenceDTO{
		CLSInstanceID: instance.ID,
		SKRInstanceID: instance.ReferencedSKRInstanceIDs[0],
	}); err != nil {
		return err
	}

	return session.Commit()
}

func (s *clsInstances) Reference(version int, clsInstanceID, skrInstanceID string) error {
	session, err := s.NewSessionWithinTransaction()
	if err != nil {
		return err
	}
	defer session.RollbackUnlessCommitted()

	if err := session.InsertCLSInstanceReference(dbmodel.CLSInstanceReferenceDTO{
		CLSInstanceID: clsInstanceID,
		SKRInstanceID: skrInstanceID,
	}); err != nil {
		return err
	}

	if err := session.IncrementCLSInstanceVersion(version, clsInstanceID); err != nil {
		return err
	}

	return session.Commit()
}

func (s *clsInstances) Unreference(version int, clsInstanceID, skrInstanceID string) error {
	session, err := s.NewSessionWithinTransaction()
	if err != nil {
		return err
	}
	defer session.RollbackUnlessCommitted()

	if err := session.DeleteCLSInstanceReference(dbmodel.CLSInstanceReferenceDTO{
		CLSInstanceID: clsInstanceID,
		SKRInstanceID: skrInstanceID,
	}); err != nil {
		return err
	}

	if err := session.IncrementCLSInstanceVersion(version, clsInstanceID); err != nil {
		return err
	}

	return session.Commit()
}

func (s *clsInstances) MarkAsBeingRemoved(version int, clsInstanceID, skrInstanceID string) error {
	session := s.NewWriteSession()
	return session.MarkCLSInstanceAsBeingRemoved(version, clsInstanceID, skrInstanceID)
}

func (s *clsInstances) Remove(clsInstanceID string) error {
	session := s.NewWriteSession()
	return session.DeleteCLSInstance(clsInstanceID)
}
