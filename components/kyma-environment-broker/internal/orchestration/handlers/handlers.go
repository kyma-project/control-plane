package handlers

import (
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	AttachRoutes(router *mux.Router)
}

type handler struct {
	handlers []Handler
}

func NewOrchestrationHandler(db storage.BrokerStorage, kymaQueue *process.Queue, clusterQueue *process.Queue, defaultMaxPage int, log logrus.FieldLogger) Handler {
	return &handler{
		handlers: []Handler{
			NewKymaHandler(db.Orchestrations(), kymaQueue, log),
			NewClusterHandler(db.Orchestrations(), clusterQueue, log),
			NewOrchestrationStatusHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), kymaQueue, clusterQueue, defaultMaxPage, log),
		},
	}
}

func (h *handler) AttachRoutes(router *mux.Router) {
	for _, handler := range h.handlers {
		handler.AttachRoutes(router)
	}
}

func validateTarget(spec orchestration.TargetSpec) error {
	if spec.Include == nil || len(spec.Include) == 0 {
		return errors.New("targets.include array must be not empty")
	}
	return nil
}

func defaultOrchestrationStrategy(spec *orchestration.StrategySpec) {
	if spec.Parallel.Workers == 0 {
		spec.Parallel.Workers = 1
	}

	switch spec.Type {
	case orchestration.ParallelStrategy:
	default:
		spec.Type = orchestration.ParallelStrategy
	}

	switch spec.Schedule {
	case orchestration.MaintenanceWindow:
	case orchestration.Immediate:
	default:
		spec.Schedule = orchestration.Immediate
	}
}
