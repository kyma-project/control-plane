package gardener

import (
	"context"
	"fmt"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type KubeconfigProvider struct {
	secretsClient v12.SecretInterface
}

func NewKubeconfigProvider(secretsClient v12.SecretInterface) KubeconfigProvider {
	return KubeconfigProvider{
		secretsClient: secretsClient,
	}
}

func (kp KubeconfigProvider) FetchRaw(shootName string) ([]byte, error) {
	secret, err := kp.secretsClient.Get(context.Background(), fmt.Sprintf("%s.kubeconfig", shootName), v1.GetOptions{})
	if err != nil {
		return nil, util.K8SErrorToAppError(err).Append("error fetching kubeconfig").SetComponent(apperrors.ErrGardenerClient)
	}

	kubeconfig, found := secret.Data["kubeconfig"]
	if !found {
		return nil, util.K8SErrorToAppError(err).Append("error fetching kubeconfig: secret does not contain kubeconfig").SetComponent(apperrors.ErrGardenerClient)
	}

	return kubeconfig, nil
}
