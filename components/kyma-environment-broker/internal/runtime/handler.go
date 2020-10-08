package runtime

import (
	"fmt"
	"net/http"
	"strconv"

	pkg "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

//go:generate mockery -name=Converter -output=automock -outpkg=automock -case=underscore
type Converter interface {
	InstancesAndOperationsToDTO(internal.Instance, *internal.ProvisioningOperation, *internal.DeprovisioningOperation, *internal.UpgradeKymaOperation) (pkg.RuntimeDTO, error)
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
	toReturn := make([]pkg.RuntimeDTO, 0)
	filter, err := h.getFilters(req)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrap(err, "while getting query parameters"))
		return
	}

	instances, count, totalCount, err := h.instancesDb.List(filter)
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

	runtimePage := pkg.RuntimesPage{
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

func (h *Handler) getFilters(req *http.Request) (dbmodel.InstanceFilter, error) {
	var filter dbmodel.InstanceFilter
	var err error

	query := req.URL.Query()
	pageSizeArr, ok := query[pkg.PageSizeParam]
	if len(pageSizeArr) > 1 {
		return filter, errors.New("pageSize has to be one parameter")
	}

	if !ok {
		filter.PageSize = h.defaultMaxPage
	} else {
		filter.PageSize, err = strconv.Atoi(pageSizeArr[0])
		if err != nil {
			return filter, errors.New("pageSize has to be an integer")
		}
	}

	if filter.PageSize > h.defaultMaxPage {
		return filter, errors.New(fmt.Sprintf("pageSize is bigger than maxPage(%d)", h.defaultMaxPage))
	}
	if filter.PageSize < 1 {
		return filter, errors.New("pageSize cannot be smaller than 1")
	}

	pageArr, ok := query[pkg.PageParam]
	if len(pageArr) > 1 {
		return filter, errors.New("page has to be one parameter")
	}
	if !ok {
		filter.Page = 1
	} else {
		filter.Page, err = strconv.Atoi(pageArr[0])
		if err != nil {
			return filter, errors.New("page has to be an integer")
		}
		if filter.Page < 1 {
			return filter, errors.New("page cannot be smaller than 1")
		}
	}

	// For optional filter filter, zero value (nil) is fine if not supplied
	filter.GlobalAccountIDs = query[pkg.GlobalAccountIDParam]
	filter.SubAccountIDs = query[pkg.SubAccountIDParam]
	filter.InstanceIDs = query[pkg.InstanceIDParam]
	filter.RuntimeIDs = query[pkg.RuntimeIDParam]
	filter.Regions = query[pkg.RegionParam]
	filter.Domains = query[pkg.ShootParam]

	return filter, nil
}
