package events

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
)

type Handler struct {
	e storage.Events
	i storage.Instances
}

func NewHandler(e storage.Events, i storage.Instances) Handler {
	return Handler{e, i}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	instanceId := r.URL.Query().Get("instance_id")
	runtimeId := r.URL.Query().Get("runtime_id")
	operationId := r.URL.Query().Get("operation_id")
	if runtimeId != "" && instanceId == "" {
		instances, _, _, err := h.i.List(dbmodel.InstanceFilter{RuntimeIDs: []string{runtimeId}})
		if len(instances) == 0 {
			http.Error(w, fmt.Sprintf("runtime_id=%v not found", runtimeId), 404)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), 503)
			return
		}
		instanceId = instances[0].InstanceID
	}
	events, err := h.e.ListEvents(dbmodel.EventFilter{InstanceID: instanceId, OperationID: operationId})
	if err != nil {
		http.Error(w, err.Error(), 503)
		return
	}
	bytes, err := json.Marshal(events)
	if err != nil {
		http.Error(w, err.Error(), 503)
		return
	}
	if _, err = w.Write(bytes); err != nil {
		http.Error(w, err.Error(), 503)
	}
}
