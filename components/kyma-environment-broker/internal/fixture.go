package internal

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"k8s.io/utils/pointer"
	"time"
)

const (
	kymaVersion = "1.19.0"
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
		tenantId        = fmt.Sprintf("Tenant-%s", id)
		subAccountId    = fmt.Sprintf("SubAccount-%s", id)
		globalAccountId = fmt.Sprintf("GlobalAccount-%s", id)
		userId          = fmt.Sprintf("User-%s", id)
	)

	return ERSContext{
		TenantID:        tenantId,
		SubAccountID:    subAccountId,
		GlobalAccountID: globalAccountId,
		ServiceManager:  FixServiceManagerEntryDTO(),
		Active:          pointer.BoolPtr(true),
		UserID:          userId,
	}
}

func FixProvisioningParametersDTO() ProvisioningParametersDTO {
	trialCloudProvider := TrialCloudProvider("provider")
	return ProvisioningParametersDTO{
		Name:                        "cluster-name",
		TargetSecret:                pointer.StringPtr("targetSecret"),
		VolumeSizeGb:                ptr.Integer(50),
		MachineType:                 pointer.StringPtr("machineType"),
		Region:                      pointer.StringPtr("region"),
		Purpose:                     pointer.StringPtr("purpose"),
		LicenceType:                 pointer.StringPtr("licenceType"),
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
	var (
		planId    = fmt.Sprintf("Plan-%s", id)
		serviceId = fmt.Sprintf("Service-%s", id)
	)

	return ProvisioningParameters{
		PlanID:         planId,
		ServiceID:      serviceId,
		ErsContext:     FixERSContext(id),
		Parameters:     FixProvisioningParametersDTO(),
		PlatformRegion: "platformRegion",
	}
}

func FixInstance(id string) Instance {
	var (
		instanceId      = fmt.Sprintf("Instance-%s", id)
		runtimeId       = fmt.Sprintf("Runtime-%s", id)
		globalAccountId = fmt.Sprintf("GlobalAccount-%s", id)
		subAccountId    = fmt.Sprintf("SubAccount-%s", id)
		serviceId       = fmt.Sprintf("Service-%s", id)
		planId          = fmt.Sprintf("Plan-%s", id)
		tenantId        = fmt.Sprintf("Tenant-%s", id)
		bindingId       = fmt.Sprintf("Binding-%s", id)
		brokerId        = fmt.Sprintf("Broker-%s", id)
	)

	lms := LMS{
		TenantID:    tenantId,
		Failed:      false,
		RequestedAt: time.Now(),
	}

	avsLifecycleData := AvsLifecycleData{
		AvsEvaluationInternalId: 1,
		AVSEvaluationExternalId: 2,
		AvsInternalEvaluationStatus: AvsEvaluationStatus{
			Current:  "currentStatus",
			Original: "originalStatus",
		},
		AvsExternalEvaluationStatus: AvsEvaluationStatus{
			Current:  "currentStatus",
			Original: "originalStatus",
		},
		AVSInternalEvaluationDeleted: false,
		AVSExternalEvaluationDeleted: false,
	}

	serviceManagerInstanceInfo := ServiceManagerInstanceInfo{
		BrokerID:              brokerId,
		ServiceID:             serviceId,
		PlanID:                planId,
		InstanceID:            instanceId,
		Provisioned:           false,
		ProvisioningTriggered: false,
	}

	xsuaaData := XSUAAData{
		Instance:  serviceManagerInstanceInfo,
		XSAppname: "xsappName",
		BindingID: bindingId,
	}

	emsData := EmsData{
		Instance: serviceManagerInstanceInfo,
		BindingID: bindingId,
		Overrides: "overrides",
	}

	instanceDetails := InstanceDetails{
		Lms:          lms,
		Avs:          avsLifecycleData,
		EventHub:     EventHub{Deleted: false},
		SubAccountID: subAccountId,
		RuntimeID:    runtimeId,
		ShootName:    "shootName",
		ShootDomain:  "shootDomain",
		XSUAA:        xsuaaData,
		Ems:          emsData,
	}

	return Instance{
		InstanceID:      instanceId,
		RuntimeID:       runtimeId,
		GlobalAccountID: globalAccountId,
		SubAccountID:    subAccountId,
		ServiceID:       serviceId,
		ServiceName:     "ServiceName",
		ServicePlanID:   planId,
		ServicePlanName: "ServicePlanName",
		DashboardURL:    "https://dashboard.local",
		Parameters:      FixProvisioningParameters(id),
		ProviderRegion:  "provider-region",
		InstanceDetails: instanceDetails,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now().Add(time.Minute * 5),
		DeletedAt:       time.Now().Add(time.Hour * 1),
		Version:         1,
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
