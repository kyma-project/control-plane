package internal

import (
	"fmt"
	"time"

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

func FixProvisioningOperation(operationId, instanceId string) ProvisioningOperation {
	return ProvisioningOperation{
		Operation: FixOperation(operationId, instanceId),
		RuntimeVersion: RuntimeVersionData{
			Version: kymaVersion,
			Origin:  Defaults,
		},
		InputCreator:    nil,
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
		InputCreator:     nil,
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
