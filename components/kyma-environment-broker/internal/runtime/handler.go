package runtime

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"golang.org/x/exp/slices"

	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"
	pkg "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
)

const numberOfUpgradeOperationsToReturn = 2

type Handler struct {
	instancesDb     storage.Instances
	operationsDb    storage.Operations
	runtimeStatesDb storage.RuntimeStates
	converter       Converter

	defaultMaxPage int
}

func NewHandler(instanceDb storage.Instances, operationDb storage.Operations, runtimeStatesDb storage.RuntimeStates, defaultMaxPage int, defaultRequestRegion string) *Handler {
	return &Handler{
		instancesDb:     instanceDb,
		operationsDb:    operationDb,
		runtimeStatesDb: runtimeStatesDb,
		converter:       NewConverter(defaultRequestRegion),
		defaultMaxPage:  defaultMaxPage,
	}
}

func (h *Handler) AttachRoutes(router *mux.Router) {
	router.HandleFunc("/runtimes", h.getRuntimes)
}

func findLastDeprovisioning(operations []internal.Operation) internal.Operation {
	for i := len(operations) - 1; i > 0; i-- {
		o := operations[i]
		if o.Type != internal.OperationTypeDeprovision {
			continue
		}
		if o.State != domain.Succeeded {
			continue
		}
		return o
	}
	return operations[len(operations)-1]
}

func recreateInstances(operations []internal.Operation) []internal.Instance {
	byInstance := make(map[string][]internal.Operation)
	for _, o := range operations {
		byInstance[o.InstanceID] = append(byInstance[o.InstanceID], o)
	}
	var instances []internal.Instance
	for id, op := range byInstance {
		o := op[0]
		last := findLastDeprovisioning(op)
		instances = append(instances, internal.Instance{
			InstanceID:      id,
			GlobalAccountID: o.GlobalAccountID,
			SubAccountID:    o.SubAccountID,
			RuntimeID:       o.RuntimeID,
			CreatedAt:       o.CreatedAt,
			ServicePlanID:   o.ProvisioningParameters.PlanID,
			DeletedAt:       last.UpdatedAt,
			InstanceDetails: last.InstanceDetails,
			Parameters:      last.ProvisioningParameters,
		})
	}
	return instances
}

func unionInstances(sets ...[]internal.Instance) (union []internal.Instance) {
	m := make(map[string]internal.Instance)
	for _, s := range sets {
		for _, i := range s {
			if _, exists := m[i.InstanceID]; !exists {
				m[i.InstanceID] = i
			}
		}
	}
	for _, i := range m {
		union = append(union, i)
	}
	return
}

func (h *Handler) listInstances(filter dbmodel.InstanceFilter) ([]internal.Instance, int, int, error) {
	if slices.Contains(filter.States, dbmodel.InstanceDeprovisioned) {
		// try to list instances where deletion didn't finish successfully
		// entry in the Instances table still exists but has deletion timestamp and contains list of incomplete steps
		deletionAttempted := true
		filter.DeletionAttempted = &deletionAttempted
		instances, instancesCount, instancesTotalCount, _ := h.instancesDb.List(filter)

		// try to recreate instances from the operations table where entry in the instances table is gone
		opFilter := dbmodel.OperationFilter{}
		opFilter.InstanceFilter = &filter
		opFilter.Page = filter.Page
		opFilter.PageSize = filter.PageSize
		operations, _, _, err := h.operationsDb.ListOperations(opFilter)
		if err != nil {
			return instances, instancesCount, instancesTotalCount, err
		}
		instancesFromOperations := recreateInstances(operations)

		// return union of both sets of instances
		instancesUnion := unionInstances(instances, instancesFromOperations)
		count := len(instancesFromOperations)
		return instancesUnion, count + instancesCount, count + instancesTotalCount, nil
	}
	return h.instancesDb.List(filter)
}

