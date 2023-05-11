package btpmgrcreds

import (
	"context"
	"net/http"

	"github.com/kyma-project/runtime-watcher/listener/pkg/event"
	"github.com/kyma-project/runtime-watcher/listener/pkg/types"
	"github.com/sirupsen/logrus"
)

type Watcher struct {
	ctx                context.Context
	listener           *event.SKREventListener
	logs               *logrus.Logger
	btpOperatorManager *Manager
}

func NewWatcher(ctx context.Context, listenerAddr, componentName string, btpOperatorManager *Manager, logs *logrus.Logger) *Watcher {
	noVerify := func(r *http.Request, watcherEvtObject *types.WatchEvent) error {
		return nil
	}
	listener, _ := event.RegisterListenerComponent(listenerAddr, componentName, noVerify)
	return &Watcher{
		ctx:                ctx,
		listener:           listener,
		btpOperatorManager: btpOperatorManager,
		logs:               logs,
	}
}

func (s *Watcher) ReactOnSkrEvent() {
	go func() {
		for {
			select {
			case response := <-s.listener.ReceivedEvents:
				kymaName := response.Object.GetName()
				s.logs.Infof("event received for: %s", kymaName)
				instance, err := s.btpOperatorManager.MatchInstance(kymaName)
				if err != nil {
					s.logs.Errorf("while trying to match instance for kyma name : %s, %s", kymaName, err)
					continue
				}
				updated, err := s.btpOperatorManager.ReconcileSecretForInstance(instance)
				if err != nil {
					s.logs.Errorf("while trying to update for instance %s with kyma name : %s, %s", instance.InstanceID, kymaName, err)
					continue
				}
				if updated {
					s.logs.Infof("instance id: %s updated kyma %s with success", instance.InstanceID, kymaName)
				}
			case <-s.ctx.Done():
				s.logs.Info("runtime watcher: context closed")
				return
			}
		}
	}()

	if err := s.listener.Start(s.ctx); err != nil {
		s.logs.Errorf("cannot start listener: %s", err.Error())
	}
}
