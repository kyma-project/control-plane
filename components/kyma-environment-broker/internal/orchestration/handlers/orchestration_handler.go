package handlers

import (
	"fmt"
	"net/http"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"

	internalError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"

	"github.com/gorilla/mux"
	commonOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"

	"github.com/sirupsen/logrus"
)

type orchestrationHandler struct {
	orchestrations storage.Orchestrations
	operations     storage.Operations
	runtimeStates  storage.RuntimeStates

	converter Converter
	log       logrus.FieldLogger

	canceler       *Canceler
	kymaRetryer    *kymaRetryer
	clusterRetryer *clusterRetryer

	defaultMaxPage int
}

// NewOrchestrationStatusHandler exposes data about orchestrations and allows to manage them
func NewOrchestrationStatusHandler(operations storage.Operations,
	orchestrations storage.Orchestrations,
	runtimeStates storage.RuntimeStates,
	kymaQueue *process.Queue,
	clusterQueue *process.Queue,
	defaultMaxPage int,
	log logrus.FieldLogger) *orchestrationHandler {
	return &orchestrationHandler{
		operations:     operations,
		orchestrations: orchestrations,
		runtimeStates:  runtimeStates,
		log:            log,
		defaultMaxPage: defaultMaxPage,
		converter:      Converter{},
		canceler:       NewCanceler(orchestrations, log),
		kymaRetryer:    NewKymaRetryer(orchestrations, operations, kymaQueue, log),
		clusterRetryer: NewClusterRetryer(orchestrations, operations, clusterQueue, log),
	}
}

func (h *orchestrationHandler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/orchestrations", h.listOrchestration).Methods(http.MethodGet)
	router.HandleFunc("/orchestrations/{orchestration_id}", h.getOrchestration).Methods(http.MethodGet)
	router.HandleFunc("/orchestrations/{orchestration_id}/cancel", h.cancelOrchestrationByID).Methods(http.MethodPut)
	router.HandleFunc("/orchestrations/{orchestration_id}/operations", h.listOperations).Methods(http.MethodGet)
	router.HandleFunc("/orchestrations/{orchestration_id}/operations/{operation_id}", h.getOperation).Methods(http.MethodGet)
	router.HandleFunc("/orchestrations/{orchestration_id}/retry", h.retryOrchestrationByID).Methods(http.MethodPost)
}

func (h *orchestrationHandler) getOrchestration(w http.ResponseWriter, r *http.Request) {
	orchestrationID := mux.Vars(r)["orchestration_id"]

	o, err := h.orchestrations.GetByID(orchestrationID)
	if err != nil {
		h.log.Errorf("while getting orchestration %s: %v", orchestrationID, err)
		httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), fmt.Errorf("while getting orchestration %s: %w", orchestrationID, err))
		return
	}

	stats, err := h.operations.GetOperationStatsForOrchestration(orchestrationID)
	if err != nil {
		h.log.Errorf("while getting orchestration %s operation statistics: %v", orchestrationID, err)
		httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), fmt.Errorf("while getting orchestration %s operation statistics: %w", orchestrationID, err))
		return
	}

	response, err := h.converter.OrchestrationToDTO(o, stats)
	if err != nil {
		h.log.Errorf("while converting orchestration: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while converting orchestration: %w", err))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, response)
}

func (h *orchestrationHandler) cancelOrchestrationByID(w http.ResponseWriter, r *http.Request) {
	orchestrationID := mux.Vars(r)["orchestration_id"]

	err := h.canceler.CancelForID(orchestrationID)
	if err != nil {
		h.log.Errorf("while canceling orchestration %s: %v", orchestrationID, err)
		httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), fmt.Errorf("while canceling orchestration %s: %w", orchestrationID, err))
		return
	}

	response := commonOrchestration.UpgradeResponse{OrchestrationID: orchestrationID}

	httputil.WriteResponse(w, http.StatusOK, response)
}

