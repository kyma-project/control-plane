package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type clusterHandler struct {
	orchestrations storage.Orchestrations
	queue          *process.Queue
	converter      Converter
	log            logrus.FieldLogger
}

func NewClusterHandler(orchestrations storage.Orchestrations, q *process.Queue, log logrus.FieldLogger) *clusterHandler {
	return &clusterHandler{
		orchestrations: orchestrations,
		queue:          q,
		log:            log,
		converter:      Converter{},
	}
}

func (h *clusterHandler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/upgrade/cluster", h.createOrchestration).Methods(http.MethodPost)
}

func (h *clusterHandler) createOrchestration(w http.ResponseWriter, r *http.Request) {
	// validate request body
	params := orchestration.Parameters{}
	if r.Body != nil {
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			h.log.Errorf("while decoding request body: %v", err)
			httputil.WriteErrorResponse(w, http.StatusBadRequest, fmt.Errorf("while decoding request body: %w", err))
			return
		}
	}

	// validate target
	err := validateTarget(params.Targets)
	if err != nil {
		h.log.Errorf("while validating target: %v", err)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, fmt.Errorf("while validating target: %w", err))
		return
	}

	// validate deprecated parameteter `maintenanceWindow`
	err = ValidateDeprecatedParameters(params)
	if err != nil {
		h.log.Errorf("found deprecated value: %v", err)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "found deprecated value"))
		return
	}

	// validate `schedule` field
	err = ValidateScheduleParameter(&params)
	if err != nil {
		h.log.Errorf("found deprecated value: %v", err)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "found deprecated value"))
		return
	}

	now := time.Now()
	o := internal.Orchestration{
		OrchestrationID: uuid.New().String(),
		Type:            orchestration.UpgradeClusterOrchestration,
		State:           orchestration.Pending,
		Description:     "queued for processing",
		Parameters:      params,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err = h.orchestrations.Insert(o)
	if err != nil {
		h.log.Errorf("while inserting orchestration to storage: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while inserting orchestration to storage: %w", err))
		return
	}

	h.queue.Add(o.OrchestrationID)

	response := orchestration.UpgradeResponse{OrchestrationID: o.OrchestrationID}

	httputil.WriteResponse(w, http.StatusAccepted, response)
}
