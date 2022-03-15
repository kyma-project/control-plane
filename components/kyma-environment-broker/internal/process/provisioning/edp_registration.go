package provisioning

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
)

//go:generate mockery -name=EDPClient -output=automock -outpkg=automock -case=underscore
type EDPClient interface {
	CreateDataTenant(data edp.DataTenantPayload) error
	CreateMetadataTenant(name, env string, data edp.MetadataTenantPayload) error

	DeleteDataTenant(name, env string) error
	DeleteMetadataTenant(name, env, key string) error
}

type EDPRegistrationStep struct {
	operationManager *process.ProvisionOperationManager
	client           EDPClient
	config           edp.Config
}

func NewEDPRegistrationStep(os storage.Operations, client EDPClient, config edp.Config) *EDPRegistrationStep {
	return &EDPRegistrationStep{
		operationManager: process.NewProvisionOperationManager(os),
		client:           client,
		config:           config,
	}
}

func (s *EDPRegistrationStep) Name() string {
	return "EDP_Registration"
}

func (s *EDPRegistrationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.EDPCreated {
		return operation, 0, nil
	}
	subAccountID := operation.ProvisioningParameters.ErsContext.SubAccountID

	log.Infof("Create DataTenant for %s subaccount (env=%s)", subAccountID, s.config.Environment)
	err := s.client.CreateDataTenant(edp.DataTenantPayload{
		Name:        subAccountID,
		Environment: s.config.Environment,
		Secret:      s.generateSecret(subAccountID, s.config.Environment),
	})
	if err != nil {
		if edp.IsConflictError(err) {
			log.Warnf("Data Tenant already exists, deleting")
			return s.handleConflict(operation, log)
		}
		return s.handleError(operation, err, log, "cannot create DataTenant")
	}

	log.Infof("Create DataTenant metadata for %s subaccount", subAccountID)
	for key, value := range map[string]string{
		edp.MaasConsumerEnvironmentKey: s.selectEnvironmentKey(operation.ProvisioningParameters.PlatformRegion, log),
		edp.MaasConsumerRegionKey:      operation.ProvisioningParameters.PlatformRegion,
		edp.MaasConsumerSubAccountKey:  subAccountID,
		edp.MaasConsumerServicePlan:    s.selectServicePlan(operation.ProvisioningParameters.PlanID),
	} {
		payload := edp.MetadataTenantPayload{
			Key:   key,
			Value: value,
		}
		log.Infof("Sending metadata %s: %s", payload.Key, payload.Value)
		err = s.client.CreateMetadataTenant(subAccountID, s.config.Environment, payload)
		if err != nil {
			if edp.IsConflictError(err) {
				log.Warnf("Metadata already exists, deleting")
				return s.handleConflict(operation, log)
			}
			return s.handleError(operation, err, log, fmt.Sprintf("cannot create DataTenant metadata %s", key))
		}
	}

	newOp, repeat, _ := s.operationManager.UpdateOperation(operation, func(op *internal.ProvisioningOperation) {
		op.EDPCreated = true
	}, log)
	if repeat != 0 {
		log.Errorf("cannot save operation")
		return newOp, 5 * time.Second, nil
	}

	return newOp, 0, nil
}

func (s *EDPRegistrationStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)

	if kebError.IsTemporaryError(err) {
		since := time.Since(operation.UpdatedAt)
		if since < time.Minute*30 {
			log.Errorf("request to EDP failed: %s. Retry...", err)
			return operation, 10 * time.Second, nil
		}
	}

	if !s.config.Required {
		log.Errorf("Step %s failed. Step is not required. Skip step.", s.Name())
		return operation, 0, nil
	}

	return s.operationManager.OperationFailed(operation, msg, err, log)
}

func (s *EDPRegistrationStep) selectEnvironmentKey(region string, log logrus.FieldLogger) string {
	parts := strings.Split(region, "-")
	switch parts[0] {
	case "cf":
		return "CF"
	case "k8s":
		return "KUBERNETES"
	case "neo":
		return "NEO"
	default:
		log.Warnf("region %s does not fit any of the options, default CF is used", region)
		return "CF"
	}
}

func (s *EDPRegistrationStep) selectServicePlan(planID string) string {
	switch planID {
	case broker.FreemiumPlanID:
		return "free"
	case broker.AzureLitePlanID:
		return "tdd"
	default:
		return "standard"
	}
}

// generateSecret generates secret during dataTenant creation, at this moment the secret is not needed
// except required parameter
func (s *EDPRegistrationStep) generateSecret(name, env string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%s", name, env)))
}

func (s *EDPRegistrationStep) handleConflict(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	for _, key := range []string{
		edp.MaasConsumerEnvironmentKey,
		edp.MaasConsumerRegionKey,
		edp.MaasConsumerSubAccountKey,
		edp.MaasConsumerServicePlan,
	} {
		log.Infof("Deleting DataTenant metadata %s (%s): %s", operation.SubAccountID, s.config.Environment, key)
		err := s.client.DeleteMetadataTenant(operation.SubAccountID, s.config.Environment, key)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("cannot remove DataTenant metadata with key: %s", key))
		}
	}

	log.Infof("Deleting DataTenant %s (%s)", operation.SubAccountID, s.config.Environment)
	err := s.client.DeleteDataTenant(operation.SubAccountID, s.config.Environment)
	if err != nil {
		return s.handleError(operation, err, log, "cannot remove DataTenant")
	}

	log.Infof("Retrying...")
	return operation, time.Second, nil
}