func (h *orchestrationHandler) retryOrchestrationByID(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-type")
	if contentType != "application/x-www-form-urlencoded" {
		h.log.Errorf("invalide content type %s for retrying orchestration", contentType)
		httputil.WriteErrorResponse(w, http.StatusUnsupportedMediaType, fmt.Errorf("invalide content type %s for retrying orchestration", contentType))
		return
	}

	orchestrationID := mux.Vars(r)["orchestration_id"]
	operationIDs := []string{}
	var immediateID string

	if r.Body != nil {
		if err := r.ParseForm(); err != nil {
			h.log.Errorf("cannot parse form while retrying orchestration: %s: %v", orchestrationID, err)
			httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), fmt.Errorf("cannot parse form while retrying orchestration: %s: %w", orchestrationID, err))
			return
		}

		operationIDs = r.Form["operation-id"]
		immediateID = r.FormValue("immediate")
	}

	o, err := h.orchestrations.GetByID(orchestrationID)
	if err != nil {
		h.log.Errorf("while retrying orchestration %s: %v", orchestrationID, err)
		httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), fmt.Errorf("while retrying orchestration %s: %w", orchestrationID, err))
		return
	}

	filter := dbmodel.OperationFilter{
		States: []string{commonOrchestration.Failed},
	}

	var response commonOrchestration.RetryResponse
	switch o.Type {
	case commonOrchestration.UpgradeKymaOrchestration:
		allOps, _, _, err := h.operations.ListUpgradeKymaOperationsByOrchestrationID(o.OrchestrationID, filter)
		if err != nil {
			h.log.Errorf("while getting operations: %v", err)
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while getting operations: %w", err))
			return
		}

		response, err = h.kymaRetryer.orchestrationRetry(o, allOps, operationIDs, immediateID)
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

	case commonOrchestration.UpgradeClusterOrchestration:
		allOps, _, _, err := h.operations.ListUpgradeClusterOperationsByOrchestrationID(o.OrchestrationID, filter)
		if err != nil {
			h.log.Errorf("while getting operations: %v", err)
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while getting operations: %w", err))
			return
		}

		response, err = h.clusterRetryer.orchestrationRetry(o, allOps, operationIDs)
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

	default:
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("unsupported orchestration type: %s", o.Type))
		return
	}

	httputil.WriteResponse(w, http.StatusAccepted, response)
}

func (h *orchestrationHandler) listOrchestration(w http.ResponseWriter, r *http.Request) {
	pageSize, page, err := pagination.ExtractPaginationConfigFromRequest(r, h.defaultMaxPage)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusBadRequest, fmt.Errorf("while getting query parameters: %w", err))
		return
	}
	query := r.URL.Query()
	filter := dbmodel.OrchestrationFilter{
		Page:     page,
		PageSize: pageSize,
		// For optional filters, zero value (nil) is ok if not supplied
		States: query[commonOrchestration.StateParam],
	}

	orchestrations, count, totalCount, err := h.orchestrations.List(filter)
	if err != nil {
		h.log.Errorf("while getting orchestrations: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while getting orchestrations: %w", err))
		return
	}

	response, err := h.converter.OrchestrationListToDTO(orchestrations, count, totalCount)
	if err != nil {
		h.log.Errorf("while converting orchestrations: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while converting orchestrations: %w", err))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, response)
}

