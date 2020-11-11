package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type kymaHandler struct {
	orchestrations storage.Orchestrations
	operations     storage.Operations
	runtimeStates  storage.RuntimeStates

	queue *process.Queue
	conv  Converter
	log   logrus.FieldLogger

	defaultMaxPage int
}

func NewKymaOrchestrationHandler(operations storage.Operations, orchestrations storage.Orchestrations, runtimeStates storage.RuntimeStates, defaultMaxPage int, q *process.Queue, log logrus.FieldLogger) *kymaHandler {
	return &kymaHandler{
		operations:     operations,
		orchestrations: orchestrations,
		runtimeStates:  runtimeStates,
		queue:          q,
		log:            log,
		conv:           Converter{},
		defaultMaxPage: defaultMaxPage,
	}
}

func (h *kymaHandler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/upgrade/kyma", h.createOrchestration).Methods(http.MethodPost)

	router.HandleFunc("/orchestrations", h.listOrchestration).Methods(http.MethodGet)
	router.HandleFunc("/orchestrations/{orchestration_id}", h.getOrchestration).Methods(http.MethodGet)
	router.HandleFunc("/orchestrations/{orchestration_id}/operations", h.listOperations).Methods(http.MethodGet)
	router.HandleFunc("/orchestrations/{orchestration_id}/operations/{operation_id}", h.getOperation).Methods(http.MethodGet)
}

func (h *kymaHandler) getOrchestration(w http.ResponseWriter, r *http.Request) {
	orchestrationID := mux.Vars(r)["orchestration_id"]

	o, err := h.orchestrations.GetByID(orchestrationID)
	if err != nil {
		h.log.Errorf("while getting orchestration %s: %v", orchestrationID, err)
		httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), errors.Wrapf(err, "while getting orchestration %s", orchestrationID))
		return
	}

	response, err := h.conv.OrchestrationToDTO(o)
	if err != nil {
		h.log.Errorf("while converting orchestration: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while converting orchestration"))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, response)
}

func (h *kymaHandler) listOrchestration(w http.ResponseWriter, r *http.Request) {
	pageSize, page, err := pagination.ExtractPaginationConfigFromRequest(r, h.defaultMaxPage)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrap(err, "while getting query parameters"))
		return
	}
	query := r.URL.Query()
	filter := dbmodel.OrchestrationFilter{
		Page:     page,
		PageSize: pageSize,
		// For optional filters, zero value (nil) is ok if not supplied
		States: query[orchestration.StateParam],
	}

	orchestrations, count, totalCount, err := h.orchestrations.List(filter)
	if err != nil {
		h.log.Errorf("while getting orchestrations: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while getting orchestrations"))
		return
	}

	response, err := h.conv.OrchestrationListToDTO(orchestrations, count, totalCount)
	if err != nil {
		h.log.Errorf("while converting orchestrations: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while converting orchestrations"))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, response)
}

func (h *kymaHandler) listOperations(w http.ResponseWriter, r *http.Request) {
	orchestrationID := mux.Vars(r)["orchestration_id"]
	pageSize, page, err := pagination.ExtractPaginationConfigFromRequest(r, h.defaultMaxPage)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrap(err, "while getting query parameters"))
		return
	}
	query := r.URL.Query()
	filter := dbmodel.OperationFilter{
		Page:     page,
		PageSize: pageSize,
		// For optional filters, zero value (nil) is ok if not supplied
		States: query[orchestration.StateParam],
	}

	operations, count, totalCount, err := h.operations.ListUpgradeKymaOperationsByOrchestrationID(orchestrationID, filter)
	if err != nil {
		h.log.Errorf("while getting operations: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while getting operations"))
		return
	}

	response, err := h.conv.UpgradeKymaOperationListToDTO(operations, count, totalCount)
	if err != nil {
		h.log.Errorf("while converting operations: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while converting operations"))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, response)
}

func (h *kymaHandler) getOperation(w http.ResponseWriter, r *http.Request) {
	operationID := mux.Vars(r)["operation_id"]

	operation, err := h.operations.GetUpgradeKymaOperationByID(operationID)
	if err != nil {
		h.log.Errorf("while getting upgrade operation %s: %v", operationID, err)
		httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), errors.Wrapf(err, "while getting operation %s", operationID))
		return
	}
	provisioningOp, err := h.operations.GetProvisioningOperationByInstanceID(operation.InstanceID)
	if err != nil {
		h.log.Errorf("while getting provisioning operation for instance %s: %v", operation.InstanceID, err)
		httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), errors.Wrapf(err, "while getting provisioning operation for instance %s", operation.InstanceID))
		return
	}
	provisioningState, err := h.runtimeStates.GetByOperationID(provisioningOp.ID)
	if err != nil {
		h.log.Errorf("while getting runtime state for operation %s: %v", provisioningOp.ID, err)
	}

	upgradeState, err := h.runtimeStates.GetByOperationID(operationID)
	if err != nil && !dberr.IsNotFound(err) {
		h.log.Errorf("while getting runtime state for upgrade operation %s: %v", operationID, err)
	}

	response, err := h.conv.UpgradeKymaOperationToDetailDTO(*operation, upgradeState.KymaConfig, provisioningState.ClusterConfig)
	if err != nil {
		h.log.Errorf("while converting operation: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while converting operation"))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, response)
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
		State:           internal.Pending,
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
	switch {
	case dberr.IsNotFound(err):
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
