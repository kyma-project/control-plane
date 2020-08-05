package cloudprovider

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/subscription-cleanup-job/internal/model"
	"github.com/pkg/errors"
)

type ResourceCleaner interface {
	Do() error
}

func New(hyperscalerType model.HyperscalerType, secretData map[string][]byte) (ResourceCleaner, error) {
	switch hyperscalerType {
	case model.GCP:
		{
			return NewGCPeResourcesCleaner(secretData), nil
		}
	case model.Azure:
		{
			return NewAzureResourcesCleaner(secretData), nil
		}
	default:
		return nil, errors.New(fmt.Sprintf("unknown hyperscaler type"))
	}
}
