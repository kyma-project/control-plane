package kubeconfig

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kennygrant/sanitize"

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

//go:generate mockery --name=KcBuilder --output=automock --outpkg=automock --case=underscore

type KcBuilder interface {
	Build(*internal.Instance) (string, error)
	BuildFromAdminKubeconfig(instance *internal.Instance, adminKubeconfig string) (string, error)
}

type Handler struct {
	kubeconfigBuilder KcBuilder
	allowOrigins      string
	instanceStorage   storage.Instances
	operationStorage  storage.Operations
	log               logrus.FieldLogger
}

func NewHandler(storage storage.BrokerStorage, b KcBuilder, origins string, log logrus.FieldLogger) *Handler {
	return &Handler{
		instanceStorage:   storage.Instances(),
		operationStorage:  storage.Operations(),
		kubeconfigBuilder: b,
		allowOrigins:      origins,
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

	h.specifyAllowOriginHeader(r, w)

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

	if broker.IsOwnClusterPlan(instance.ServicePlanID) {
		h.handleResponse(w, http.StatusNotFound, fmt.Errorf("kubeconfig for instance %s does not exist", instanceID))
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
		h.handleResponse(w, http.StatusBadRequest, fmt.Errorf("mismatch between operation and instance"))
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

	var newKubeconfig string
	if instance.ServicePlanID == broker.OwnClusterPlanID {
		newKubeconfig, err = h.kubeconfigBuilder.BuildFromAdminKubeconfig(instance, instance.InstanceDetails.Kubeconfig)
	} else {
		newKubeconfig, err = h.kubeconfigBuilder.Build(instance)
	}
	if err != nil {
		h.handleResponse(w, http.StatusInternalServerError, fmt.Errorf("cannot fetch SKR kubeconfig: %s", err))
		return
	}

	writeToResponse(w, newKubeconfig, h.log)
}

func (h *Handler) handleResponse(w http.ResponseWriter, code int, err error) {
	errEncode := httputil.JSONEncodeWithCode(w, &ErrorResponse{Error: err.Error()}, code)
	if errEncode != nil {
		h.log.Errorf("cannot encode error response: %s", errEncode)
	}
}

func (h *Handler) specifyAllowOriginHeader(r *http.Request, w http.ResponseWriter) {
	origin := r.Header.Get("Origin")
	origin = strings.ReplaceAll(origin, "\r", "")
	origin = strings.ReplaceAll(origin, "\n", "")
	if origin == "" {
		return
	}

	if h.allowOrigins == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return
	}

	for _, o := range strings.Split(h.allowOrigins, ",") {
		if o == origin {
			w.Header().Set("Access-Control-Allow-Origin", sanitize.HTML(origin))
			return
		}
	}
}

func writeToResponse(w http.ResponseWriter, data string, l logrus.FieldLogger) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", attachmentName))
	w.Header().Add("Content-Type", "application/x-yaml")
	_, err := w.Write([]byte(data))
	if err != nil {
		l.Errorf("cannot write response with new kubeconfig: %s", err)
	}
}
