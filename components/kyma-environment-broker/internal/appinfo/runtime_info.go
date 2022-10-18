package appinfo

import (
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
	"github.com/pkg/errors"
)

//go:generate mockery --name=InstanceFinder --output=automock --outpkg=automock --case=underscore

type (
	InstanceFinder interface {
		FindAllJoinedWithOperations(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, error)
	}

	LastOperationFinder interface {
		GetLastOperation(instanceID string) (*internal.Operation, error)
	}

	ResponseWriter interface {
		InternalServerError(rw http.ResponseWriter, r *http.Request, err error, context string)
	}
)

type RuntimeInfoHandler struct {
	instanceFinder          InstanceFinder
	lastOperationFinder     LastOperationFinder
	respWriter              ResponseWriter
	plansConfig             broker.PlansConfig
	defaultSubaccountRegion string
}

func NewRuntimeInfoHandler(instanceFinder InstanceFinder, lastOpFinder LastOperationFinder, plansConfig broker.PlansConfig, region string, respWriter ResponseWriter) *RuntimeInfoHandler {
	return &RuntimeInfoHandler{
		instanceFinder:          instanceFinder,
		lastOperationFinder:     lastOpFinder,
		respWriter:              respWriter,
		plansConfig:             plansConfig,
		defaultSubaccountRegion: region,
	}
}

func (h *RuntimeInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	allInstances, err := h.instanceFinder.FindAllJoinedWithOperations(predicate.SortAscByCreatedAt())
	if err != nil {
		h.respWriter.InternalServerError(w, r, err, "while fetching all instances")
		return
	}

	dto, err := h.mapToDTO(allInstances)
	if err != nil {
		h.respWriter.InternalServerError(w, r, err, "while mapping instance model to dto")
	}

	if err := httputil.JSONEncode(w, dto); err != nil {
		h.respWriter.InternalServerError(w, r, err, "while encoding response to JSON")
		return
	}
}

func (h *RuntimeInfoHandler) mapToDTO(instances []internal.InstanceWithOperation) ([]*RuntimeDTO, error) {
	items := make([]*RuntimeDTO, 0, len(instances))
	indexer := map[string]int{}
	firstProvOpCreationTimePerInstance := map[string]time.Time{}
	lastDeprovOpCreationTimesPerInstance := map[string]time.Time{}

	for _, inst := range instances {
		region := h.getRegionOrDefault(inst)

		idx, found := indexer[inst.InstanceID]
		if !found {
			// Determine runtime modifiedAt timestamp based on the last operation of the runtime
			lastOp, err := h.lastOperationFinder.GetLastOperation(inst.InstanceID)
			if err != nil && !dberr.IsNotFound(err) {
				return nil, errors.Wrapf(err, "while getting last operation for instance %s", inst.InstanceID)
			}
			updatedAt := inst.UpdatedAt
			if lastOp != nil {
				updatedAt = lastOp.UpdatedAt
			}
			items = append(items, &RuntimeDTO{
				RuntimeID:         inst.RuntimeID,
				SubAccountID:      inst.SubAccountID,
				SubAccountRegion:  region,
				ServiceInstanceID: inst.InstanceID,
				GlobalAccountID:   inst.GlobalAccountID,
				ServiceClassID:    inst.ServiceID,
				ServiceClassName:  svcNameOrDefault(inst),
				ServicePlanID:     inst.ServicePlanID,
				ServicePlanName:   h.planNameOrDefault(inst),
				Status: StatusDTO{
					CreatedAt: getIfNotZero(inst.CreatedAt),
					UpdatedAt: getIfNotZero(updatedAt),
					DeletedAt: getIfNotZero(inst.DeletedAt),
				},
			})
			idx = len(items) - 1
			indexer[inst.InstanceID] = idx
		}

		// TODO: consider to merge the rows in sql query
		opStatus := &OperationStatusDTO{
			State:       inst.State.String,
			Description: inst.Description.String,
		}
		switch internal.OperationType(inst.Type.String) {
		case internal.OperationTypeProvision:
			opCreationTime, exists := firstProvOpCreationTimePerInstance[inst.InstanceID]
			if !exists {
				firstProvOpCreationTimePerInstance[inst.InstanceID] = inst.OpCreatedAt
				items[idx].Status.Provisioning = opStatus
			}
			if inst.OpCreatedAt.Before(opCreationTime) {
				firstProvOpCreationTimePerInstance[inst.InstanceID] = inst.OpCreatedAt
				items[idx].Status.Provisioning = opStatus
			}
		case internal.OperationTypeDeprovision:
			if !inst.IsSuspensionOp {
				opCreationTime, exists := lastDeprovOpCreationTimesPerInstance[inst.InstanceID]
				if !exists {
					lastDeprovOpCreationTimesPerInstance[inst.InstanceID] = inst.OpCreatedAt
					items[idx].Status.Deprovisioning = opStatus
				}
				if inst.OpCreatedAt.After(opCreationTime) {
					lastDeprovOpCreationTimesPerInstance[inst.InstanceID] = inst.OpCreatedAt
					items[idx].Status.Deprovisioning = opStatus
				}
			}
		}
	}

	return items, nil
}

func (h *RuntimeInfoHandler) getRegionOrDefault(inst internal.InstanceWithOperation) string {
	if inst.Parameters.PlatformRegion == "" {
		return h.defaultSubaccountRegion
	}
	return inst.Parameters.PlatformRegion
}

func svcNameOrDefault(inst internal.InstanceWithOperation) string {
	if inst.ServiceName != "" {
		return inst.ServiceName
	}
	return broker.KymaServiceName
}

func (h *RuntimeInfoHandler) planNameOrDefault(inst internal.InstanceWithOperation) string {
	if inst.ServicePlanName != "" {
		return inst.ServicePlanName
	}
	return broker.Plans(h.plansConfig, "", false)[inst.ServicePlanID].Name
}

func getIfNotZero(in time.Time) *time.Time {
	if in.IsZero() {
		return nil
	}
	return ptr.Time(in)
}
