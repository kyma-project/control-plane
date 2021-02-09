package internal

import (
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
		Name:                        "ProvisioningParametersDTO",
		TargetSecret:                pointer.StringPtr("targetSecret"),
		VolumeSizeGb:                ptr.Integer(1),
		MachineType:                 pointer.StringPtr("machine"),
		Region:                      pointer.StringPtr("region"),
		Purpose:                     pointer.StringPtr("purpose"),
		LicenceType:                 pointer.StringPtr("licenceType"),
		Zones:                       []string{"zone1", "zone2"},
		AutoScalerMin:               ptr.Integer(2),
		AutoScalerMax:               ptr.Integer(10),
		MaxSurge:                    ptr.Integer(999),
		MaxUnavailable:              ptr.Integer(99),
		OptionalComponentsToInstall: []string{"component1", "component2"},
		KymaVersion:                 kymaVersion,
		Provider:                    &trialCloudProvider,
	}
}

func FixInstance() Instance {
	var (
		globalAccountID = uuid.New().String()
		subAccountID    = uuid.New().String()
		serviceID       = uuid.New().String()
		planID          = uuid.New().String()
	)

	ersContext := ERSContext{
		TenantID:        uuid.New().String(),
		SubAccountID:    subAccountID,
		GlobalAccountID: globalAccountID,
		ServiceManager:  fixServiceManagerEntryDTO(),
		Active:          pointer.BoolPtr(true),
		UserID:          uuid.New().String(),
	}

	provisioningParameters := ProvisioningParameters{
		PlanID:         planID,
		ServiceID:      serviceID,
		ErsContext:     ersContext,
		Parameters:     fixProvisioningParametersDTO(),
		PlatformRegion: "region",
	}

	return Instance{
		InstanceID:      uuid.New().String(),
		RuntimeID:       uuid.New().String(),
		GlobalAccountID: globalAccountID,
		SubAccountID:    subAccountID,
		ServiceID:       serviceID,
		ServiceName:     "ServiceName",
		ServicePlanID:   planID,
		ServicePlanName: "ServicePlanName",
		DashboardURL:    "https://dashboard.local",
		Parameters:      provisioningParameters,
		ProviderRegion:  "provider-region",
		InstanceDetails: InstanceDetails{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now().Add(time.Minute * 5),
		DeletedAt:       time.Now().Add(time.Hour * 1),
		Version:         1,
	}
}
