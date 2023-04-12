package skrlisteners

import (
	"context"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	btpoperatorcredentials "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/btpmanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/runtime-watcher/listener/pkg/event"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BtpManagerSecretListener struct {
	ListenerConfig
	instances storage.Instances
	handler   btpoperatorcredentials.BTPOperatorHandler
}

func NewBtpManagerSecretListener(ctx context.Context, instanceDb storage.Instances, listenerAddr, componentName string, verifyFunc event.Verify, logs *logrus.Logger) *BtpManagerSecretListener {
	return &BtpManagerSecretListener{
		*NewListenerConfig(ctx, logs, listenerAddr, componentName, verifyFunc),
		instanceDb,
		btpoperatorcredentials.BTPOperatorHandler{},
	}
}

var _ Listener = (*BtpManagerSecretListener)(nil)

func (s *BtpManagerSecretListener) ReactOnSkrEvent() {
	listener, _ := event.RegisterListenerComponent(s.ListenerAddr, s.ComponentName, s.VerifyFunc)

	go func() {
		for {
			select {
			case response := <-listener.ReceivedEvents:
				s.Logger.Info("watcher event received....")
				skrId := response.Object.GetName()
				instance, err := s.instances.GetByID(skrId)
				if err != nil {
					log.Fatal(err)
					continue
				}
				_, err = s.ReconcileSecretForInstance(instance)
				if err != nil {
					s.Logger.Fatalf("%s", err)
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

func (s *BtpManagerSecretListener) GetNeededDataFromInstance(instance *internal.Instance) (skrK8sCfg []byte, credentials *internal.ServiceManagerOperatorCredentials) {
	skrK8sCfg = []byte(instance.Parameters.Parameters.Kubeconfig)
	credentials = instance.Parameters.ErsContext.SMOperatorCredentials
	return
}

func (s *BtpManagerSecretListener) GetSkrK8sCfgClient(skrK8sCfg []byte) (*client.Client, error) {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(skrK8sCfg)
	if err != nil {
		return nil, err
	}
	k8sClient, err := client.New(restCfg, client.Options{})
	if err != nil {
		return nil, err
	}

	return &k8sClient, err
}

func (s *BtpManagerSecretListener) Reconcile() {
	i, err := s.instances.FindAllInstancesForRuntimes(nil)
	if err != nil {
		panic(err)
	}

	var instanceWithinErrors []*internal.Instance
	for _, v := range i {
		_, err := s.ReconcileSecretForInstance(&v)
		if err != nil {
			s.Logger.Fatalf("%s", err)
			instanceWithinErrors = append(instanceWithinErrors, &v)
		}
	}
}

func (s *BtpManagerSecretListener) ReconcileSecretForInstance(v *internal.Instance) (bool, error) {
	skrUpdated := false
	skrK8sCfg, credentials := s.GetNeededDataFromInstance(v)
	k8sClient, err := s.GetSkrK8sCfgClient(skrK8sCfg)
	if err != nil {
		log.Fatal(err)
		return skrUpdated, err
	}

	expectedSecret := s.handler.PrepareSecret(credentials, "")

	currentSecret := &v1.Secret{}
	err = (*k8sClient).Get(context.Background(), client.ObjectKey{Name: btpoperatorcredentials.BtpManagerSecretName, Namespace: btpoperatorcredentials.BtpManagerSecretName}, currentSecret)
	if err != nil && errors.IsNotFound(err) {
		if err := s.handler.CreateOrUpdateSecret(*k8sClient, expectedSecret, s.Logger); err != nil {
			return skrUpdated, err
		}
		skrUpdated = true
		return skrUpdated, nil
	} else if err != nil {
		return skrUpdated, err
	}

	secretDifferent, err := btpoperatorcredentials.CompareSecretsData(expectedSecret, currentSecret)
	if err != nil {
		return skrUpdated, err
	}

	if secretDifferent {
		if err := s.handler.CreateOrUpdateSecret(*k8sClient, expectedSecret, s.Logger); err != nil {
			s.Logger.Fatalf("%v", err)
			return skrUpdated, err
		}
		skrUpdated = true
		return skrUpdated, nil
	}

	return skrUpdated, nil
}
