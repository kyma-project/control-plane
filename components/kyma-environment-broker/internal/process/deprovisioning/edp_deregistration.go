package deprovisioning

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=EDPClient --output=automock --outpkg=automock --case=underscore
type EDPClient interface {
	DeleteDataTenant(name, env string) error
	DeleteMetadataTenant(name, env, key string) error
}

type EDPDeregistrationStep struct {
	operationManager *process.OperationManager
	client           EDPClient
	config           edp.Config
}

func NewEDPDeregistrationStep(os storage.Operations, client EDPClient, config edp.Config) *EDPDeregistrationStep {
	return &EDPDeregistrationStep{
		operationManager: process.NewOperationManager(os),
		client:           client,
		config:           config,
	}
}

func (s *EDPDeregistrationStep) Name() string {
	return "EDP_Deregistration"
}

func (s *EDPDeregistrationStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	log.Info("Delete DataTenant metadata")

	subAccountID := strings.ToLower(operation.SubAccountID)
	for _, key := range []string{
		edp.MaasConsumerEnvironmentKey,
		edp.MaasConsumerRegionKey,
		edp.MaasConsumerSubAccountKey,
		edp.MaasConsumerServicePlan,
	} {
		err := s.client.DeleteMetadataTenant(subAccountID, s.config.Environment, key)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("cannot remove DataTenant metadata with key: %s", key))
		}
	}

	log.Info("Delete DataTenant")
	err := s.client.DeleteDataTenant(subAccountID, s.config.Environment)
	if err != nil {
		return s.handleError(operation, err, log, "cannot remove DataTenant")
	}

	return operation, 0, nil
}

func (s *EDPDeregistrationStep) handleError(operation internal.Operation, err error, log logrus.FieldLogger, msg string) (internal.Operation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)

	if kebError.IsTemporaryError(err) {
		since := time.Since(operation.UpdatedAt)
		if since < time.Minute*30 {
			log.Errorf("request to EDP failed: %s. Retry...", err)
			return operation, 10 * time.Second, nil
		}
	}

	errMsg := fmt.Sprintf("Step %s failed. EDP data have not been deleted.", s.Name())
	operation, repeat, err := s.operationManager.MarkStepAsExcutedButNotCompleted(operation, s.Name(), errMsg, log)
	if repeat != 0 {
		return operation, repeat, err
	}
	return operation, 0, nil
}
