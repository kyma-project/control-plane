package internal

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"k8s.io/utils/pointer"
	"time"
)

const (
	kymaVersion = "1.19.0"
)

func fixServiceManagerEntryDTO() *ServiceManagerEntryDTO {
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

func fixProvisioningParametersDTO() ProvisioningParametersDTO {
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

func FixInstance(id string) Instance {
	var (
		instanceId      = fmt.Sprintf("Instance%s", id)
		runtimeId       = fmt.Sprintf("Runtime%s", id)
		globalAccountId = fmt.Sprintf("GlobalAccount%s", id)
		subAccountId    = fmt.Sprintf("SubAccount%s", id)
		serviceId       = fmt.Sprintf("Service%s", id)
		planId          = fmt.Sprintf("Plan%s", id)
		tenantId        = fmt.Sprintf("Tenant%s", id)
		bindingId       = fmt.Sprintf("Binding%s", id)
		brokerId        = fmt.Sprintf("Broker%s", id)
	)

	ersContext := ERSContext{
		TenantID:        tenantId,
		SubAccountID:    subAccountId,
		GlobalAccountID: globalAccountId,
		ServiceManager:  fixServiceManagerEntryDTO(),
		Active:          pointer.BoolPtr(true),
		UserID:          uuid.New().String(),
	}

	provisioningParameters := ProvisioningParameters{
		PlanID:         planId,
		ServiceID:      serviceId,
		ErsContext:     ersContext,
		Parameters:     fixProvisioningParametersDTO(),
		PlatformRegion: "region",
	}

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

	xsuaaData := XSUAAData{
		Instance:  ServiceManagerInstanceInfo{},
		XSAppname: "xsappName",
		BindingID: bindingId,
	}

	emsData := EmsData{
		Instance: ServiceManagerInstanceInfo{
			BrokerID:              brokerId,
			ServiceID:             serviceId,
			PlanID:                planId,
			InstanceID:            instanceId,
			Provisioned:           false,
			ProvisioningTriggered: false,
		},
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
		Parameters:      provisioningParameters,
		ProviderRegion:  "provider-region",
		InstanceDetails: instanceDetails,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now().Add(time.Minute * 5),
		DeletedAt:       time.Now().Add(time.Hour * 1),
		Version:         1,
	}
}
