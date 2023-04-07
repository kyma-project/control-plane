package skrlistener

import (
	"context"
	"github.com/kyma-project/runtime-watcher/listener/pkg/event"
	"github.com/sirupsen/logrus"
)

type Listener interface {
	ReactOnSkrEvent()
}

type ListenerConfig struct {
	Ctx           context.Context
	Logger        *logrus.Logger
	ListenerAddr  string
	ComponentName string
	VerifyFunc    event.Verify
}

func NewListenerConfig(ctx context.Context, logs *logrus.Logger, listenerAddr, componentName string, verifyFunc event.Verify) *ListenerConfig {
	return &ListenerConfig{
		Logger:        logs,
		Ctx:           ctx,
		ListenerAddr:  listenerAddr,
		ComponentName: componentName,
		VerifyFunc:    verifyFunc,
	}
}
