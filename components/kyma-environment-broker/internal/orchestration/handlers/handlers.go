package handlers

import (
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	AttachRoutes(router *mux.Router)
}

type handler struct {
	handlers []Handler
}

func NewOrchestrationHandler(db storage.BrokerStorage, kymaQueue *process.Queue, defaultMaxPage int, log logrus.FieldLogger) Handler {
	return &handler{
		handlers: []Handler{
			NewKymaHandler(db.Orchestrations(), kymaQueue, log),
			NewOrchestrationStatusHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), defaultMaxPage, log),
		},
	}
}

func (h *handler) AttachRoutes(router *mux.Router) {
	for _, handler := range h.handlers {
		handler.AttachRoutes(router)
	}
}
