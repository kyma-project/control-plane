package handlers

import (
	"fmt"
	"time"

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

// ValidateDeprecatedParameters cheks if `maintenanceWindow` parameter is used as schedule.
func ValidateDeprecatedParameters(params orchestration.Parameters) error {
	if params.Strategy.Schedule == string(orchestration.MaintenanceWindow) {
		return fmt.Errorf("{\"strategy\":{\"schedule\": \"maintenanceWindow\"} is deprecated use {\"strategy\":{\"MaintenanceWindow\": true} instead")
	}
	return nil
}

// ValidateScheduleParameter cheks if the schedule parameter is valid.
func ValidateScheduleParameter(params *orchestration.Parameters) error {
	switch params.Strategy.Schedule {
	case "immediate":
	case "now":
		params.Strategy.ScheduleTime = time.Now()
	default:
		parsedTime, err := time.Parse(time.RFC3339, params.Strategy.Schedule)
		if err == nil {
			params.Strategy.ScheduleTime = parsedTime
		} else {
			return fmt.Errorf("the schedule filed does not contain 'imediate'/'now' nor is a date: %w", err)
		}
	}
	return nil
}
