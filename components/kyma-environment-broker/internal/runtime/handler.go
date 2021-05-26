package runtime

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"
	pkg "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pkg/errors"
)

const numberOfUpgradeOperationsToReturn = 2

type Handler struct {
	instancesDb  storage.Instances
	operationsDb storage.Operations
	converter    Converter

	defaultMaxPage int
}

func NewHandler(instanceDb storage.Instances, operationDb storage.Operations, defaultMaxPage int, defaultRequestRegion string) *Handler {
	return &Handler{
		instancesDb:    instanceDb,
		operationsDb:   operationDb,
		converter:      NewConverter(defaultRequestRegion),
		defaultMaxPage: defaultMaxPage,
	}
}

func (h *Handler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/runtimes", h.getRuntimes)
}

func (h *Handler) getRuntimes(w http.ResponseWriter, req *http.Request) {
	toReturn := make([]pkg.RuntimeDTO, 0)

	pageSize, page, err := pagination.ExtractPaginationConfigFromRequest(req, h.defaultMaxPage)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusBadRequest, errors.Wrap(err, "while getting query parameters"))
		return
	}
	filter := h.getFilters(req)
	filter.PageSize = pageSize
	filter.Page = page

	instances, count, totalCount, err := h.instancesDb.List(filter)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching instances"))
		return
	}

	for _, instance := range instances {
		dto, err := h.converter.NewDTO(instance)
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while converting instance to DTO"))
			return
		}

		provOprs, err := h.operationsDb.ListProvisioningOperationsByInstanceID(instance.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching provisioning operations list for instance"))
			return
		}
		var firstProvOp *internal.ProvisioningOperation
		if len(provOprs) != 0 {
			firstProvOp = &provOprs[len(provOprs)-1]
		}
		h.converter.ApplyProvisioningOperation(&dto, firstProvOp)
		h.converter.ApplyUnsuspensionOperations(&dto, provOprs)

		deprovOprs, err := h.operationsDb.ListDeprovisioningOperationsByInstanceID(instance.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching deprovisioning operations list for instance"))
			return
		}
		var deprovOp *internal.DeprovisioningOperation
		if len(deprovOprs) != 0 {
			for _, op := range deprovOprs {
				if !op.Temporary {
					deprovOp = &op
					break
				}
			}
		}
		h.converter.ApplyDeprovisioningOperation(&dto, deprovOp)
		h.converter.ApplySuspensionOperations(&dto, deprovOprs)

		ukOprs, err := h.operationsDb.ListUpgradeKymaOperationsByInstanceID(instance.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching upgrade kyma operation for instance"))
			return
		}
		ukOprs, totalCount := h.takeLastNonDryRunOperations(ukOprs)
		h.converter.ApplyUpgradingKymaOperations(&dto, ukOprs, totalCount)

		ucOprs, err := h.operationsDb.ListUpgradeClusterOperationsByInstanceID(instance.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching upgrade cluster operation for instance"))
			return
		}
		ucOprs, totalCount = h.takeLastNonDryRunClusterOperations(ucOprs)
		h.converter.ApplyUpgradingClusterOperations(&dto, ucOprs, totalCount)

		toReturn = append(toReturn, dto)
	}

	runtimePage := pkg.RuntimesPage{
		Data:       toReturn,
		Count:      count,
		TotalCount: totalCount,
	}
	httputil.WriteResponse(w, http.StatusOK, runtimePage)
}

func (h *Handler) takeLastNonDryRunOperations(oprs []internal.UpgradeKymaOperation) ([]internal.UpgradeKymaOperation, int) {
	toReturn := make([]internal.UpgradeKymaOperation, 0)
	totalCount := 0
	for _, op := range oprs {
		if op.DryRun {
			continue
		}
		if len(toReturn) < numberOfUpgradeOperationsToReturn {
			toReturn = append(toReturn, op)
		}
		totalCount = totalCount + 1
	}
	return toReturn, totalCount
}

func (h *Handler) takeLastNonDryRunClusterOperations(oprs []internal.UpgradeClusterOperation) ([]internal.UpgradeClusterOperation, int) {
	toReturn := make([]internal.UpgradeClusterOperation, 0)
	totalCount := 0
	for _, op := range oprs {
		if op.DryRun {
			continue
		}
		if len(toReturn) < numberOfUpgradeOperationsToReturn {
			toReturn = append(toReturn, op)
		}
		totalCount = totalCount + 1
	}
	return toReturn, totalCount
}

func (h *Handler) getFilters(req *http.Request) dbmodel.InstanceFilter {
	var filter dbmodel.InstanceFilter
	query := req.URL.Query()
	// For optional filter, zero value (nil) is fine if not supplied
	filter.GlobalAccountIDs = query[pkg.GlobalAccountIDParam]
	filter.SubAccountIDs = query[pkg.SubAccountIDParam]
	filter.InstanceIDs = query[pkg.InstanceIDParam]
	filter.RuntimeIDs = query[pkg.RuntimeIDParam]
	filter.Regions = query[pkg.RegionParam]
	filter.Domains = query[pkg.ShootParam]
	filter.Plans = query[pkg.PlanParam]
	states := query[pkg.StateParam]
	if len(states) == 0 {
		// By default if no state filters are specified, suspended/deprovisioned runtimes are still excluded.
		filter.States = append(filter.States, dbmodel.InstanceNotDeprovisioned)
	} else {
		allState := false
		for _, s := range states {
			switch pkg.State(s) {
			case pkg.StateSucceeded:
				filter.States = append(filter.States, dbmodel.InstanceSucceeded)
			case pkg.StateFailed:
				filter.States = append(filter.States, dbmodel.InstanceFailed)
			case pkg.StateProvisioning:
				filter.States = append(filter.States, dbmodel.InstanceProvisioning)
			case pkg.StateDeprovisioning:
				filter.States = append(filter.States, dbmodel.InstanceDeprovisioning)
			case pkg.StateUpgrading:
				filter.States = append(filter.States, dbmodel.InstanceUpgrading)
			case pkg.StateSuspended:
				filter.States = append(filter.States, dbmodel.InstanceDeprovisioned)
			case pkg.AllState:
				allState = true
			}
		}
		if allState {
			filter.States = nil
		}
	}

	return filter
}
