package internal

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/pivotal-cf/brokerapi/v7/domain"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"k8s.io/utils/pointer"
)

const (
	serviceId              = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	serviceName            = "kymaruntime"
	planId                 = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	planName               = "azure"
	globalAccountId        = "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4"
	provisionerOperationId = "e04de524-53b3-4890-b05a-296be393e4ba"
	kymaVersion            = "1.19.0"
)

type SimpleInputCreator struct {
	Overrides         map[string][]*gqlschema.ConfigEntryInput
	Labels            map[string]string
	EnabledComponents []string
	ShootName         *string
}

func FixServiceManagerEntryDTO() *ServiceManagerEntryDTO {
	return &ServiceManagerEntryDTO{
		Credentials: ServiceManagerCredentials{
			BasicAuth: ServiceManagerBasicAuth{
				Username: "username",
				Password: "password",
			},
		},
		URL: "https://service-manager.local",
	}
}

func FixERSContext(id string) ERSContext {
	var (
		tenantID     = fmt.Sprintf("Tenant-%s", id)
		subAccountId = fmt.Sprintf("SA-%s", id)
		userID       = fmt.Sprintf("User-%s", id)
	)

	return ERSContext{
		TenantID:        tenantID,
		SubAccountID:    subAccountId,
		GlobalAccountID: globalAccountId,
		ServiceManager:  FixServiceManagerEntryDTO(),
		Active:          pointer.BoolPtr(true),
		UserID:          userID,
	}
}

func FixProvisioningParametersDTO() ProvisioningParametersDTO {
	trialCloudProvider := TrialCloudProvider("provider")
	return ProvisioningParametersDTO{
		Name:                        "cluster-name",
		TargetSecret:                pointer.StringPtr("TargetSecret"),
		VolumeSizeGb:                ptr.Integer(50),
		MachineType:                 pointer.StringPtr("MachineType"),
		Region:                      pointer.StringPtr("Region"),
		Purpose:                     pointer.StringPtr("Purpose"),
		LicenceType:                 pointer.StringPtr("LicenceType"),
		Zones:                       []string{"1", "2"},
		AutoScalerMin:               ptr.Integer(3),
		AutoScalerMax:               ptr.Integer(10),
		MaxSurge:                    ptr.Integer(4),
		MaxUnavailable:              ptr.Integer(1),
		OptionalComponentsToInstall: []string{"component1", "component2"},
		KymaVersion:                 kymaVersion,
		Provider:                    &trialCloudProvider,
	}
}

func FixProvisioningParameters(id string) ProvisioningParameters {
	return ProvisioningParameters{
		PlanID:         planId,
		ServiceID:      serviceId,
		ErsContext:     FixERSContext(id),
		Parameters:     FixProvisioningParametersDTO(),
		PlatformRegion: "region",
	}
}

