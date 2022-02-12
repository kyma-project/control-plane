package k8s

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
)

//go:generate mockery -name=K8sClientProvider
type K8sClientProvider interface {
	CreateK8SClient(kubeconfigRaw string) (kubernetes.Interface, apperrors.AppError)
}

type k8sClientBuilder struct{}

func NewK8sClientProvider() K8sClientProvider {
	return &k8sClientBuilder{}
}

func (c *k8sClientBuilder) CreateK8SClient(kubeconfigRaw string) (kubernetes.Interface, apperrors.AppError) {
	k8sConfig, err := ParseToK8sConfig([]byte(kubeconfigRaw))

	if err != nil {
		return nil, util.K8SErrorToAppError(errors.Wrap(err, "failed to parse kubeconfig"))
	}

	coreClientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, util.K8SErrorToAppError(errors.Wrap(err, "failed to create k8s core client"))
	}

	return coreClientset, nil
}
