package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
)

type LmsTenantProvider interface {
	ProvideLMSTenantID(name, region string) (string, error)
}

// provideLmsTenantStep creates (if not exists) LMS tenant and provides its ID.
// The step does not breaks the provisioning flow.
type provideLmsTenantStep struct {
	LmsStep
	tenantProvider   LmsTenantProvider
	operationManager *process.ProvisionOperationManager
	regionOverride   string
}

func NewProvideLmsTenantStep(tp LmsTenantProvider, repo storage.Operations, regionOverride string, isMandatory bool) *provideLmsTenantStep {
	return &provideLmsTenantStep{
		LmsStep: LmsStep{
			operationManager: process.NewProvisionOperationManager(repo),
			isMandatory:      isMandatory,
			expirationTime:   3 * time.Minute,
		},
		operationManager: process.NewProvisionOperationManager(repo),
		tenantProvider:   tp,
		regionOverride:   regionOverride,
	}
}

func (s *provideLmsTenantStep) Name() string {
	return "Create_LMS_Tenant"
}

func (s *provideLmsTenantStep) Run(operation internal.ProvisioningOperation, logger logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.Lms.TenantID != "" {
		return operation, 0, nil
	}

	region := s.provideRegion(operation.ProvisioningParameters.Parameters.Region)
	lmsTenantID, err := s.tenantProvider.ProvideLMSTenantID(operation.ProvisioningParameters.ErsContext.GlobalAccountID, region)
	if err != nil {
		return s.handleError(
			operation,
			logger,
			time.Since(operation.UpdatedAt),
			fmt.Sprintf("Unable to get tenant for GlobalaccountID/region %s/%s", operation.ProvisioningParameters.ErsContext.GlobalAccountID, region),
			err)
	}

	op, repeat := s.operationManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
		operation.Lms.TenantID = lmsTenantID
		if operation.Lms.RequestedAt.IsZero() {
			operation.Lms.RequestedAt = time.Now()
		}
	})
	if repeat != 0 {
		logger.Errorf("cannot save LMS tenant ID")
		return operation, time.Second, nil
	}

	return op, 0, nil
}

var lmsRegionsMap = map[string]string{
	"westeurope":    "eu",
	"eastus":        "us",
	"eastus2":       "us",
	"centralus":     "us",
	"northeurope":   "eu",
	"southeastasia": "aus",
	"japaneast":     "aus",
	"westus2":       "eu",
	"uksouth":       "eu",
	"FranceCentral": "eu",
	"EastUS2EUAP":   "us",
	"uaenorth":      "eu",
}

func (s *provideLmsTenantStep) provideRegion(r *string) string {
	if s.regionOverride != "" {
		return s.regionOverride
	}
	if r == nil {
		return "eu"
	}
	region, found := lmsRegionsMap[*r]
	if !found {
		return "eu"
	}
	return region
}
