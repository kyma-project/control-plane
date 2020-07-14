package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/poc/component/pkg/apperror"
	"github.com/kyma-project/control-plane/poc/component/pkg/model"
	"github.com/pkg/errors"
)

const rtmIDKey = "runtimeID"

type Store interface {
	GetRuntimeByID(id string) (model.Runtime, error)
	ListRuntimes() ([]model.Runtime, error)
}

type Handler struct {
	store Store
}

func New(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Get(writer http.ResponseWriter, request *http.Request) {
	defer h.closeBody(request)

	rtmID := h.getRuntimeID(request)
	log.Printf("handling get request for ID '%s'...\n", rtmID)

	item, err := h.store.GetRuntimeByID(rtmID)
	if err != nil {
		h.writeError(writer, err)
		return
	}

	err = h.writeResponse(writer, item)
	if err != nil {
		h.writeError(writer, err)
		return
	}
}

func (h *Handler) List(writer http.ResponseWriter, request *http.Request) {
	defer h.closeBody(request)

	log.Println("handling list request...")

	items, err := h.store.ListRuntimes()
	if err != nil {
		h.writeError(writer, err)
		return
	}

	err = h.writeResponse(writer, items)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) closeBody(rq *http.Request) {
	err := rq.Body.Close()
	if err != nil {
		log.Print(errors.Wrap(err, "while closing body"))
	}
}

func (h *Handler) getRuntimeID(request *http.Request) string {
	vars := mux.Vars(request)
	id := vars[rtmIDKey]
	return id
}

const (
	HeaderContentTypeKey   = "Content-Type"
	HeaderContentTypeValue = "application/json;charset=UTF-8"
)

func (h *Handler) writeError(writer http.ResponseWriter, err error) {
	log.Printf("writing error... %+v\n", err)
	if errors.Is(err, apperror.NotFoundError) {
		http.Error(writer, err.Error(), http.StatusNotFound)
		return
	}

	http.Error(writer, err.Error(), http.StatusInternalServerError)
}

func (h *Handler) writeResponse(writer http.ResponseWriter, res interface{}) error {
	log.Printf("writing response... %+v\n", res)
	writer.Header().Set(HeaderContentTypeKey, HeaderContentTypeValue)
	return json.NewEncoder(writer).Encode(&res)
}
