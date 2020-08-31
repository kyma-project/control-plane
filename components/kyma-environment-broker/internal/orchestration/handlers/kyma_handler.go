package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

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
	log   logrus.FieldLogger
}

func NewKymaOrchestrationHandler(db storage.Orchestrations, executor process.Executor, log logrus.FieldLogger) *kymaHandler {
	return &kymaHandler{
		db:    db,
		log:   log,
		queue: process.NewQueue(executor, log),
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
		h.log.Errorf("while getting orchestration %s: %v", orchestrationID, err)
		writeErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while getting orchestration %s", orchestrationID))
	}

	writeResponse(w, http.StatusOK, o)
}

func (h *kymaHandler) listOrchestration(w http.ResponseWriter, r *http.Request) {
	o, err := h.db.ListAll()
	if err != nil {
		h.log.Errorf("while getting orchestrations: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while getting orchestration"))
	}

	writeResponse(w, http.StatusOK, o)
}

func (h *kymaHandler) createOrchestration(w http.ResponseWriter, r *http.Request) {
	dto := orchestration.Parameters{}

	err := json.NewDecoder(r.Body).Decode(&dto)
	if err != nil {
		h.log.Errorf("while decoding request body: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "while decoding request body"))
	}

	dto.Targets, err = h.resolveTargets(dto.Targets)
	if err != nil {
		h.log.Errorf("while resolving targets: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, errors.Wrapf(err, "while resolving targets"))
	}

	params, err := json.Marshal(dto)
	if err != nil {
		h.log.Errorf("while encoding request params: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while encoding request params"))
	}

	now := time.Now()
	o := internal.Orchestration{
		OrchestrationID: uuid.New().String(),
		State:           internal.Pending,
		Description:     "started processing of Kyma upgrade",
		Parameters: sql.NullString{
			String: string(params),
			Valid:  true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = h.db.Insert(o)
	if err != nil {
		h.log.Errorf("while inserting operation to storage: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, errors.Wrapf(err, "while inserting operation to storage"))
	}

	h.queue.Add(o.OrchestrationID)

	response := orchestration.UpgradeResponse{OrchestrationID: o.OrchestrationID}

	writeResponse(w, http.StatusAccepted, response)
}

// resolveTargets helps to reject request if the request body is incorrect
// TODO(upgrade): get rid of it after switching to bulk
func (h *kymaHandler) resolveTargets(targets internal.TargetSpec) (internal.TargetSpec, error) {
	lenTargets := len(targets.Include)
	if lenTargets > 1 {
		h.log.Errorf("only 1 target is allowed")
		return internal.TargetSpec{}, errors.New("only 1 target is allowed")
	}
	if lenTargets == 1 {
		if targets.Include[0].RuntimeID == "" {
			return internal.TargetSpec{}, errors.New("runtimeId must be specified in the included target")
		}
	}
	// TODO(upgrade): exclude not supported until bulk
	targets.Exclude = nil

	return targets, nil
}

func writeResponse(w http.ResponseWriter, code int, object interface{}) {
	data, err := json.Marshal(object)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(data)
	if err != nil {
		logrus.Warnf("could not write response %s", string(data))
	}
}

func writeErrorResponse(w http.ResponseWriter, code int, err error) {
	writeResponse(w, code, err.Error())
}
