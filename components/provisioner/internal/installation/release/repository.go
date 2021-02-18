package release

import (
	"github.com/gocraft/dbr/v2"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid"
	"github.com/lib/pq"
)

const (
	UniqueConstraintViolationError = "23505"
)

//go:generate mockery -name=Repository
type Repository interface {
	GetReleaseByVersion(version string) (model.Release, dberrors.Error)
	ReleaseExists(version string) (bool, dberrors.Error)
	SaveRelease(artifacts model.Release) (model.Release, dberrors.Error)
}

func NewReleaseRepository(connection *dbr.Connection, generator uuid.UUIDGenerator) *releaseRepository {
	return &releaseRepository{
		connection: connection,
		generator:  generator,
	}
}

type releaseRepository struct {
	connection *dbr.Connection
	generator  uuid.UUIDGenerator
}

func (r releaseRepository) GetReleaseByVersion(version string) (model.Release, dberrors.Error) {
	session := r.connection.NewSession(nil)

	var release model.Release

	err := session.
		Select("id", "version", "tiller_yaml", "installer_yaml").
		From("kyma_release").
		Where(dbr.Eq("version", version)).
		LoadOne(&release)

	if err != nil {
		if err == dbr.ErrNotFound {
			return model.Release{}, dberrors.NotFound("Kyma release for version %s not found", version)
		}
		return model.Release{}, dberrors.Internal("Failed to get Kyma release for version %s: %s", version, err.Error())
	}

	return release, nil
}

func (r releaseRepository) ReleaseExists(version string) (bool, dberrors.Error) {
	_, err := r.GetReleaseByVersion(version)

	if err != nil {
		if err.Code() == dberrors.CodeNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r releaseRepository) SaveRelease(artifacts model.Release) (model.Release, dberrors.Error) {
	artifacts.Id = r.generator.New()
	session := r.connection.NewSession(nil)

	_, err := session.InsertInto("kyma_release").
		Columns("id", "version", "tiller_yaml", "installer_yaml").
		Record(artifacts).
		Exec()

	if err != nil {
		// The artifacts could be saved by different thread before
		psqlErr, converted := err.(*pq.Error)
		if converted && psqlErr.Code == UniqueConstraintViolationError {
			return model.Release{}, dberrors.AlreadyExists("Artifacts for version %s already exist: %s", artifacts.Version, err.Error())
		}
		return model.Release{}, dberrors.Internal("Failed to save Kyma release artifacts for version %s: %s", artifacts.Version, err.Error())
	}

	return artifacts, nil
}
