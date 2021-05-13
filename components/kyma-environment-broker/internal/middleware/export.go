package middleware

import (
	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func AddRegionToCtx(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, requestRegionKey, region)
}

func AddProviderToCtx(ctx context.Context, provider internal.CloudProvider) context.Context {
	return context.WithValue(ctx, requestProviderKey, provider)
}
