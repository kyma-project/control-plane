package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type kymaHandler struct {
	orchestrations storage.Orchestrations
	queue          *process.Queue
	converter      Converter
	log            logrus.FieldLogger
}

func NewKymaHandler(orchestrations storage.Orchestrations, q *process.Queue, log logrus.FieldLogger) *kymaHandler {
	return &kymaHandler{
		orchestrations: orchestrations,
		queue:          q,
		log:            log,
		converter:      Converter{},
	}
}

func (h *kymaHandler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/upgrade/kyma", h.createOrchestration).Methods(http.MethodPost)
}

func (h *kymaHandler) createOrchestration(w http.ResponseWriter, r *http.Request) {
	params := orchestration.Parameters{}

	if r.Body != nil {
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			h.log.Errorf("while decoding request body: %v", err)
			httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "while decoding request body"))
			return
		}
	}
	err := h.validateTarget(params.Targets)
	if err != nil {
		h.log.Errorf("while validating target: %v", err)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "while validating target"))
		return
	}

	// defaults strategy if not specified to Parallel with Immediate schedule
	h.defaultOrchestrationStrategy(&params.Strategy)

	now := time.Now()
	o := internal.Orchestration{
		OrchestrationID: uuid.New().String(),
		State:           orchestration.Pending,
		Description:     "started processing of Kyma upgrade",
		Parameters:      params,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err = h.orchestrations.Insert(o)
	if err != nil {
		h.log.Errorf("while inserting orchestration to storage: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while inserting orchestration to storage"))
		return
	}

	h.queue.Add(o.OrchestrationID)

	response := orchestration.UpgradeResponse{OrchestrationID: o.OrchestrationID}

	httputil.WriteResponse(w, http.StatusAccepted, response)
}

func (h *kymaHandler) resolveErrorStatus(err error) int {
	cause := errors.Cause(err)
	switch {
	case dberr.IsNotFound(cause):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func (h *kymaHandler) validateTarget(spec orchestration.TargetSpec) error {
	if spec.Include == nil || len(spec.Include) == 0 {
		return errors.New("targets.include array must be not empty")
	}
	return nil
}

func (h *kymaHandler) defaultOrchestrationStrategy(spec *orchestration.StrategySpec) {
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
