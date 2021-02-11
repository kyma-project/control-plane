package internal

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"k8s.io/utils/pointer"
)

const (
	serviceId       = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	serviceName     = "kymaruntime"
	planId          = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	planName        = "azure"
	globalAccountId = "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4"
	kymaVersion     = "1.19.0"
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

func FixProvisioningOperation() ProvisioningOperation {
	return ProvisioningOperation{
		Operation:       Operation{},
		RuntimeVersion:  RuntimeVersionData{},
		InputCreator:    nil,
		SMClientFactory: nil,
	}
}

func FixDeprovisioningOperation() DeprovisioningOperation {
	return DeprovisioningOperation{
		Operation:       Operation{},
		SMClientFactory: nil,
		Temporary:       false,
	}
}

func FixOperation() Operation {
	return Operation{
		InstanceDetails:        InstanceDetails{},
		ID:                     "",
		Version:                0,
		CreatedAt:              time.Time{},
		UpdatedAt:              time.Time{},
		InstanceID:             "",
		ProvisionerOperationID: "",
		State:                  "",
		Description:            "",
		ProvisioningParameters: ProvisioningParameters{},
		OrchestrationID:        "",
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
