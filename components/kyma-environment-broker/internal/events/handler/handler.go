package events

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/events"
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

func split(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	instanceId := r.URL.Query().Get("instance_ids")
	instanceIds := split(instanceId)
	runtimeId := r.URL.Query().Get("runtime_ids")
	operationId := r.URL.Query().Get("operation_ids")
	operationIds := split(operationId)
	if runtimeId != "" {
		instances, _, _, err := h.i.List(dbmodel.InstanceFilter{RuntimeIDs: split(runtimeId)})
		if err != nil {
			http.Error(w, err.Error(), 503)
			return
		}
		for _, i := range instances {
			instanceIds = append(instanceIds, i.InstanceID)
		}
	}
	events, err := h.e.ListEvents(events.EventFilter{InstanceIDs: instanceIds, OperationIDs: operationIds})
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
