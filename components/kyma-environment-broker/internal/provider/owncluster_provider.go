package provider

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const ()

type NoHyperscalerInput struct {
}

func (p *NoHyperscalerInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{}
}

func (p *NoHyperscalerInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
	// TODO: validate and moge kubeconfit here?
}

func (p *NoHyperscalerInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *NoHyperscalerInput) Provider() internal.CloudProvider {
	return internal.UnknownProvider
}