func (h *Handler) getRuntimes(w http.ResponseWriter, req *http.Request) {
	toReturn := make([]pkg.RuntimeDTO, 0)

	pageSize, page, err := pagination.ExtractPaginationConfigFromRequest(req, h.defaultMaxPage)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusBadRequest, fmt.Errorf("while getting query parameters: %w", err))
		return
	}
	filter := h.getFilters(req)
	filter.PageSize = pageSize
	filter.Page = page
	opDetail := getOpDetail(req)
	kymaConfig := getBoolParam(pkg.KymaConfigParam, req)
	clusterConfig := getBoolParam(pkg.ClusterConfigParam, req)

	instances, count, totalCount, err := h.listInstances(filter)
	if err != nil {
		httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while fetching instances: %w", err))
		return
	}

	for _, instance := range instances {
		dto, err := h.converter.NewDTO(instance)
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("while converting instance to DTO: %w", err))
			return
		}

		switch opDetail {
		case pkg.AllOperation:
			err = h.setRuntimeAllOperations(instance, &dto)
		case pkg.LastOperation:
			err = h.setRuntimeLastOperation(instance, &dto)
		}
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		err = h.determineStatusModifiedAt(&dto)
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		err = h.setRuntimeOptionalAttributes(instance, &dto, kymaConfig, clusterConfig)
		if err != nil {
			httputil.WriteErrorResponse(w, http.StatusInternalServerError, err)
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

func (h *Handler) determineStatusModifiedAt(dto *pkg.RuntimeDTO) error {
	// Determine runtime modifiedAt timestamp based on the last operation of the runtime
	last, err := h.operationsDb.GetLastOperation(dto.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return fmt.Errorf("while fetching last operation for instance %s: %w", dto.InstanceID, err)
	}
	if last != nil {
		dto.Status.ModifiedAt = last.UpdatedAt
	}
	return nil
}

func (h *Handler) setRuntimeAllOperations(instance internal.Instance, dto *pkg.RuntimeDTO) error {
	provOprs, err := h.operationsDb.ListProvisioningOperationsByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return fmt.Errorf("while fetching provisioning operations list for instance %s: %w", instance.InstanceID, err)
	}
	if len(provOprs) != 0 {
		firstProvOp := &provOprs[len(provOprs)-1]
		lastProvOp := provOprs[0]
		// Set AVS evaluation ID based on the data in the last provisioning operation
		dto.AVSInternalEvaluationID = lastProvOp.InstanceDetails.Avs.AvsEvaluationInternalId
		h.converter.ApplyProvisioningOperation(dto, firstProvOp)
		if len(provOprs) > 1 {
			h.converter.ApplyUnsuspensionOperations(dto, provOprs[:len(provOprs)-1])
		}
	}

	deprovOprs, err := h.operationsDb.ListDeprovisioningOperationsByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return fmt.Errorf("while fetching deprovisioning operations list for instance %s: %w", instance.InstanceID, err)
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
	h.converter.ApplyDeprovisioningOperation(dto, deprovOp)
	h.converter.ApplySuspensionOperations(dto, deprovOprs)

	ukOprs, err := h.operationsDb.ListUpgradeKymaOperationsByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return fmt.Errorf("while fetching upgrade kyma operation for instance %s: %w", instance.InstanceID, err)
	}
	dto.KymaVersion = determineKymaVersion(provOprs, ukOprs)
	ukOprs, totalCount := h.takeLastNonDryRunOperations(ukOprs)
	h.converter.ApplyUpgradingKymaOperations(dto, ukOprs, totalCount)

	ucOprs, err := h.operationsDb.ListUpgradeClusterOperationsByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return fmt.Errorf("while fetching upgrade cluster operation for instance %s: %w", instance.InstanceID, err)
	}
	ucOprs, totalCount = h.takeLastNonDryRunClusterOperations(ucOprs)
	h.converter.ApplyUpgradingClusterOperations(dto, ucOprs, totalCount)

	uOprs, err := h.operationsDb.ListUpdatingOperationsByInstanceID(instance.InstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return fmt.Errorf("while fetching update operation for instance %s: %w", instance.InstanceID, err)
	}
	totalCount = len(uOprs)
	if len(uOprs) > numberOfUpgradeOperationsToReturn {
		uOprs = uOprs[0:numberOfUpgradeOperationsToReturn]
	}
	h.converter.ApplyUpdateOperations(dto, uOprs, totalCount)

	return nil
}

func (h *Handler) setRuntimeLastOperation(instance internal.Instance, dto *pkg.RuntimeDTO) error {
	lastOp, err := h.operationsDb.GetLastOperation(instance.InstanceID)
	if err != nil {
		return fmt.Errorf("while fetching last operation instance %s: %w", instance.InstanceID, err)
	}

	// Set AVS evaluation ID based on the data in the last operation
	dto.AVSInternalEvaluationID = lastOp.InstanceDetails.Avs.AvsEvaluationInternalId

	switch lastOp.Type {
	case internal.OperationTypeProvision:
		provOps, err := h.operationsDb.ListProvisioningOperationsByInstanceID(instance.InstanceID)
		if err != nil {
			return fmt.Errorf("while fetching provisioning operations for instance %s: %w", instance.InstanceID, err)
		}
		lastProvOp := &provOps[0]
		if len(provOps) > 1 {
			h.converter.ApplyUnsuspensionOperations(dto, []internal.ProvisioningOperation{*lastProvOp})
		} else {
			h.converter.ApplyProvisioningOperation(dto, lastProvOp)
		}

	case internal.OperationTypeDeprovision:
		deprovOp, err := h.operationsDb.GetDeprovisioningOperationByID(lastOp.ID)
		if err != nil {
			return fmt.Errorf("while fetching deprovisioning operation for instance %s: %w", instance.InstanceID, err)
		}
		if deprovOp.Temporary {
			h.converter.ApplySuspensionOperations(dto, []internal.DeprovisioningOperation{*deprovOp})
		} else {
			h.converter.ApplyDeprovisioningOperation(dto, deprovOp)
		}

	case internal.OperationTypeUpgradeKyma:
		upgKymaOp, err := h.operationsDb.GetUpgradeKymaOperationByID(lastOp.ID)
		if err != nil {
			return fmt.Errorf("while fetching upgrade kyma operation for instance %s: %w", instance.InstanceID, err)
		}
		h.converter.ApplyUpgradingKymaOperations(dto, []internal.UpgradeKymaOperation{*upgKymaOp}, 1)

	case internal.OperationTypeUpgradeCluster:
		upgClusterOp, err := h.operationsDb.GetUpgradeClusterOperationByID(lastOp.ID)
		if err != nil {
			return fmt.Errorf("while fetching upgrade cluster operation for instance %s: %w", instance.InstanceID, err)
		}
		h.converter.ApplyUpgradingClusterOperations(dto, []internal.UpgradeClusterOperation{*upgClusterOp}, 1)

	case internal.OperationTypeUpdate:
		updOp, err := h.operationsDb.GetUpdatingOperationByID(lastOp.ID)
		if err != nil {
			return fmt.Errorf("while fetching update operation for instance %s: %w", instance.InstanceID, err)
		}
		h.converter.ApplyUpdateOperations(dto, []internal.UpdatingOperation{*updOp}, 1)

	default:
		return fmt.Errorf("unsupported operation type: %s", lastOp.Type)
	}

	return nil
}

