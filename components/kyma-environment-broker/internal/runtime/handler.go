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

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	db             storage.Instances
	log            logrus.FieldLogger
	defaultMaxPage int
}

func NewHandler(db storage.Instances, log logrus.FieldLogger, defaultMaxPage int) *Handler {
	return &Handler{
		db:             db,
		log:            log,
		defaultMaxPage: defaultMaxPage,
	}
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
	limit, cursor, err := h.getParams(req)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, errors.Wrap(err, "while getting query parameters"))
		return
	}

	instances, pageInfo, totalCount, err := h.db.List(limit, cursor)

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
	var limit int
	var cursor string
	var err error

	params := req.URL.Query()
	h.log.Printf("A")
	limitArr, ok := params["limit"]
	if len(limitArr) > 1 {
		return 0, "", errors.New("limit has to be one parameter")
	}
	h.log.Printf("B")

	if !ok {
		limit = h.defaultMaxPage
	} else {
		limit, err = strconv.Atoi(limitArr[0])
		if err != nil {
			return 0, "", errors.New("limit has to be an integer")
		}
	}

	h.log.Printf("C")
	if limit > h.defaultMaxPage {
		return 0, "", errors.New(fmt.Sprintf("limit is bigger than maxPage(%d)", h.defaultMaxPage))
	}

	h.log.Printf("D")
	cursorArr, ok := params["cursor"]
	if len(cursorArr) > 1 {
		return 0, "", errors.New("cursor has to be one parameter")
	}
	if !ok {
		cursor = ""
	} else {
		cursor = cursorArr[0]
	}

	return limit, cursor, nil
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
