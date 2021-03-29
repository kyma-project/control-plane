package runtime

import (
	"net/http"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"
	pkg "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"

	"github.com/gorilla/mux"
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
		var firstProvOp internal.ProvisioningOperation
		if len(provOprs) != 0 {
			firstProvOp = provOprs[len(provOprs)-1]
		}
		h.converter.ApplyProvisioningOperation(&dto, &firstProvOp)
		h.converter.ApplyUnsuspensionOperations(&dto, provOprs)

		dOpr, err := h.operationsDb.GetDeprovisioningOperationByInstanceID(instance.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching deprovisioning operation for instance"))
			return
		}
		h.converter.ApplyDeprovisioningOperation(&dto, dOpr)

		ukOprs, err := h.operationsDb.ListUpgradeKymaOperationsByInstanceID(instance.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching upgrade kyma operation for instance"))
			return
		}
		ukOprs, totalCount := h.takeLastNonDryRunKymaOperations(ukOprs)
		h.converter.ApplyUpgradingKymaOperations(&dto, ukOprs, totalCount)

		ucOprs, err := h.operationsDb.ListUpgradeClusterOperationsByInstanceID(instance.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching upgrade cluster operation for instance"))
			return
		}
		ucOprs, totalCount = h.takeLastNonDryRunClusterOperations(ucOprs)
		h.converter.ApplyUpgradingClusterOperations(&dto, ucOprs, totalCount)

		deprovOprs, err := h.operationsDb.ListDeprovisioningOperationsByInstanceID(instance.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "while fetching deprovisioning operations list for instance"))
			return
		}
		h.converter.ApplySuspensionOperations(&dto, deprovOprs)

		toReturn = append(toReturn, dto)
	}

	runtimePage := pkg.RuntimesPage{
		Data:       toReturn,
		Count:      count,
		TotalCount: totalCount,
	}
	httputil.WriteResponse(w, http.StatusOK, runtimePage)
}

func (h *Handler) takeLastNonDryRunKymaOperations(oprs []internal.UpgradeKymaOperation) ([]internal.UpgradeKymaOperation, int) {
	ops := make([]interface{}, len(oprs))
	for i, o := range oprs {
		ops[i] = o
	}

	ri, totalCount := h.takeLastNonDryRunOperations(ops)
	toReturn := make([]internal.UpgradeKymaOperation, len(ri))
	for i, o := range ri {
		toReturn[i] = o.(internal.UpgradeKymaOperation)
	}

	return toReturn, totalCount

}

func (h *Handler) takeLastNonDryRunClusterOperations(oprs []internal.UpgradeClusterOperation) ([]internal.UpgradeClusterOperation, int) {
	ops := make([]interface{}, len(oprs))
	for i, o := range oprs {
		ops[i] = o
	}

	ri, totalCount := h.takeLastNonDryRunOperations(ops)
	toReturn := make([]internal.UpgradeClusterOperation, len(ri))
	for i, o := range ri {
		toReturn[i] = o.(internal.UpgradeClusterOperation)
	}

	return toReturn, totalCount
}

// common "counter" for two types of operations
func (h *Handler) takeLastNonDryRunOperations(oprs []interface{}) ([]interface{}, int) {
	toReturn := make([]interface{}, 0)
	totalCount := 0
	for _, op := range oprs {
		switch op.(type) {

		case internal.UpgradeKymaOperation:
			o := op.(internal.UpgradeKymaOperation)
			if o.DryRun {
				continue
			}

		case internal.UpgradeClusterOperation:
			o := op.(internal.UpgradeClusterOperation)
			if o.DryRun {
				continue
			}
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

	return filter
}