func (h *Handler) setRuntimeOptionalAttributes(instance internal.Instance, dto *pkg.RuntimeDTO, kymaConfig, clusterConfig bool) error {
	if kymaConfig || clusterConfig {
		states, err := h.runtimeStatesDb.ListByRuntimeID(instance.RuntimeID)
		if err != nil && !dberr.IsNotFound(err) {
			return fmt.Errorf("while fetching runtime states for instance %s: %w", instance.InstanceID, err)
		}
		for _, state := range states {
			if kymaConfig && dto.KymaConfig == nil && state.KymaConfig.Version != "" {
				config := state.KymaConfig
				dto.KymaConfig = &config
			}
			if clusterConfig && dto.ClusterConfig == nil && state.ClusterConfig.Provider != "" {
				config := state.ClusterConfig
				dto.ClusterConfig = &config
			}
			if dto.KymaConfig != nil && dto.ClusterConfig != nil {
				break
			}
		}
	}

	return nil
}

func determineKymaVersion(pOprs []internal.ProvisioningOperation, uOprs []internal.UpgradeKymaOperation) string {
	kymaVersion := ""
	kymaVersionSetAt := time.Time{}

	// Set kyma version from the last provisioning operation
	if len(pOprs) != 0 {
		kymaVersion = pOprs[0].RuntimeVersion.Version
		kymaVersionSetAt = pOprs[0].CreatedAt
	}

	// Take the last upgrade kyma operation which
	//   - is not dry-run
	//   - is created after the last provisioning operation
	//   - has the kyma version set
	//   - has been processed, i.e. not pending, canceling or canceled
	// Use the last provisioning kyma version if no such upgrade operation was found, or the processed upgrade happened before the last provisioning operation.
	for _, u := range uOprs {
		if !u.DryRun && u.CreatedAt.After(kymaVersionSetAt) && u.RuntimeVersion.Version != "" && u.State != orchestration.Pending && u.State != orchestration.Canceling && u.State != orchestration.Canceled {
			kymaVersion = u.RuntimeVersion.Version
			break
		} else if u.CreatedAt.Before(kymaVersionSetAt) {
			break
		}
	}

	return kymaVersion
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
	filter.Shoots = query[pkg.ShootParam]
	filter.Plans = query[pkg.PlanParam]
	if v, exists := query[pkg.ExpiredParam]; exists && v[0] == "true" {
		filter.Expired = ptr.Bool(true)
	}
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
			case pkg.StateError:
				filter.States = append(filter.States, dbmodel.InstanceError)
			case pkg.StateProvisioning:
				filter.States = append(filter.States, dbmodel.InstanceProvisioning)
			case pkg.StateDeprovisioning:
				filter.States = append(filter.States, dbmodel.InstanceDeprovisioning)
			case pkg.StateUpgrading:
				filter.States = append(filter.States, dbmodel.InstanceUpgrading)
			case pkg.StateUpdating:
				filter.States = append(filter.States, dbmodel.InstanceUpdating)
			case pkg.StateSuspended:
				filter.States = append(filter.States, dbmodel.InstanceDeprovisioned)
			case pkg.StateDeprovisioned:
				filter.States = append(filter.States, dbmodel.InstanceDeprovisioned)
			case pkg.StateDeprovisionIncomplete:
				deletionAttempted := true
				filter.DeletionAttempted = &deletionAttempted
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

func getOpDetail(req *http.Request) pkg.OperationDetail {
	opDetail := pkg.AllOperation
	opDetailParams := req.URL.Query()[pkg.OperationDetailParam]
	for _, p := range opDetailParams {
		opDetailParam := pkg.OperationDetail(p)
		switch opDetailParam {
		case pkg.AllOperation, pkg.LastOperation:
			opDetail = opDetailParam
		}
	}

	return opDetail
}

func getBoolParam(param string, req *http.Request) bool {
	requested := false
	params := req.URL.Query()[param]
	for _, p := range params {
		if p == "true" {
			requested = true
			break
		}
	}

	return requested
}