func FixInstance(id string) Instance {
	var (
		runtimeId    = fmt.Sprintf("Runtime-%s", id)
		subAccountId = fmt.Sprintf("SA-%s", id)
	)

	return Instance{
		InstanceID:      id,
		RuntimeID:       runtimeId,
		GlobalAccountID: globalAccountId,
		SubAccountID:    subAccountId,
		ServiceID:       serviceId,
		ServiceName:     serviceName,
		ServicePlanID:   planId,
		ServicePlanName: planName,
		DashboardURL:    "https://dashboard.local",
		Parameters:      FixProvisioningParameters(id),
		ProviderRegion:  "region",
		InstanceDetails: InstanceDetails{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now().Add(time.Minute * 5),
		DeletedAt:       time.Now().Add(time.Hour * 1),
		Version:         0,
	}
}

func FixOperation(id, instanceId string) Operation {
	var (
		description     = fmt.Sprintf("Description for operation %s", id)
		orchestrationId = fmt.Sprintf("Orchestration-%s", id)
	)

	return Operation{
		InstanceDetails:        InstanceDetails{},
		ID:                     id,
		Version:                0,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now().Add(time.Hour * 48),
		InstanceID:             instanceId,
		ProvisionerOperationID: provisionerOperationId,
		State:                  domain.Succeeded,
		Description:            description,
		ProvisioningParameters: FixProvisioningParameters(id),
		OrchestrationID:        orchestrationId,
	}
}

func FixInputCreator() *SimpleInputCreator {
	return &SimpleInputCreator{
		Overrides:         make(map[string][]*gqlschema.ConfigEntryInput, 0),
		Labels:            make(map[string]string),
		EnabledComponents: []string{},
		ShootName:         pointer.StringPtr("ShootName"),
	}
}

func FixProvisioningOperation(operationId, instanceId string) ProvisioningOperation {
	return ProvisioningOperation{
		Operation: FixOperation(operationId, instanceId),
		RuntimeVersion: RuntimeVersionData{
			Version: kymaVersion,
			Origin:  Defaults,
		},
		InputCreator:    FixInputCreator(),
		SMClientFactory: nil,
	}
}

func FixDeprovisioningOperation(operationId, instanceId string) DeprovisioningOperation {
	return DeprovisioningOperation{
		Operation:       FixOperation(operationId, instanceId),
		SMClientFactory: nil,
		Temporary:       false,
	}
}

func FixRuntime(id string) orchestration.Runtime {
	var (
		instanceId   = fmt.Sprintf("Instance-%s", id)
		subAccountId = fmt.Sprintf("SA-%s", id)
	)

	return orchestration.Runtime{
		InstanceID:             instanceId,
		RuntimeID:              id,
		GlobalAccountID:        globalAccountId,
		SubAccountID:           subAccountId,
		ShootName:              "ShootName",
		MaintenanceWindowBegin: time.Now().Truncate(time.Millisecond).Add(time.Hour),
		MaintenanceWindowEnd:   time.Now().Truncate(time.Millisecond).Add(time.Minute).Add(time.Hour),
	}
}

func FixRuntimeOperation(operationId string) orchestration.RuntimeOperation {
	return orchestration.RuntimeOperation{
		Runtime: orchestration.Runtime{},
		ID:      operationId,
		DryRun:  false,
	}
}

func FixUpgradeKymaOperation(operationId, instanceId string) UpgradeKymaOperation {
	return UpgradeKymaOperation{
		Operation:        FixOperation(operationId, instanceId),
		RuntimeOperation: FixRuntimeOperation(operationId),
		InputCreator:     FixInputCreator(),
		RuntimeVersion: RuntimeVersionData{
			Version: kymaVersion,
			Origin:  Defaults,
		},
		SMClientFactory: nil,
	}
}

func FixOrchestration() Orchestration {
	return Orchestration{
		OrchestrationID: "",
		State:           "",
		Description:     "",
		CreatedAt:       time.Time{},
		UpdatedAt:       time.Time{},
		Parameters:      orchestration.Parameters{},
	}
}

// SimpleInputCreator implements ProvisionerInputCreator interface
func (c *SimpleInputCreator) SetProvisioningParameters(params ProvisioningParameters) ProvisionerInputCreator {
	return c
}

func (c *SimpleInputCreator) SetShootName(name string) ProvisionerInputCreator {
	c.ShootName = &name
	return c
}

func (c *SimpleInputCreator) SetLabel(key, val string) ProvisionerInputCreator {
	c.Labels[key] = val
	return c
}

func (c *SimpleInputCreator) SetOverrides(component string, overrides []*gqlschema.ConfigEntryInput) ProvisionerInputCreator {
	return c
}

func (c *SimpleInputCreator) AppendOverrides(component string, overrides []*gqlschema.ConfigEntryInput) ProvisionerInputCreator {
	c.Overrides[component] = append(c.Overrides[component], overrides...)
	return c
}

func (c *SimpleInputCreator) AppendGlobalOverrides(overrides []*gqlschema.ConfigEntryInput) ProvisionerInputCreator {
	return c
}

func (c *SimpleInputCreator) CreateProvisionRuntimeInput() (gqlschema.ProvisionRuntimeInput, error) {
	return gqlschema.ProvisionRuntimeInput{}, nil
}

func (c *SimpleInputCreator) CreateUpgradeRuntimeInput() (gqlschema.UpgradeRuntimeInput, error) {
	return gqlschema.UpgradeRuntimeInput{}, nil
}

func (c *SimpleInputCreator) EnableOptionalComponent(name string) ProvisionerInputCreator {
	c.EnabledComponents = append(c.EnabledComponents, name)
	return c
}
