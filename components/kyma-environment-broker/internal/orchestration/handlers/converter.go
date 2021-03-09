package handlers

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
)

type Converter struct{}

func (*Converter) OrchestrationToDTO(o *internal.Orchestration, stats map[string]int) (*orchestration.StatusResponse, error) {
	return &orchestration.StatusResponse{
		OrchestrationID: o.OrchestrationID,
		Type:            o.Type,
		State:           o.State,
		Description:     o.Description,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
		Parameters:      o.Parameters,
		OperationStats:  stats,
	}, nil
}

func (c *Converter) OrchestrationListToDTO(orchestrations []internal.Orchestration, count, totalCount int) (orchestration.StatusResponseList, error) {
	responses := make([]orchestration.StatusResponse, 0)

	for _, o := range orchestrations {
		status, err := c.OrchestrationToDTO(&o, nil)
		if err != nil {
			return orchestration.StatusResponseList{}, errors.Wrap(err, "while converting orchestration to DTO")
		}
		responses = append(responses, *status)
	}

	return orchestration.StatusResponseList{
		Data:       responses,
		Count:      count,
		TotalCount: totalCount,
	}, nil
}

func (c *Converter) UpgradeKymaOperationToDTO(op internal.UpgradeKymaOperation) (orchestration.OperationResponse, error) {
	return orchestration.OperationResponse{
		OperationID:            op.Operation.ID,
		RuntimeID:              op.RuntimeOperation.RuntimeID,
		GlobalAccountID:        op.GlobalAccountID,
		SubAccountID:           op.RuntimeOperation.SubAccountID,
		OrchestrationID:        op.OrchestrationID,
		ServicePlanID:          op.ProvisioningParameters.PlanID,
		ServicePlanName:        broker.PlanNamesMapping[op.ProvisioningParameters.PlanID],
		DryRun:                 op.DryRun,
		ShootName:              op.RuntimeOperation.ShootName,
		MaintenanceWindowBegin: op.MaintenanceWindowBegin,
		MaintenanceWindowEnd:   op.MaintenanceWindowEnd,
		State:                  string(op.Operation.State),
		Description:            op.Operation.Description,
	}, nil
}

func (c *Converter) UpgradeKymaOperationListToDTO(ops []internal.UpgradeKymaOperation, count, totalCount int) (orchestration.OperationResponseList, error) {
	data := make([]orchestration.OperationResponse, 0)

	for _, op := range ops {
		o, err := c.UpgradeKymaOperationToDTO(op)
		if err != nil {
			return orchestration.OperationResponseList{}, errors.Wrap(err, "while converting operation to DTO")
		}
		data = append(data, o)
	}

	return orchestration.OperationResponseList{
		Data:       data,
		Count:      count,
		TotalCount: totalCount,
	}, nil
}

func (c *Converter) UpgradeKymaOperationToDetailDTO(op internal.UpgradeKymaOperation, kymaConfig gqlschema.KymaConfigInput, clusterConfig gqlschema.GardenerConfigInput) (orchestration.OperationDetailResponse, error) {
	resp, err := c.UpgradeKymaOperationToDTO(op)
	if err != nil {
		return orchestration.OperationDetailResponse{}, errors.Wrap(err, "while converting operation to DTO")
	}
	return orchestration.OperationDetailResponse{
		OperationResponse: resp,
		KymaConfig:        kymaConfig,
		ClusterConfig:     clusterConfig,
	}, nil
}
