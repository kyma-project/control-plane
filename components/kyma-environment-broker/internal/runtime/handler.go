package runtime

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"

	"github.com/gorilla/mux"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	db             storage.Instances
	queue          *process.Queue
	log            logrus.FieldLogger
	defaultMaxPage int
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/runtimes", h.getRuntimes)
}

func (h *Handler) getRuntimes(w http.ResponseWriter, req *http.Request) {

	instances, err := h.db.FindAllJoinedWithOperations(predicate.SortAscByCreatedAt())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching instances"))
		return
	}

	writeResponse(w, http.StatusOK, instances)
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

type errObj struct {
	Error string `json:"error"`
}

func writeErrorResponse(w http.ResponseWriter, code int, err error) {
	writeResponse(w, code, errObj{Error: err.Error()})
}
