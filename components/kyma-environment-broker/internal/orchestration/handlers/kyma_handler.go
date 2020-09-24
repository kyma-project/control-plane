package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type kymaHandler struct {
	db    storage.Orchestrations
	queue *process.Queue
	conv  Converter
	log   logrus.FieldLogger
}

func NewKymaOrchestrationHandler(db storage.Orchestrations, q *process.Queue, log logrus.FieldLogger) *kymaHandler {
	return &kymaHandler{
		db:    db,
		log:   log,
		conv:  Converter{},
		queue: q,
	}
}

func (h *kymaHandler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/orchestrations/{orchestration_id}", h.getOrchestration).Methods(http.MethodGet)
	router.HandleFunc("/orchestrations", h.listOrchestration).Methods(http.MethodGet)
	router.HandleFunc("/upgrade/kyma", h.createOrchestration).Methods(http.MethodPost)
}

func (h *kymaHandler) getOrchestration(w http.ResponseWriter, r *http.Request) {
	orchestrationID := mux.Vars(r)["orchestration_id"]

	o, err := h.db.GetByID(orchestrationID)
	if err != nil {
		status := http.StatusInternalServerError
		if dberr.IsNotFound(err) {
			status = http.StatusNotFound
		}
		h.log.Errorf("while getting orchestration %s: %v", orchestrationID, err)
		httputil.WriteErrorResponse(w, status, errors.Wrapf(err, "while getting orchestration %s", orchestrationID))
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
	orchestrations, err := h.db.ListAll()
	if err != nil {
		h.log.Errorf("while getting orchestrations: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while getting orchestrations"))
		return
	}

	response, err := h.conv.OrchestrationListToDTO(orchestrations)
	if err != nil {
		h.log.Errorf("while converting orchestrations: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while converting orchestrations"))
		return
	}

	httputil.WriteResponse(w, http.StatusOK, response)
}

func (h *kymaHandler) createOrchestration(w http.ResponseWriter, r *http.Request) {
	params := internal.OrchestrationParameters{}

	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		h.log.Errorf("while decoding request body: %v", err)
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "while decoding request body"))
		return
	}

	now := time.Now()
	o := internal.Orchestration{
		OrchestrationID: uuid.New().String(),
		State:           internal.Pending,
		Description:     "started processing of Kyma upgrade",
		Parameters:      params,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err = h.db.Insert(o)
	if err != nil {
		h.log.Errorf("while inserting orchestration to storage: %v", err)
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while inserting orchestration to storage"))
		return
	}

	h.queue.Add(o.OrchestrationID)

	response := orchestration.UpgradeResponse{OrchestrationID: o.OrchestrationID}

	httputil.WriteResponse(w, http.StatusAccepted, response)
}
