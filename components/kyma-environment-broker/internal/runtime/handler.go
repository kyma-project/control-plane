package runtime

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

const (
	pageSizeParam = "page_size"
	pageParam     = "page"
)

//go:generate mockery -name=Converter -output=automock -outpkg=automock -case=underscore
type Converter interface {
	InstancesAndOperationsToDTO(internal.Instance, *internal.ProvisioningOperation, *internal.DeprovisioningOperation, *internal.UpgradeKymaOperation) (RuntimeDTO, error)
}

type Handler struct {
	instancesDb  storage.Instances
	operationsDb storage.Operations
	converter    Converter

	defaultMaxPage int
}

func NewHandler(instanceDb storage.Instances, operationDb storage.Operations, defaultMaxPage int, converter Converter) *Handler {
	return &Handler{
		instancesDb:    instanceDb,
		operationsDb:   operationDb,
		converter:      converter,
		defaultMaxPage: defaultMaxPage,
	}
}

func (h *Handler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/runtimes", h.getRuntimes)
}

func (h *Handler) getRuntimes(w http.ResponseWriter, req *http.Request) {
	toReturn := make([]RuntimeDTO, 0)
	pageSize, page, err := h.getParams(req)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrap(err, "while getting query parameters"))
		return
	}

	instances, count, totalCount, err := h.instancesDb.List(pageSize, page)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching instances"))
		return
	}

	for _, instance := range instances {
		pOpr, dOpr, ukOpr, err := h.getOperationsForInstance(instance)
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching operations for instance"))
			return
		}

		dto, err := h.converter.InstancesAndOperationsToDTO(instance, pOpr, dOpr, ukOpr)
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while converting instances to DTO"))
			return
		}

		toReturn = append(toReturn, dto)
	}

	runtimePage := RuntimesPage{
		Data:       toReturn,
		Count:      count,
		TotalCount: totalCount,
	}
	httputil.WriteResponse(w, http.StatusOK, runtimePage)
}

func (h *Handler) getOperationsForInstance(instance internal.Instance) (*internal.ProvisioningOperation, *internal.DeprovisioningOperation, *internal.UpgradeKymaOperation, error) {
	pOpr, err := h.operationsDb.GetProvisioningOperationByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return nil, nil, nil, err
	}
	dOpr, err := h.operationsDb.GetDeprovisioningOperationByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return nil, nil, nil, err
	}
	ukOpr, err := h.operationsDb.GetUpgradeKymaOperationByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return nil, nil, nil, err
	}
	return pOpr, dOpr, ukOpr, nil
}

func (h *Handler) getParams(req *http.Request) (int, int, error) {
	var pageSize int
	var page int
	var err error

	params := req.URL.Query()
	pageSizeArr, ok := params[pageSizeParam]
	if len(pageSizeArr) > 1 {
		return 0, 0, errors.New("pageSize has to be one parameter")
	}

	if !ok {
		pageSize = h.defaultMaxPage
	} else {
		pageSize, err = strconv.Atoi(pageSizeArr[0])
		if err != nil {
			return 0, 0, errors.New("pageSize has to be an integer")
		}
	}

	if pageSize > h.defaultMaxPage {
		return 0, 0, errors.New(fmt.Sprintf("pageSize is bigger than maxPage(%d)", h.defaultMaxPage))
	}

	pageArr, ok := params[pageParam]
	if len(pageArr) > 1 {
		return 0, 0, errors.New("page has to be one parameter")
	}
	if !ok {
		page = 1
	} else {
		page, err = strconv.Atoi(pageArr[0])
		if err != nil {
			return 0, 0, errors.New("page has to be an integer")
		}
	}

	return pageSize, page, nil
}
