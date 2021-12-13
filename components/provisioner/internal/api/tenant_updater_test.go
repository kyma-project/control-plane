package api

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/api/middlewares"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantUpdater_GetTenant(t *testing.T) {
	t.Run("should extract tenant from context", func(t *testing.T) {
		tenant := "tenant"
		ctx := context.WithValue(context.Background(), middlewares.Tenant, tenant)

		tenantUpdater := NewTenantUpdater(nil)

		ctxTenant, appError := tenantUpdater.GetTenant(ctx)
		require.NoError(t, appError)
		assert.Equal(t, tenant, ctxTenant)
	})

	t.Run("should return error when tenant header is empty", func(t *testing.T) {
		ctx := context.Background()

		tenantUpdater := NewTenantUpdater(nil)

		_, appError := tenantUpdater.GetTenant(ctx)
		require.Error(t, appError)
	})
}

func TestTenantUpdater_GetAndUpdateTenant(t *testing.T) {
	t.Run("should update tenant when differs from db tenant", func(t *testing.T) {
		newTenant := "tenant"
		dbTenant := "tenet"
		runtimeId := "runtimeID"
		ctx := context.WithValue(context.Background(), middlewares.Tenant, newTenant)

		rwsMock := &mocks.ReadWriteSession{}
		tenantUpdater := NewTenantUpdater(rwsMock)

		rwsMock.On("GetTenant", ctx).Return(dbTenant, nil)
		rwsMock.On("UpdateTenant", runtimeId, newTenant).Return(nil)

		err := tenantUpdater.GetAndUpdateTenant(runtimeId, ctx)
		require.NoError(t, err)
		rwsMock.AssertExpectations(t)
	})
	t.Run("should not update tenant when equal to db tenant", func(t *testing.T) {
		newTenant := "tenant"
		dbTenant := "tenant"
		runtimeId := "runtimeID"
		ctx := context.WithValue(context.Background(), middlewares.Tenant, newTenant)

		rwsMock := &mocks.ReadWriteSession{}
		tenantUpdater := NewTenantUpdater(rwsMock)

		rwsMock.On("GetTenant", ctx).Return(dbTenant, nil)

		err := tenantUpdater.GetAndUpdateTenant(runtimeId, ctx)
		require.NoError(t, err)
		rwsMock.AssertExpectations(t)
	})
}
