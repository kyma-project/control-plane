package gardener

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	gardenerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type KubeconfigProvider struct {
	secretsClient          v12.SecretInterface
	adminKubeconfigRequest AdminKubeconfigRequester
	gardenerShootClient    Client
	logger                 log.FieldLogger
}

func NewKubeconfigProvider(gardenerShootClient Client, adminKubeconfigRequest AdminKubeconfigRequester, secretsClient v12.SecretInterface) KubeconfigProvider {
	return KubeconfigProvider{
		secretsClient:          secretsClient,
		adminKubeconfigRequest: adminKubeconfigRequest,
		gardenerShootClient:    gardenerShootClient,
		logger:                 log.New(),
	}
}

type AdminKubeconfigRequester interface {
	Create(ctx context.Context, obj gardenerClient.Object, subResource gardenerClient.Object, opts ...gardenerClient.SubResourceCreateOption) error
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

func (kp KubeconfigProvider) FetchFromGardener(shootName string) ([]byte, error) {
	shoot, err := kp.gardenerShootClient.Get(context.Background(), shootName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	expiration := 8 * time.Hour
	expirationSeconds := int64(expiration.Seconds())
	adminKubeconfigRequest := authenticationv1alpha1.AdminKubeconfigRequest{
		Spec: authenticationv1alpha1.AdminKubeconfigRequestSpec{
			ExpirationSeconds: &expirationSeconds,
		},
	}

	err = kp.adminKubeconfigRequest.Create(context.Background(), shoot, &adminKubeconfigRequest)
	if err != nil {
		kp.logger.Errorf("failed to create dynamic kubeconfig: %s", err)
		return nil, err
	}

	return adminKubeconfigRequest.Status.Kubeconfig, nil
}
