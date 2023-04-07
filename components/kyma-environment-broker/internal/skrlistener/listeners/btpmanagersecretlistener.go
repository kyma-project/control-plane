package skrlisteners

import (
	"context"
	"fmt"
	btpoperatorcredentials "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/btpmanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/skrlistener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/runtime-watcher/listener/pkg/event"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BtpManagerSecretListener struct {
	skrlistener.ListenerConfig
	instances storage.Instances
	handler   btpoperatorcredentials.BTPOperatorHandler
}

func NewBtpManagerSecretListener(instanceDb storage.Instances, ctx context.Context, logs *logrus.Logger, listenerAddr, componentName string, verifyFunc event.Verify) *BtpManagerSecretListener {
	return &BtpManagerSecretListener{
		*skrlistener.NewListenerConfig(ctx, logs, listenerAddr, componentName, verifyFunc),
		instanceDb,
		btpoperatorcredentials.BTPOperatorHandler{},
	}
}

var _ skrlistener.Listener = (*BtpManagerSecretListener)(nil)

func (s *BtpManagerSecretListener) ReactOnSkrEvent() {
	listener, _ := event.RegisterListenerComponent(s.ListenerAddr, s.ComponentName, s.VerifyFunc)

	go func() {
		for {
			select {
			case response := <-listener.ReceivedEvents:
				s.Logger.Info("watcher event received....")
				s.Logger.Info(fmt.Sprintf("%v", response.Object))
				instance, err := s.instances.GetByID("")
				if err != nil {
					log.Fatal(err)
					continue
				}
				restCfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(instance.Parameters.Parameters.Kubeconfig))
				if err != nil {
					log.Fatal(err)
					continue
				}
				k8sClient, err := client.New(restCfg, client.Options{})
				secret := s.handler.PrepareSecret(instance.Parameters.ErsContext.SMOperatorCredentials, "")
				secretDiff := btpoperatorcredentials.CompareContentFromSkr(secret, response.Object)
				if secretDiff {
					if err := s.handler.CreateOrUpdateSecret(k8sClient, secret, s.Logger); err != nil {
						s.Logger.Fatalf("%v", err)
					}
				}
			case <-s.Ctx.Done():
				s.Logger.Info("context closed")
				return
			}
		}
	}()

	if err := listener.Start(s.Ctx); err != nil {
		s.Logger.Errorf("cannot start listener: %s", err.Error())
	}
}
