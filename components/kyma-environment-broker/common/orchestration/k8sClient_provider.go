package orchestration

import (
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

//go:generate mockery -name=K8sClientProvider
type K8sClientProvider interface {
	InitClient(cfg *rest.Config) (k8sClient.Client, error)
}

type k8sClientBuilder struct{}

func NewK8sClientProvider() *k8sClientBuilder {
	return &k8sClientBuilder{}
}

func (c *k8sClientBuilder) InitClient(cfg *rest.Config) (k8sClient.Client, error) {
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
			mapper, err = apiutil.NewDiscoveryRESTMapper(cfg)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "while waiting for client mapper")
		}
	}
	cli, err := k8sClient.New(cfg, k8sClient.Options{Mapper: mapper})
	if err != nil {
		return nil, errors.Wrap(err, "while creating a client")
	}
	return cli, nil
}
