package suspension

import (
	"errors"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
)

type ContextUpdateHandler struct {
	operations          storage.Operations
	provisioningQueue   Adder
	deprovisioningQueue Adder

	log logrus.FieldLogger
}

type Adder interface {
	Add(processId string)
}

func NewContextUpdateHandler(operations storage.Operations, provisioningQueue Adder, deprovisioningQueue Adder, l logrus.FieldLogger) *ContextUpdateHandler {
	return &ContextUpdateHandler{
		operations:          operations,
		provisioningQueue:   provisioningQueue,
		deprovisioningQueue: deprovisioningQueue,
		log:                 l,
	}
}

// Handle performs suspension/unsuspension for given instance.
// Applies only when 'Active' parameter has changes and ServicePlanID is `Trial`
func (h *ContextUpdateHandler) Handle(instance *internal.Instance, newCtx internal.ERSContext) (string, error) {
	l := h.log.WithFields(logrus.Fields{
		"instanceID":      instance.InstanceID,
		"runtimeID":       instance.RuntimeID,
		"globalAccountID": instance.GlobalAccountID,
	})

	if !broker.IsTrialPlan(instance.ServicePlanID) {
		l.Info("Context update for non-trial instance, skipping")
		return "", nil
	}

	isActivated := true
	if instance.Parameters.ErsContext.Active != nil {
		isActivated = *instance.Parameters.ErsContext.Active
	}

	if newCtx.Active == nil || isActivated == *newCtx.Active {
		logrus.Debugf("Context.Active flag was not changed, the current value: %v", newCtx.Active)
		return "", nil
	}

	if *newCtx.Active {
		return h.unsuspend(instance, l)
	} else {
		return h.suspend(instance, l)
	}
}

func (h *ContextUpdateHandler) suspend(instance *internal.Instance, log logrus.FieldLogger) (string, error) {
	lastDeprovisioning, err := h.operations.GetDeprovisioningOperationByInstanceID(instance.InstanceID)
	// there was an error - fail
	if err != nil && !dberr.IsNotFound(err) {
		return "", err
	}

	// no error, deprovisioning operation exists and is in progress
	if err == nil && (lastDeprovisioning.State == domain.InProgress || lastDeprovisioning.State == orchestration.Pending) {
		if !lastDeprovisioning.Temporary {
			// found ordinary deprovisioning operation in progress - fail suspension
			log.Warnf("Deprovisioning operation already started for instance %s", instance.InstanceID)
			return "", errors.New("cannot suspend instance - deprovisioning operation already started")
		}
		// else found deprovisioning operation is suspension in progress
		return lastDeprovisioning.ID, nil
	}

	id := uuid.New().String()
	operation := internal.NewSuspensionOperationWithID(id, instance)
	err = h.operations.InsertDeprovisioningOperation(operation)
	if err != nil {
		return "", err
	}
	h.deprovisioningQueue.Add(operation.ID)
	return id, nil
}

func (h *ContextUpdateHandler) unsuspend(instance *internal.Instance, log logrus.FieldLogger) (string, error) {
	lastProvisioning, err := h.operations.GetProvisioningOperationByInstanceID(instance.InstanceID)
	// there was an error - fail
	if err != nil && !dberr.IsNotFound(err) {
		return "", err
	}

	// no error, unsuspension operation exists and is in progress
	if err == nil && (lastProvisioning.State == domain.InProgress || lastProvisioning.State == orchestration.Pending) {
		log.Infof("Provisioning operation already started for instance %s", instance.InstanceID)
		return lastProvisioning.ID, nil
	}

	id := uuid.New().String()

	operation, err := internal.NewProvisioningOperationWithID(id, instance.InstanceID, instance.Parameters)
	operation.InstanceDetails = instance.InstanceDetails
	log.Info("Starting unsuspension: shootName=%s shootDomain=%s", operation.ShootName, operation.ShootDomain)
	// RuntimeID must be cleaned  - this mean that there is no runtime in the provisioner/director
	operation.RuntimeID = ""

	err = h.operations.InsertProvisioningOperation(operation)
	if err != nil {
		return "", err
	}
	h.provisioningQueue.Add(operation.ID)
	return id, nil
}
