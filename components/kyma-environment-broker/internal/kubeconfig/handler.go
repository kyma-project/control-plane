package kubeconfig

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

const attachmentName = "kubeconfig.yaml"

//go:generate mockery -name=KcBuilder -output=automock -outpkg=automock -case=underscore

type KcBuilder interface {
	Build(*internal.Instance) (string, error)
}

type Handler struct {
	kubeconfigBuilder KcBuilder
	instanceStorage   storage.Instances
	operationStorage  storage.Operations
	log               logrus.FieldLogger
}

func NewHandler(storage storage.BrokerStorage, b KcBuilder, log logrus.FieldLogger) *Handler {
	return &Handler{
		instanceStorage:   storage.Instances(),
		operationStorage:  storage.Operations(),
		kubeconfigBuilder: b,
		log:               log,
	}
}

func (h *Handler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/kubeconfig/{instance_id}", h.GetKubeconfig).Methods(http.MethodGet)
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("instanceID is required"))
	})
}

type ErrorResponse struct {
	Error string
}

func (h *Handler) GetKubeconfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceID := vars["instance_id"]

	instance, err := h.instanceStorage.GetByID(instanceID)
	switch {
	case err == nil:
	case dberr.IsNotFound(err):
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("instance with ID %s does not exist", instanceID))
		return
	default:
		h.handleResponse(w, http.StatusInternalServerError, err)
		return
	}

	if instance.RuntimeID == "" {
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("kubeconfig for instance %s does not exist. Provisioning could be in progress, please try again later", instanceID))
		return
	}

	operation, err := h.operationStorage.GetProvisioningOperationByInstanceID(instanceID)
	switch {
	case err == nil:
	case dberr.IsNotFound(err):
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("provisioning operation for instance with ID %s does not exist", instanceID))
		return
	default:
		h.handleResponse(w, http.StatusInternalServerError, err)
		return
	}

	if operation.InstanceID != instanceID {
		h.handleResponse(w, http.StatusBadRequest, errors.New("mismatch between operation and instance"))
		return
	}

	switch operation.State {
	case domain.InProgress, orchestration.Pending:
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("provisioning operation for instance %s is in progress state, kubeconfig not exist yet, please try again later", instanceID))
		return
	case domain.Failed:
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("provisioning operation for instance %s failed, kubeconfig does not exist", instanceID))
		return
	}

	newKubeconfig, err := h.kubeconfigBuilder.Build(instance)
	if err != nil {
		h.handleResponse(w, http.StatusInternalServerError, fmt.Errorf("cannot fetch SKR kubeconfig: %s", err))
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", attachmentName))
	w.Header().Add("Content-Type", "application/x-yaml")
	_, err = w.Write([]byte(newKubeconfig))
	if err != nil {
		h.log.Errorf("cannot write response with new kubeconfig: %s", err)
	}
}

func (h *Handler) handleResponse(w http.ResponseWriter, code int, err error) {
	errEncode := httputil.JSONEncodeWithCode(w, &ErrorResponse{Error: err.Error()}, code)
	if errEncode != nil {
		h.log.Errorf("cannot encode error response: %s", errEncode)
	}
}
