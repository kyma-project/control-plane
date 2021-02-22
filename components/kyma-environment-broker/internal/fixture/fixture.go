package fixture

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/pivotal-cf/brokerapi/v7/domain"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
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

func FixServiceManagerEntryDTO() *internal.ServiceManagerEntryDTO {
	return &internal.ServiceManagerEntryDTO{
		Credentials: internal.ServiceManagerCredentials{
			BasicAuth: internal.ServiceManagerBasicAuth{
				Username: "username",
				Password: "password",
			},
		},
		URL: "https://service-manager.local",
	}
}

func FixERSContext(id string) internal.ERSContext {
	var (
		tenantID     = fmt.Sprintf("Tenant-%s", id)
		subAccountId = fmt.Sprintf("SA-%s", id)
		userID       = fmt.Sprintf("User-%s", id)
	)

	return internal.ERSContext{
		TenantID:        tenantID,
		SubAccountID:    subAccountId,
		GlobalAccountID: globalAccountId,
		ServiceManager:  FixServiceManagerEntryDTO(),
		Active:          ptr.Bool(true),
		UserID:          userID,
	}
}

func FixProvisioningParametersDTO() internal.ProvisioningParametersDTO {
	trialCloudProvider := internal.Azure
	return internal.ProvisioningParametersDTO{
		Name:           "cluster-name",
		VolumeSizeGb:   ptr.Integer(50),
		MachineType:    ptr.String("MachineType"),
		Region:         ptr.String("Region"),
		Purpose:        ptr.String("Purpose"),
		LicenceType:    ptr.String("LicenceType"),
		AutoScalerMin:  ptr.Integer(3),
		AutoScalerMax:  ptr.Integer(10),
		MaxSurge:       ptr.Integer(4),
		MaxUnavailable: ptr.Integer(1),
		KymaVersion:    kymaVersion,
		Provider:       &trialCloudProvider,
	}
}

func FixProvisioningParameters(id string) internal.ProvisioningParameters {
	return internal.ProvisioningParameters{
		PlanID:         planId,
		ServiceID:      serviceId,
		ErsContext:     FixERSContext(id),
		Parameters:     FixProvisioningParametersDTO(),
		PlatformRegion: "region",
	}
}

func FixInstanceDetails(id string) internal.InstanceDetails {
	var (
		runtimeId    = fmt.Sprintf("Runtime-%s", id)
		subAccountId = fmt.Sprintf("SA-%s", id)
		tenantId     = fmt.Sprintf("Tenant-%s", id)
		bindingId    = fmt.Sprintf("Binding-%s", id)
		brokerId     = fmt.Sprintf("Broker-%s", id)
	)

	lms := internal.LMS{
		TenantID:    tenantId,
		Failed:      false,
		RequestedAt: time.Time{},
	}

	avsLifecycleData := internal.AvsLifecycleData{
		AvsEvaluationInternalId: 1,
		AVSEvaluationExternalId: 2,
		AvsInternalEvaluationStatus: internal.AvsEvaluationStatus{
			Current:  "currentStatus",
			Original: "originalStatus",
		},
		AvsExternalEvaluationStatus: internal.AvsEvaluationStatus{
			Current:  "currentStatus",
			Original: "originalStatus",
		},
		AVSInternalEvaluationDeleted: false,
		AVSExternalEvaluationDeleted: false,
	}

	serviceManagerInstanceInfo := internal.ServiceManagerInstanceInfo{
		BrokerID:              brokerId,
		ServiceID:             serviceId,
		PlanID:                planId,
		InstanceID:            id,
		Provisioned:           false,
		ProvisioningTriggered: false,
	}

	xsuaaData := internal.XSUAAData{
		Instance:  serviceManagerInstanceInfo,
		XSAppname: "xsappName",
		BindingID: bindingId,
	}

	emsData := internal.EmsData{
		Instance:  serviceManagerInstanceInfo,
		BindingID: bindingId,
		Overrides: "overrides",
	}

	return internal.InstanceDetails{
		Lms:          lms,
		Avs:          avsLifecycleData,
		EventHub:     internal.EventHub{Deleted: false},
		SubAccountID: subAccountId,
		RuntimeID:    runtimeId,
		ShootName:    "ShootName",
		ShootDomain:  "ShootDomain",
		XSUAA:        xsuaaData,
		Ems:          emsData,
	}
}

