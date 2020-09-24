package handlers

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/pkg/errors"
)

type Converter struct{}

func (*Converter) OrchestrationToDTO(o *internal.Orchestration) (*orchestration.StatusResponse, error) {

	return &orchestration.StatusResponse{
		OrchestrationID: o.OrchestrationID,
		State:           o.State,
		Description:     o.Description,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
		Parameters:      o.Parameters,
	}, nil
}

func (c *Converter) OrchestrationListToDTO(orchestrations []internal.Orchestration) (orchestration.StatusResponseList, error) {
	responses := make([]orchestration.StatusResponse, 0)

	for _, o := range orchestrations {
		status, err := c.OrchestrationToDTO(&o)
		if err != nil {
			return orchestration.StatusResponseList{}, errors.Wrap(err, "while converting orchestration to DTO")
		}

		responses = append(responses, *status)
	}

	return orchestration.StatusResponseList{Orchestrations: responses}, nil
}
