package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"

	"github.com/sirupsen/logrus"
)

const (
	setIASTypeTimeout = 10 * time.Minute
)

type IASTypeStep struct {
	bundleBuilder ias.BundleBuilder
}

// ensure the interface is implemented
var _ Step = (*IASTypeStep)(nil)

func NewIASTypeStep(builder ias.BundleBuilder) *IASTypeStep {
	return &IASTypeStep{
		bundleBuilder: builder,
	}
}

func (s *IASTypeStep) Name() string {
	return "IAS_Type"
}

func (s *IASTypeStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	for spID := range ias.ServiceProviderInputs {
		spb, err := s.bundleBuilder.NewBundle(operation.InstanceID, spID)
		if err != nil {
			log.Errorf("%s: %s", "Failed to create ServiceProvider Bundle", err)
			return operation, 0, nil
		}
		err = spb.FetchServiceProviderData()
		if err != nil {
			return s.handleError(operation, err, log, "fetching ServiceProvider data failed")
		}

		log.Infof("Configure SSO Type for ServiceProvider %q with RuntimeURL: %s", spb.ServiceProviderName(), operation.DashboardURL)
		err = spb.ConfigureServiceProviderType(operation.DashboardURL)
		if err != nil {
			return s.handleError(operation, err, log, "setting SSO Type failed")
		}
	}

	return operation, 0, nil
}

func (s *IASTypeStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	switch {
	case kebError.IsTemporaryError(err):
		if time.Since(operation.UpdatedAt) > setIASTypeTimeout {
			log.Errorf("setting IAS type has reached timeout: %s", err)
			// operation will be marked as a success, RuntimeURL will not be set in IAS ServiceProvider application
			return operation, 0, nil
		}
		log.Errorf("setting IAS type cannot be realized", err)
		return operation, 10 * time.Second, nil
	default:
		log.Errorf("setting IAS type failed: %s", err)
		// operation will be marked as a success, RuntimeURL will not be set in IAS ServiceProvider application
		return operation, 0, nil
	}
}