func FixInstance(id string) internal.Instance {
	var (
		runtimeId    = fmt.Sprintf("Runtime-%s", id)
		subAccountId = fmt.Sprintf("SA-%s", id)
	)

	return internal.Instance{
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
		InstanceDetails: internal.InstanceDetails{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now().Add(time.Minute * 5),
		DeletedAt:       time.Now().Add(time.Hour * 1),
		Version:         0,
	}
}

func FixOperation(id, instanceId string) internal.Operation {
	var (
		description     = fmt.Sprintf("Description for operation %s", id)
		orchestrationId = fmt.Sprintf("Orchestration-%s", id)
	)

	return internal.Operation{
		InstanceDetails:        internal.InstanceDetails{},
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
		ShootName:         ptr.String("ShootName"),
	}
}

func FixProvisioningOperation(operationId, instanceId string) internal.ProvisioningOperation {
	return internal.ProvisioningOperation{
		Operation: FixOperation(operationId, instanceId),
		RuntimeVersion: internal.RuntimeVersionData{
			Version: kymaVersion,
			Origin:  internal.Defaults,
		},
		InputCreator:    FixInputCreator(),
		SMClientFactory: nil,
	}
}

func FixDeprovisioningOperation(operationId, instanceId string) internal.DeprovisioningOperation {
	return internal.DeprovisioningOperation{
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

func FixUpgradeKymaOperation(operationId, instanceId string) internal.UpgradeKymaOperation {
	return internal.UpgradeKymaOperation{
		Operation:        FixOperation(operationId, instanceId),
		RuntimeOperation: FixRuntimeOperation(operationId),
		InputCreator:     FixInputCreator(),
		RuntimeVersion: internal.RuntimeVersionData{
			Version: kymaVersion,
			Origin:  internal.Defaults,
		},
		SMClientFactory: nil,
	}
}

func FixOrchestration(id string) internal.Orchestration {
	return internal.Orchestration{
		OrchestrationID: id,
		State:           orchestration.Succeeded,
		Description:     "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now().Add(time.Hour * 1),
		Parameters:      orchestration.Parameters{},
	}
}

// SimpleInputCreator implements ProvisionerInputCreator interface
func (c *SimpleInputCreator) SetProvisioningParameters(params internal.ProvisioningParameters) internal.ProvisionerInputCreator {
	return c
}

func (c *SimpleInputCreator) SetShootName(name string) internal.ProvisionerInputCreator {
	c.ShootName = &name
	return c
}

func (c *SimpleInputCreator) SetLabel(key, val string) internal.ProvisionerInputCreator {
	c.Labels[key] = val
	return c
}

func (c *SimpleInputCreator) SetOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	return c
}

func (c *SimpleInputCreator) AppendOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	c.Overrides[component] = append(c.Overrides[component], overrides...)
	return c
}

func (c *SimpleInputCreator) AppendGlobalOverrides(overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	return c
}

func (c *SimpleInputCreator) CreateProvisionRuntimeInput() (gqlschema.ProvisionRuntimeInput, error) {
	return gqlschema.ProvisionRuntimeInput{}, nil
}

func (c *SimpleInputCreator) CreateUpgradeRuntimeInput() (gqlschema.UpgradeRuntimeInput, error) {
	return gqlschema.UpgradeRuntimeInput{}, nil
}

func (c *SimpleInputCreator) EnableOptionalComponent(name string) internal.ProvisionerInputCreator {
	c.EnabledComponents = append(c.EnabledComponents, name)
	return c
}
