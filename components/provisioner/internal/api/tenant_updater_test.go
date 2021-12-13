package api

import (
	"context"
	"github.com/kyma-project/control-plane/components/provisioner/internal/api/middlewares"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
