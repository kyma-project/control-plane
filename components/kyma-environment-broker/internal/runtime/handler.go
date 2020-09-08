package runtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kyma-incubator/compass/components/director/pkg/pagination"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

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

type InstancesPage struct {
	Data       []internal.Instance `json:"Data"`
	PageInfo   *pagination.Page    `json:"PageInfo"`
	TotalCount int                 `json:"TotalCount"`
}

func (h *Handler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/runtimes", h.getRuntimes)
}

func (h *Handler) getRuntimes(w http.ResponseWriter, req *http.Request) {
	limit, offset, err := h.getParams(req)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, errors.Wrap(err, "while getting query parameters"))
	}

	instances, pageInfo, totalCount, err := h.db.List(limit, offset)

	page := InstancesPage{
		Data:       instances,
		PageInfo:   pageInfo,
		TotalCount: totalCount,
	}
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching instances"))
		return
	}

	writeResponse(w, http.StatusOK, page)
}

func (h *Handler) getParams(req *http.Request) (int, string, error) {
	params := req.URL.Query()

	limitStr, ok := params["limit"]
	if len(limitStr) > 1 {
		return 0, "", errors.New("limit has to be one parameter")
	}
	if !ok {
		limitStr[0] = string(h.defaultMaxPage)
	}
	limit, err := strconv.Atoi(limitStr[0])
	if err != nil {
		return 0, "", errors.New("limit has to be an integer")
	}
	if limit > h.defaultMaxPage {
		return 0, "", errors.New(fmt.Sprintf("limit is bigger than maxPage(%d)", h.defaultMaxPage))
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
