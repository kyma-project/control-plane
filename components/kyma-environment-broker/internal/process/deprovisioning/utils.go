package deprovisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/sirupsen/logrus"
)

func handleError(stepName string, operation internal.Operation, err error,
	log logrus.FieldLogger, msg string) (internal.Operation, time.Duration, error) {

	if kebError.IsTemporaryError(err) {
		if time.Since(operation.CreatedAt) < 30*time.Minute {
			log.Errorf("%s: %s. Retry...", msg, err)
			return operation, 10 * time.Second, nil
		}
	}

	log.Errorf("Step %s failed: %s.", stepName, err)
	return operation, 0, nil
}
