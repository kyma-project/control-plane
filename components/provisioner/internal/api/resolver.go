package api

import (
	"context"

	"github.com/kyma-project/control-plane/components/provisioner/internal/api/middlewares"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

type Resolver struct {
	provisioning  provisioning.Service
	validator     Validator
	tenantUpdater TenantUpdater
}

func (r *Resolver) Mutation() gqlschema.MutationResolver {
	return &Resolver{
		provisioning:  r.provisioning,
		validator:     r.validator,
		tenantUpdater: r.tenantUpdater,
	}
}
func (r *Resolver) Query() gqlschema.QueryResolver {
	return &Resolver{
		provisioning:  r.provisioning,
		validator:     r.validator,
		tenantUpdater: r.tenantUpdater,
	}
}

func NewResolver(provisioningService provisioning.Service, validator Validator, tenantUpdater TenantUpdater) *Resolver {
	return &Resolver{
		provisioning:  provisioningService,
		validator:     validator,
		tenantUpdater: tenantUpdater,
	}
}

func (r *Resolver) ProvisionRuntime(ctx context.Context, config gqlschema.ProvisionRuntimeInput) (*gqlschema.OperationStatus, error) {
	err := r.validator.ValidateProvisioningInput(config)
	if err != nil {
		log.Errorf("Failed to provision Runtime %s", err)
		return nil, err
	}

	tenant, err := r.tenantUpdater.GetTenant(ctx)
	if err != nil {
		log.Errorf("Failed to provision Runtime %s: %s", config.RuntimeInput.Name, err)
		return nil, err
	}

	subAccount := getSubAccount(ctx)

	log.Infof("Requested provisioning of Runtime %s.", config.RuntimeInput.Name)

	operationStatus, err := r.provisioning.ProvisionRuntime(config, tenant, subAccount)
	if err != nil {
		log.Errorf("Failed to provision Runtime %s: %s", config.RuntimeInput.Name, err)
		return nil, err
	}
	log.Infof("Provisioning started for Runtime %s. Operation id %s", config.RuntimeInput.Name, *operationStatus.ID)

	return operationStatus, nil
}

func (r *Resolver) DeprovisionRuntime(ctx context.Context, id string) (string, error) {
	log.Infof("Requested deprovisioning of Runtime %s.", id)

	err := r.tenantUpdater.GetAndUpdateTenant(id, ctx)
	if err != nil {
		log.Errorf("Failed to deprovision Runtime %s: %s", id, err)
		return "", err
	}

	operationID, err := r.provisioning.DeprovisionRuntime(id)
	if err != nil {
		log.Errorf("Failed to deprovision Runtime %s: %s", id, err)
		return "", err
	}
	log.Infof("Deprovisioning started for Runtime %s. Operation id %s", id, operationID)

	return operationID, nil
}

func (r *Resolver) UpgradeRuntime(_ context.Context, runtimeID string, _ gqlschema.UpgradeRuntimeInput) (*gqlschema.OperationStatus, error) {
	message := "Kyma 1.x upgrade operation is no longer supported"

	return &gqlschema.OperationStatus{
		Operation: gqlschema.OperationTypeUpgrade,
		State:     gqlschema.OperationStateFailed,
		RuntimeID: &runtimeID,
		Message:   &message,
	}, nil
}

func (r *Resolver) RollBackUpgradeOperation(context.Context, string) (*gqlschema.RuntimeStatus, error) {
	return nil, errors.New("Kyma 1.x upgrade operation is no longer supported")
}

func (r *Resolver) ReconnectRuntimeAgent(context.Context, string) (string, error) {
	return "", nil
}

func (r *Resolver) RuntimeStatus(ctx context.Context, runtimeID string) (*gqlschema.RuntimeStatus, error) {
	log.Infof("Requested to get status for Runtime %s.", runtimeID)

	err := r.tenantUpdater.GetAndUpdateTenant(runtimeID, ctx)
	if err != nil {
		log.Errorf("Failed to get status for Runtime %s: %s", runtimeID, err)
		return nil, err
	}

	status, err := r.provisioning.RuntimeStatus(runtimeID)
	if err != nil {
		log.Errorf("Failed to get status for Runtime %s: %s", runtimeID, err)
		return nil, err
	}
	log.Infof("Getting status for Runtime %s succeeded.", runtimeID)

	return status, nil
}

func (r *Resolver) RuntimeOperationStatus(ctx context.Context, operationID string) (*gqlschema.OperationStatus, error) {
	log.Infof("Requested to get Runtime operation status for Operation %s.", operationID)

	status, err := r.provisioning.RuntimeOperationStatus(operationID)
	if err != nil {
		log.Errorf("Failed to get Runtime operation status: %s Operation ID: %s", err, operationID)
		return nil, err
	}

	err = r.tenantUpdater.GetAndUpdateTenant(*status.RuntimeID, ctx)
	if err != nil {
		log.Errorf("Failed to get Runtime operation status: %s, Operation ID: %s", err, operationID)
		return nil, err
	}

	log.Infof("Getting Runtime operation status for Operation %s succeeded with last error %+v.", operationID, status.LastError)

	return status, nil
}

func (r *Resolver) UpgradeShoot(ctx context.Context, runtimeID string, input gqlschema.UpgradeShootInput) (*gqlschema.OperationStatus, error) {
	log.Infof("Requested to upgrade Gardener Shoot cluster specification for Runtime : %s.", runtimeID)

	err := r.tenantUpdater.GetAndUpdateTenant(runtimeID, ctx)
	if err != nil {
		log.Errorf("Failed to upgrade Gardener Shoot cluster specification for Runtime  %s: %s", runtimeID, err)
		return nil, err
	}

	err = r.validator.ValidateUpgradeShootInput(input)
	if err != nil {
		log.Errorf("Failed to upgrade Gardener Shoot cluster specification for Runtime %s", err)
		return nil, err
	}

	status, err := r.provisioning.UpgradeGardenerShoot(runtimeID, input)
	if err != nil {
		log.Errorf("Failed to upgrade Gardener Shoot cluster specification for Runtime %s: %s", runtimeID, err)
		return nil, err
	}

	log.Infof("Upgrade Gardener Shoot cluster specification for Runtime %s succeeded", runtimeID)

	return status, nil
}

func (r *Resolver) HibernateRuntime(context.Context, string) (*gqlschema.OperationStatus, error) {
	return nil, nil
}

func getSubAccount(ctx context.Context) string {
	subAccount, ok := ctx.Value(middlewares.SubAccountID).(string)
	if !ok {
		return ""
	}
	return subAccount
}
