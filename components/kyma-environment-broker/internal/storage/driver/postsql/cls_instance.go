package postsql

import (
	"database/sql"

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

	var references []string
	for _, dto := range dtos {
		if dto.SKRInstanceID.Valid {
			references = append(references, dto.SKRInstanceID.String)
		}
	}

	return internal.NewCLSInstance(
		first.GlobalAccountID,
		first.Region,
		internal.WithID(first.ID),
		internal.WithVersion(first.Version),
		internal.WithCreatedAt(first.CreatedAt),
		internal.WithReferences(references...),
		internal.WithBeingRemovedBy(first.RemovedBySKRInstanceID.String),
	), true, nil
}

func (s *clsInstances) Insert(instance internal.CLSInstance) error {
	session, err := s.NewSessionWithinTransaction()
	if err != nil {
		return err
	}
	defer session.RollbackUnlessCommitted()

	if err := session.InsertCLSInstance(dbmodel.CLSInstanceDTO{
		Version:         0,
		ID:              instance.ID(),
		GlobalAccountID: instance.GlobalAccountID(),
		Region:          instance.Region(),
		CreatedAt:       instance.CreatedAt(),
	}); err != nil {
		return err
	}

	if err := session.InsertCLSInstanceReference(dbmodel.CLSInstanceReferenceDTO{
		CLSInstanceID: instance.ID(),
		SKRInstanceID: instance.References()[0],
	}); err != nil {
		return err
	}

	return session.Commit()
}

func (s *clsInstances) Update(instance internal.CLSInstance) error {
	session, err := s.NewSessionWithinTransaction()
	if err != nil {
		return err
	}
	defer session.RollbackUnlessCommitted()

	for _, ev := range instance.Events() {
		if referencedEvent, ok := ev.(internal.CLSInstanceReferencedEvent); ok {
			if err := session.InsertCLSInstanceReference(dbmodel.CLSInstanceReferenceDTO{
				CLSInstanceID: instance.ID(),
				SKRInstanceID: referencedEvent.SKRInstanceID,
			}); err != nil {
				return err
			}
		}

		if unreferencedEvent, ok := ev.(internal.CLSInstanceUnreferencedEvent); ok {
			if err := session.DeleteCLSInstanceReference(dbmodel.CLSInstanceReferenceDTO{
				CLSInstanceID: instance.ID(),
				SKRInstanceID: unreferencedEvent.SKRInstanceID,
			}); err != nil {
				return err
			}
		}
	}

	dto := dbmodel.CLSInstanceDTO{
		Version:         instance.Version(),
		ID:              instance.ID(),
		GlobalAccountID: instance.GlobalAccountID(),
		Region:          instance.Region(),
		CreatedAt:       instance.CreatedAt(),
	}

	if instance.IsBeingRemoved() {
		dto.RemovedBySKRInstanceID = sql.NullString{String: instance.BeingRemovedBy(), Valid: true}
	}

	if err = session.UpdateCLSInstance(dto); err != nil {
		return err
	}

	return session.Commit()
}

func (s *clsInstances) Delete(clsInstanceID string) error {
	session := s.NewWriteSession()
	return session.DeleteCLSInstance(clsInstanceID)
}

func (s *clsInstances) GetCLSInstanceStatsByRegion(region string) (int, error) {
	session := s.NewReadSession()
	return session.GetClsInstanceCountByRegion(region)
}