package gardener

import (
	"context"
	"fmt"
	"time"

	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type adminKubeconfig = func(context.Context, client.Object, client.Object, ...client.SubResourceCreateOption) error

type KubeconfigProvider struct {
	secretsClient v12.SecretInterface
	request       adminKubeconfig
	logger        log.FieldLogger
}

func NewKubeconfigProvider(secretsClient v12.SecretInterface, gardener adminKubeconfig) KubeconfigProvider {
	return KubeconfigProvider{
		secretsClient: secretsClient,
		request:       gardener,
		logger:        log.New(),
	}
}

func (kp KubeconfigProvider) FetchRaw(ctx context.Context, shoot gardener_types.Shoot) ([]byte, error) {

	expiration := 8 * time.Hour
	expirationSeconds := int64(expiration.Seconds())
	adminKubeconfigRequest := authenticationv1alpha1.AdminKubeconfigRequest{
		Spec: authenticationv1alpha1.AdminKubeconfigRequestSpec{
			ExpirationSeconds: &expirationSeconds,
		},
	}

	err := kp.request(ctx, &shoot, &adminKubeconfigRequest)
	if err == nil {
		kp.logger.Debug("new admin kubeconfig created")
		return adminKubeconfigRequest.Status.Kubeconfig, nil
	}

	kp.logger.Info("unable to create new admin kubeconfig: %s", err)

	name := fmt.Sprintf("%s.kubeconfig", shoot.Name)
	secret, err := kp.secretsClient.Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return nil, util.K8SErrorToAppError(err).Append("error fetching kubeconfig").SetComponent(apperrors.ErrGardenerClient)
	}

	kubeconfig, found := secret.Data["kubeconfig"]
	if !found {
		return nil, util.K8SErrorToAppError(err).Append("error fetching kubeconfig: secret does not contain kubeconfig").SetComponent(apperrors.ErrGardenerClient)
	}

	return kubeconfig, nil
}
