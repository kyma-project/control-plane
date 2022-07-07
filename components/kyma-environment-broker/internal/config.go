package internal

import (
	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigReader struct {
	ctx       context.Context
	k8sClient client.Client
}

type ConfigForPlan struct {
	AdditionalComponents []runtime.KymaComponent `json:"additional-components"`
}

func NewConfigReader(ctx context.Context, k8sClient client.Client) *ConfigReader {
	return &ConfigReader{
		ctx:       ctx,
		k8sClient: k8sClient,
	}
}
