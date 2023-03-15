package broker

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
)

const allSubaccountsIDs = "all"

type GetInstanceEndpoint struct {
	config            Config
	instancesStorage  storage.Instances
	operationsStorage storage.Provisioning
	brokerURL         string

	log logrus.FieldLogger
}

func NewGetInstance(cfg Config,
	instancesStorage storage.Instances,
	operationsStorage storage.Provisioning,
	log logrus.FieldLogger,
) *GetInstanceEndpoint {
	return &GetInstanceEndpoint{
		config:            cfg,
		instancesStorage:  instancesStorage,
		operationsStorage: operationsStorage,
		log:               log.WithField("service", "GetInstanceEndpoint"),
	}
}

// GetInstance fetches information about a service instance
// GET /v2/service_instances/{instance_id}
func (b *GetInstanceEndpoint) GetInstance(_ context.Context, instanceID string, _ domain.FetchInstanceDetails) (domain.GetInstanceDetailsSpec, error) {
	logger := b.log.WithField("instanceID", instanceID)
	logger.Infof("GetInstance called")

	instance, err := b.instancesStorage.GetByID(instanceID)
	if err != nil {
		statusCode := http.StatusNotFound
		if !dberr.IsNotFound(err) {
			statusCode = http.StatusInternalServerError
			return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to get instanceID %s", instanceID), statusCode, fmt.Sprintf("failed to get instanceID %s", instanceID))
		}
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("instance with instanceID %s does not exist", instanceID), statusCode, fmt.Sprintf("instance with instanceID %s does not exist", instanceID))
	}

	// check if provisioning still in progress
	op, err := b.operationsStorage.GetProvisioningOperationByInstanceID(instanceID)
	if err != nil {
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to get operation for instanceID %s", instanceID), http.StatusNotFound, fmt.Sprintf("failed to get operation for instanceID %s", instanceID))
	} else if op.State == domain.InProgress || op.State == domain.Failed {
		err = fmt.Errorf("provisioning of instanceID %s %s", instanceID, op.State)
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(err, http.StatusNotFound, err.Error())
	}

	if !instance.DeletedAt.IsZero() {
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("instance with instanceID %s does not exist", instanceID), http.StatusNotFound, fmt.Sprintf("instance with instanceID %s does not exist", instanceID))
	}

	parameters := b.prepareParametersToReturn(instance.Parameters)

	spec := domain.GetInstanceDetailsSpec{
		ServiceID:    instance.ServiceID,
		PlanID:       instance.ServicePlanID,
		DashboardURL: instance.DashboardURL,
		Parameters:   parameters,
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*op, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel),
		},
	}

	if b.config.ShowTrialExpirationInfo &&
		instance.ServicePlanID == TrialPlanID &&
		(b.config.SubaccountsIdsToShowTrialExpirationInfo == allSubaccountsIDs ||
			strings.Contains(b.config.SubaccountsIdsToShowTrialExpirationInfo, instance.SubAccountID)) {
		spec.Metadata.Labels = ResponseLabelsWithExpirationInfo(*op, *instance, b.config.URL, b.config.TrialDocsURL, b.config.EnableKubeconfigURLLabel)
	}

	return spec, nil
}

func (b *GetInstanceEndpoint) prepareParametersToReturn(parameters internal.ProvisioningParameters) internal.ProvisioningParameters {
	parameters.Parameters.Kubeconfig = ""
	parameters.ErsContext.SMOperatorCredentials = nil
	return parameters
}
