package model

import (
	"github.com/kyma-project/control-plane/components/provisioner/internal/model/infrastructure/aws"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model/infrastructure/azure"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model/infrastructure/gcp"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model/infrastructure/openstack"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	infrastructureConfigKind = "InfrastructureConfig"
	controlPlaneConfigKind   = "ControlPlaneConfig"

	gcpAPIVersion       = "gcp.provider.extensions.gardener.cloud/v1alpha1"
	azureAPIVersion     = "azure.provider.extensions.gardener.cloud/v1alpha1"
	awsAPIVersion       = "aws.provider.extensions.gardener.cloud/v1alpha1"
	openStackApiVersion = "openstack.provider.extensions.gardener.cloud/v1alpha1"

	defaultConnectionTimeOutMinutes = 4
)

func NewGCPInfrastructure(workerCIDR string) *gcp.InfrastructureConfig {
	return &gcp.InfrastructureConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: gcpAPIVersion,
		},
		Networks: gcp.NetworkConfig{
			Worker:  workerCIDR,
			Workers: util.StringPtr(workerCIDR),
		},
	}
}

func NewGCPControlPlane(zones []string) *gcp.ControlPlaneConfig {
	return &gcp.ControlPlaneConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: gcpAPIVersion,
		},
		Zone: zones[0],
	}
}

func NewAzureInfrastructure(workerCIDR string, azConfig AzureGardenerConfig) *azure.InfrastructureConfig {
	isZoned := len(azConfig.input.Zones) > 0 || len(azConfig.input.AzureZones) > 0
	azureConfig := &azure.InfrastructureConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: azureAPIVersion,
		},
		Networks: azure.NetworkConfig{
			VNet: azure.VNet{
				CIDR: &azConfig.input.VnetCidr,
			},
		},
		Zoned: isZoned,
	}

	if len(azConfig.input.AzureZones) == 0 {
		workers := workerCIDR
		azureConfig.Networks.Workers = &workers
	}
	if isZoned && len(azConfig.input.AzureZones) == 0 && azConfig.input.EnableNatGateway != nil && *azConfig.input.EnableNatGateway {
		natGateway := azure.NatGateway{
			Enabled:                      *azConfig.input.EnableNatGateway,
			IdleConnectionTimeoutMinutes: util.UnwrapIntOrDefault(azConfig.input.IdleConnectionTimeoutMinutes, defaultConnectionTimeOutMinutes),
		}
		azureConfig.Networks.NatGateway = &natGateway
	}
	azureConfig.Networks.Zones = createAzureZones(azConfig.input)

	return azureConfig
}

func createAzureZones(input *gqlschema.AzureProviderConfigInput) []azure.Zone {
	zones := make([]azure.Zone, 0)

	for _, inputZone := range input.AzureZones {
		zone := azure.Zone{
			Name: inputZone.Name,
			CIDR: inputZone.Cidr,
		}
		if input.EnableNatGateway != nil && *input.EnableNatGateway {
			zone.NatGateway = &azure.NatGateway{
				Enabled:                      true,
				IdleConnectionTimeoutMinutes: util.UnwrapIntOrDefault(input.IdleConnectionTimeoutMinutes, defaultConnectionTimeOutMinutes),
			}
		}
		zones = append(zones, zone)
	}

	return zones
}

func NewAzureControlPlane(zones []string) *azure.ControlPlaneConfig {
	return &azure.ControlPlaneConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: azureAPIVersion,
		},
	}
}

func NewAWSInfrastructure(awsConfig AWSGardenerConfig) *aws.InfrastructureConfig {
	return &aws.InfrastructureConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: awsAPIVersion,
		},
		Networks: aws.Networks{
			Zones: createAWSZones(awsConfig.input.AwsZones),
			VPC: aws.VPC{
				CIDR: util.StringPtr(awsConfig.input.VpcCidr),
			},
		},
	}
}

func createAWSZones(inputZones []*gqlschema.AWSZoneInput) []aws.Zone {
	zones := make([]aws.Zone, 0)

	for _, inputZone := range inputZones {
		zone := aws.Zone{
			Name:     inputZone.Name,
			Internal: inputZone.InternalCidr,
			Public:   inputZone.PublicCidr,
			Workers:  inputZone.WorkerCidr,
		}
		zones = append(zones, zone)
	}
	return zones
}

func NewAWSControlPlane() *aws.ControlPlaneConfig {
	return &aws.ControlPlaneConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: awsAPIVersion,
		},
	}
}

func NewOpenStackInfrastructure(floatingPoolName, workerCIDR string) *openstack.InfrastructureConfig {
	return &openstack.InfrastructureConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: openStackApiVersion,
		},
		FloatingPoolName: floatingPoolName,
		Networks: openstack.Networks{
			Workers: workerCIDR,
		},
	}
}

func NewOpenStackControlPlane(loadBalancerProvider string) *openstack.ControlPlaneConfig {
	return &openstack.ControlPlaneConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: openStackApiVersion,
		},
		LoadBalancerProvider: loadBalancerProvider,
	}
}
