package api

import (
	"context"

	"github.com/kyma-project/control-plane/components/provisioner/internal/api/middlewares"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
)

//go:generate mockery -name=TenantUpdater
type TenantUpdater interface {
	GetTenant(ctx context.Context) (string, apperrors.AppError)
	GetAndUpdateTenant(runtimeID string, ctx context.Context) apperrors.AppError
}

type updater struct {
	readWriteSession dbsession.ReadWriteSession
}

func NewTenantUpdater(readWriteSession dbsession.ReadWriteSession) TenantUpdater {
	return &updater{
		readWriteSession: readWriteSession,
	}
}

func (u *updater) GetTenant(ctx context.Context) (string, apperrors.AppError) {
	tenant, ok := ctx.Value(middlewares.Tenant).(string)
	if !ok || tenant == "" {
		return "", apperrors.BadRequest("tenant header is empty")
	}

	return tenant, nil
}

func (u *updater) GetAndUpdateTenant(runtimeID string, ctx context.Context) apperrors.AppError {
	tenant, err := u.GetTenant(ctx)
	if err != nil {
		return err
	}
	dbTenant, dberr := u.readWriteSession.GetTenant(runtimeID)
	if dberr != nil {
		return dberr
	}

	if tenant != dbTenant {
		dberr := u.readWriteSession.UpdateTenant(runtimeID, tenant)
		if dberr != nil {
			return dberr
		}
	}
	return nil
}
