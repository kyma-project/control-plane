package runtime

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pkg/errors"

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
	limit, offset, err := getParams(req)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, errors.Wrap(err, "while getting query parameters"))
	}

	instances, err := h.db.List(limit, offset)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching instances"))
		return
	}

	writeResponse(w, http.StatusOK, instances)
}

func getParams(req *http.Request) (int, string, error) {
	params := req.URL.Query()

	limitStr, ok := params["limit"]
	if len(limitStr) > 1 {
		return 0, "", errors.New("limit has to be one parameter")
	}
	if !ok {
		limitStr[0] = "0"
	}
	limit, err := strconv.Atoi(limitStr[0])
	if err != nil {
		return 0, "", errors.New("limit has to be an integer")
	}

	offset, ok := params["offset"]
	if len(offset) > 1 {
		return 0, "", errors.New("offset has to be one parameter")
	}
	return limit, offset[0], nil
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