func (h *orchestrationHandler) listOperations(w http.ResponseWriter, r *http.Request) {
	orchestrationID := mux.Vars(r)["orchestration_id"]
	pageSize, page, err := pagination.ExtractPaginationConfigFromRequest(r, h.defaultMaxPage)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusBadRequest, fmt.Errorf("while getting query parameters: %w", err))
		return
	}
	query := r.URL.Query()
	filter := dbmodel.OperationFilter{
		Page:     page,
		PageSize: pageSize,
		// For optional filters, zero value (nil) is ok if not supplied
		States: query[commonOrchestration.StateParam],
	}

	o, err := h.orchestrations.GetByID(orchestrationID)
	if err != nil {
		h.log.Errorf("while getting orchestration %s: %v", orchestrationID, err)
		httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), fmt.Errorf("while getting orchestration %s: %w", orchestrationID, err))
		return
	}

	var response commonOrchestration.OperationResponseList
	switch o.Type {
	case commonOrchestration.UpgradeKymaOrchestration:
		operations, count, totalCount, err := h.operations.ListUpgradeKymaOperationsByOrchestrationID(orchestrationID, filter)
		if err != nil {
			h.log.Errorf("while getting operations: %v", err)
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while getting operations: %w", err))
			return
		}
		response, err = h.converter.UpgradeKymaOperationListToDTO(operations, count, totalCount)
		if err != nil {
			h.log.Errorf("while converting operations: %v", err)
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while converting operations: %w", err))
			return
		}

	case commonOrchestration.UpgradeClusterOrchestration:
		operations, count, totalCount, err := h.operations.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, filter)
		if err != nil {
			h.log.Errorf("while getting operations: %v", err)
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while getting operations: %w", err))
			return
		}
		response, err = h.converter.UpgradeClusterOperationListToDTO(operations, count, totalCount)
		if err != nil {
			h.log.Errorf("while converting operations: %v", err)
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while converting operations: %w", err))
			return
		}

	default:
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("unsupported orchestration type: %s", o.Type))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, response)
}

func (h *orchestrationHandler) getOperation(w http.ResponseWriter, r *http.Request) {
	orchestrationID := mux.Vars(r)["orchestration_id"]
	operationID := mux.Vars(r)["operation_id"]

	o, err := h.orchestrations.GetByID(orchestrationID)
	if err != nil {
		h.log.Errorf("while getting orchestration %s: %v", orchestrationID, err)
		httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), fmt.Errorf("while getting orchestration %s: %w", orchestrationID, err))
		return
	}

	upgradeState, err := h.runtimeStates.GetByOperationID(operationID)
	if err != nil && !dberr.IsNotFound(err) {
		h.log.Errorf("while getting runtime state for upgrade operation %s: %v", operationID, err)
	}

	kymaConfig := upgradeState.GetKymaConfig()

	var response commonOrchestration.OperationDetailResponse
	switch o.Type {
	case commonOrchestration.UpgradeKymaOrchestration:
		operation, err := h.operations.GetUpgradeKymaOperationByID(operationID)
		if err != nil {
			h.log.Errorf("while getting upgrade operation %s: %v", operationID, err)
			httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), fmt.Errorf("while getting upgrade operation %s: %w", operationID, err))
			return
		}

		response, err = h.converter.UpgradeKymaOperationToDetailDTO(*operation, &kymaConfig)
		if err != nil {
			h.log.Errorf("while converting operation: %v", err)
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while converting operation: %w", err))
			return
		}

	case commonOrchestration.UpgradeClusterOrchestration:
		operation, err := h.operations.GetUpgradeClusterOperationByID(operationID)
		if err != nil {
			h.log.Errorf("while getting upgrade operation %s: %v", operationID, err)
			httputil.WriteErrorResponse(w, h.resolveErrorStatus(err), fmt.Errorf("while getting upgrade operation %s: %w", operationID, err))
			return
		}

		response, err = h.converter.UpgradeClusterOperationToDetailDTO(*operation, &upgradeState.ClusterConfig)
		if err != nil {
			h.log.Errorf("while converting operation: %v", err)
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while converting operation: %w", err))
			return
		}

	default:
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("unsupported orchestration type: %s", o.Type))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, response)
}

func (h *orchestrationHandler) resolveErrorStatus(err error) int {
	cause := internalError.UnwrapAll(err)
	switch { //TODO: rethink to validate error type
	case dberr.IsNotFound(cause):
		return http.StatusNotFound
	case apiErrors.IsBadRequest(cause):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
