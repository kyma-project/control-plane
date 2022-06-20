package gardener

import (
	"context"
	"fmt"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	gardener_Types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:generate mockery -name=ShootClient
type ShootClient interface {
	List(ctx context.Context, opts metav1.ListOptions) (*gardener_Types.ShootList, error)
}

type ShootProvider struct {
	shootClient ShootClient
}

func NewShootProvider(shootClient ShootClient) ShootProvider {
	return ShootProvider{
		shootClient: shootClient,
	}
}

func (s ShootProvider) Get(runtimeID string, tenant string) (gardener_Types.Shoot, apperrors.AppError) {
	labelSelector := fmt.Sprintf("%s=%s", model.AccountLabel, tenant)

	shoots, err := s.shootClient.List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return gardener_Types.Shoot{}, apperrors.Internal("failed to list shoots: %s", err.Error())
	}

	for _, shoot := range shoots.Items {
		id := shoot.Annotations[runtimeIDAnnotation]

		if id == runtimeID {
			return shoot, nil
		}
	}

	return gardener_Types.Shoot{}, apperrors.Internal("failed to find shoot for Runtime %s", runtimeID)
}
