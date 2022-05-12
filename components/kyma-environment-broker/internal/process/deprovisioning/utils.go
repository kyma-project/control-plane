package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/sirupsen/logrus"
)

func handleError(stepName string, operation internal.DeprovisioningOperation, err error,
	log logrus.FieldLogger, msg string) (internal.DeprovisioningOperation, time.Duration, error) {

	if kebError.IsTemporaryError(err) {
		since := time.Since(operation.UpdatedAt)
		if since < time.Minute*30 {
			log.Errorf("%s: %s. Retry...", msg, err)
			return operation, 10 * time.Second, nil
		}
	}

	log.Errorf("Step %s failed: %s.", stepName, err)
	return operation, 0, nil
}
