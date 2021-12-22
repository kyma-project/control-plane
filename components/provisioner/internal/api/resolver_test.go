package api_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/api"

	"github.com/kyma-project/control-plane/components/provisioner/internal/api/middlewares"
	validatorMocks "github.com/kyma-project/control-plane/components/provisioner/internal/api/mocks"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	operationID = "ec781980-0533-4098-aab7-96b535569732"
	runtimeID   = "1100bb59-9c40-4ebb-b846-7477c4dc5bbb"
)

func TestResolver_ProvisionRuntime(t *testing.T) {
	ctx := context.WithValue(context.Background(), middlewares.Tenant, tenant)

	clusterConfig := &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			KubernetesVersion:      "1.15.4",
			VolumeSizeGb:           util.IntPtr(30),
			MachineType:            "n1-standard-4",
			Region:                 "europe",
			Provider:               "gcp",
			Seed:                   util.StringPtr(""),
			TargetSecret:           "test-secret",
			DiskType:               util.StringPtr("ssd"),
			WorkerCidr:             "10.10.10.10/255",
			AutoScalerMin:          1,
			AutoScalerMax:          3,
			MaxSurge:               40,
			MaxUnavailable:         1,
			ProviderSpecificConfig: nil,
			OidcConfig:             oidcInput(),
			DNSConfig:              dnsInput(),
		},
	}

	runtimeInput := &gqlschema.RuntimeInput{
		Name:        "test runtime",
		Description: new(string),
	}

	t.Run("Should start provisioning and return operation ID", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}
		resolver := api.NewResolver(provisioningService, validator, tenantUpdater)

		tenantUpdater.On("GetTenant", ctx).Return(tenant, nil)

		kymaConfig := &gqlschema.KymaConfigInput{
			Version: "1.5",
			Components: []*gqlschema.ComponentConfigurationInput{
				{
					Component:     "core",
					Configuration: nil,
				},
			},
		}

		operation := &gqlschema.OperationStatus{
			ID:        util.StringPtr(operationID),
			Operation: gqlschema.OperationTypeProvision,
			State:     gqlschema.OperationStateInProgress,
			Message:   util.StringPtr("Message"),
			RuntimeID: util.StringPtr(runtimeID),
		}

		config := gqlschema.ProvisionRuntimeInput{
			RuntimeInput:  runtimeInput,
			ClusterConfig: clusterConfig,
			KymaConfig:    kymaConfig,
		}

		provisioningService.On("ProvisionRuntime", config, tenant, "").Return(operation, nil)
		validator.On("ValidateProvisioningInput", config).Return(nil)

		//when
		status, err := resolver.ProvisionRuntime(ctx, config)

		//then
		require.NoError(t, err)
		require.NotNil(t, status)
		require.NotNil(t, status.ID)
		require.NotNil(t, status.RuntimeID)
		assert.Equal(t, operationID, *status.ID)
		assert.Equal(t, runtimeID, *status.RuntimeID)
		assert.Equal(t, gqlschema.OperationStateInProgress, status.State)
		assert.Equal(t, gqlschema.OperationTypeProvision, status.Operation)
		assert.Equal(t, util.StringPtr("Message"), status.Message)
	})

	t.Run("Should return error when Kyma config validation fails", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}
		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		kymaConfig := &gqlschema.KymaConfigInput{
			Version: "1.5",
		}

		config := gqlschema.ProvisionRuntimeInput{RuntimeInput: runtimeInput, ClusterConfig: clusterConfig, KymaConfig: kymaConfig}

		tenantUpdater.On("GetTenant", ctx).Return(tenant, nil)
		validator.On("ValidateProvisioningInput", config).Return(apperrors.BadRequest("Some error"))

		//when
		status, err := provisioner.ProvisionRuntime(ctx, config)

		//then
		require.Error(t, err)
		assert.Nil(t, status)
	})

	t.Run("Should return error when provisioning fails", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}
		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		kymaConfig := &gqlschema.KymaConfigInput{
			Version: "1.5",
			Components: []*gqlschema.ComponentConfigurationInput{
				{
					Component:     "core",
					Configuration: nil,
				},
			},
		}

		config := gqlschema.ProvisionRuntimeInput{RuntimeInput: runtimeInput, ClusterConfig: clusterConfig, KymaConfig: kymaConfig}

		tenantUpdater.On("GetTenant", ctx).Return(tenant, nil)
		provisioningService.On("ProvisionRuntime", config, tenant, "").Return(nil, apperrors.Internal("Provisioning failed"))
		validator.On("ValidateProvisioningInput", config).Return(nil)

		//when
		status, err := provisioner.ProvisionRuntime(ctx, config)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)
		assert.Nil(t, status)
	})

	t.Run("Should fail when tenant header is not passed to context", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}
		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		kymaConfig := &gqlschema.KymaConfigInput{
			Version: "1.5",
			Components: []*gqlschema.ComponentConfigurationInput{
				{
					Component:     "core",
					Configuration: nil,
				},
			},
		}

		config := gqlschema.ProvisionRuntimeInput{
			RuntimeInput:  runtimeInput,
			ClusterConfig: clusterConfig,
			KymaConfig:    kymaConfig,
		}

		ctx := context.Background()

		tenantUpdater.On("GetTenant", ctx).Return("", apperrors.BadRequest("missing tenant header"))
		validator.On("ValidateProvisioningInput", config).Return(nil)

		//when
		status, err := provisioner.ProvisionRuntime(ctx, config)

		//then
		require.Error(t, err)
		assert.Nil(t, status)
	})
}

func TestResolver_DeprovisionRuntime(t *testing.T) {
	ctx := context.WithValue(context.Background(), middlewares.Tenant, tenant)

	t.Run("Should start deprovisioning and return operation ID", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}
		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		expectedID := "ec781980-0533-4098-aab7-96b535569732"

		provisioningService.On("DeprovisionRuntime", runtimeID).Return(expectedID, nil)
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		//when
		operationID, err := provisioner.DeprovisionRuntime(ctx, runtimeID)

		//then
		require.NoError(t, err)
		assert.Equal(t, expectedID, operationID)
	})

	t.Run("Should return error when deprovisioning fails", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}
		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)
		provisioningService.On("DeprovisionRuntime", runtimeID).Return("", apperrors.Internal("Deprovisioning fails because reasons"))
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		//when
		operationID, err := provisioner.DeprovisionRuntime(ctx, runtimeID)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)
		assert.Empty(t, operationID)
	})

	t.Run("Should fail when tenant header is not passed to context", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}
		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)
		expectedID := "ec781980-0533-4098-aab7-96b535569732"

		ctx := context.Background()

		provisioningService.On("DeprovisionRuntime", runtimeID).Return(expectedID, nil, nil)
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(apperrors.BadRequest("tenant header not passed"))

		//when
		operationID, err := provisioner.DeprovisionRuntime(ctx, runtimeID)

		//then
		require.Error(t, err)
		require.Empty(t, operationID)
	})
}

func TestResolver_UpgradeRuntime(t *testing.T) {
	ctx := context.WithValue(context.Background(), middlewares.Tenant, tenant)

	upgradeInput := gqlschema.UpgradeRuntimeInput{
		KymaConfig: fixKymaGraphQLConfigInput(),
	}

	t.Run("Should start upgrade and return operation id", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		operation := &gqlschema.OperationStatus{
			ID:        util.StringPtr(operationID),
			Operation: gqlschema.OperationTypeUpgrade,
			State:     gqlschema.OperationStateInProgress,
			Message:   util.StringPtr("Message"),
			RuntimeID: util.StringPtr(runtimeID),
		}

		provisioningService.On("UpgradeRuntime", runtimeID, upgradeInput).Return(operation, nil)
		validator.On("ValidateUpgradeInput", upgradeInput).Return(nil)
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		resolver := api.NewResolver(provisioningService, validator, tenantUpdater)

		//when
		status, err := resolver.UpgradeRuntime(ctx, runtimeID, upgradeInput)

		//then
		require.NoError(t, err)
		require.NotNil(t, status)
		require.NotNil(t, status.ID)
		require.NotNil(t, status.RuntimeID)
		assert.Equal(t, operation, status)
	})

	t.Run("Should return error when upgrade fails", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		provisioningService.On("UpgradeRuntime", runtimeID, upgradeInput).Return(nil, apperrors.Internal("error"))
		validator.On("ValidateUpgradeInput", upgradeInput).Return(nil)
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		resolver := api.NewResolver(provisioningService, validator, tenantUpdater)

		//when
		_, err := resolver.UpgradeRuntime(ctx, runtimeID, upgradeInput)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)
	})

	t.Run("Should return error when validation fails", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		validator.On("ValidateUpgradeInput", upgradeInput).Return(apperrors.BadRequest("error"))
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		resolver := api.NewResolver(provisioningService, validator, tenantUpdater)

		//when
		_, err := resolver.UpgradeRuntime(ctx, runtimeID, upgradeInput)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeBadRequest)
	})
}

func TestResolver_RollBackUpgradeOperation(t *testing.T) {
	ctx := context.WithValue(context.Background(), middlewares.Tenant, tenant)

	runtimeStatus := gqlschema.RuntimeStatus{}

	t.Run("Should start upgrade and return operation id", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		provisioningService.On("RollBackLastUpgrade", runtimeID).Return(&runtimeStatus, nil)
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		resolver := api.NewResolver(provisioningService, validator, tenantUpdater)

		//when
		status, err := resolver.RollBackUpgradeOperation(ctx, runtimeID)

		//then
		require.NoError(t, err)
		require.NotNil(t, status)
	})

	t.Run("Should return error when failed to roll back upgrade", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		provisioningService.On("RollBackLastUpgrade", runtimeID).Return(nil, apperrors.Internal("error"))
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		resolver := api.NewResolver(provisioningService, validator, tenantUpdater)

		//when
		_, err := resolver.RollBackUpgradeOperation(ctx, runtimeID)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)
	})
}

func TestResolver_RuntimeStatus(t *testing.T) {
	ctx := context.WithValue(context.Background(), middlewares.Tenant, tenant)
	runtimeID := "1100bb59-9c40-4ebb-b846-7477c4dc5bbd"

	t.Run("Should return operation status", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		operationID := "acc5040c-3bb6-47b8-8651-07f6950bd0a7"
		message := "some message"

		status := &gqlschema.RuntimeStatus{
			LastOperationStatus: &gqlschema.OperationStatus{
				ID:        &operationID,
				Operation: gqlschema.OperationTypeProvision,
				State:     gqlschema.OperationStateInProgress,
				RuntimeID: &runtimeID,
				Message:   &message,
			},
			RuntimeConfiguration:    &gqlschema.RuntimeConfig{},
			RuntimeConnectionStatus: &gqlschema.RuntimeConnectionStatus{},
		}

		provisioningService.On("RuntimeStatus", runtimeID).Return(status, nil)
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		//when
		runtimeStatus, err := provisioner.RuntimeStatus(ctx, runtimeID)

		//then
		require.NoError(t, err)
		assert.Equal(t, status, runtimeStatus)
	})

	t.Run("Should return error when runtime status fails", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		provisioningService.On("RuntimeStatus", runtimeID).Return(nil, apperrors.Internal("Runtime status fails"))
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		//when
		status, err := provisioner.RuntimeStatus(ctx, runtimeID)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)
		require.Empty(t, status)
	})
}

func TestResolver_RuntimeOperationStatus(t *testing.T) {
	ctx := context.WithValue(context.Background(), middlewares.Tenant, tenant)
	runtimeID := "1100bb59-9c40-4ebb-b846-7477c4dc5bbd"

	t.Run("Should return operation status", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		operationID := "acc5040c-3bb6-47b8-8651-07f6950bd0a7"
		message := "some message"

		operationStatus := &gqlschema.OperationStatus{
			ID:        &operationID,
			Operation: gqlschema.OperationTypeProvision,
			State:     gqlschema.OperationStateInProgress,
			RuntimeID: &runtimeID,
			Message:   &message,
		}

		provisioningService.On("RuntimeOperationStatus", operationID).Return(operationStatus, nil)
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		//when
		status, err := provisioner.RuntimeOperationStatus(ctx, operationID)

		//then
		require.NoError(t, err)
		assert.Equal(t, operationStatus, status)
	})

	t.Run("Should return error when Runtime Operation fails", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		validator.On("ValidateTenantForOperation", operationID, tenant).Return(nil)
		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		provisioningService.On("RuntimeOperationStatus", operationID).Return(nil, apperrors.Internal("Some error"))
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		//when
		status, err := provisioner.RuntimeOperationStatus(ctx, operationID)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)
		require.Empty(t, status)
	})
}

func TestResolver_UpgradeShoot(t *testing.T) {
	ctx := context.WithValue(context.Background(), middlewares.Tenant, tenant)

	upgradeShootInput := NewUpgradeShootInput()

	t.Run("Should start shoot upgrade and return operation id", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		operation := &gqlschema.OperationStatus{
			ID:        util.StringPtr(operationID),
			Operation: gqlschema.OperationTypeUpgradeShoot,
			State:     gqlschema.OperationStateInProgress,
			Message:   util.StringPtr("Message"),
			RuntimeID: util.StringPtr(runtimeID),
		}

		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)
		validator.On("ValidateUpgradeShootInput", upgradeShootInput).Return(nil)
		provisioningService.On("UpgradeGardenerShoot", runtimeID, upgradeShootInput).Return(operation, nil)

		resolver := api.NewResolver(provisioningService, validator, tenantUpdater)

		//when
		status, err := resolver.UpgradeShoot(ctx, runtimeID, upgradeShootInput)

		//then
		require.NoError(t, err)
		require.NotNil(t, status)
		require.NotNil(t, status.ID)
		require.NotNil(t, status.RuntimeID)
		assert.Equal(t, operation, status)
	})

	t.Run("Should return error when validation fails", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		validator.On("ValidateUpgradeShootInput", upgradeShootInput).Return(apperrors.BadRequest("error"))
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		resolver := api.NewResolver(provisioningService, validator, tenantUpdater)

		//when
		_, err := resolver.UpgradeShoot(ctx, runtimeID, upgradeShootInput)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeBadRequest)
	})
}

func TestResolver_HibernateRuntime(t *testing.T) {
	ctx := context.WithValue(context.Background(), middlewares.Tenant, tenant)
	runtimeID := "1100bb59-9c40-4ebb-b846-7477c4dc5bbd"

	t.Run("Should hibernate cluster", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		operationID := "acc5040c-3bb6-47b8-8651-07f6950bd0a7"
		message := "some message"

		operationStatus := &gqlschema.OperationStatus{
			ID:        &operationID,
			Operation: gqlschema.OperationTypeHibernate,
			State:     gqlschema.OperationStateInProgress,
			RuntimeID: &runtimeID,
			Message:   &message,
		}

		provisioningService.On("HibernateCluster", runtimeID).Return(operationStatus, nil)
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		//when
		status, err := provisioner.HibernateRuntime(ctx, runtimeID)

		//then
		require.NoError(t, err)
		assert.Equal(t, operationStatus, status)
	})

	t.Run("Should return error when hibernation fails", func(t *testing.T) {
		//given
		provisioningService := &mocks.Service{}
		validator := &validatorMocks.Validator{}
		tenantUpdater := &validatorMocks.TenantUpdater{}

		provisioner := api.NewResolver(provisioningService, validator, tenantUpdater)

		provisioningService.On("HibernateCluster", runtimeID).Return(nil, apperrors.Internal("Some error"))
		tenantUpdater.On("GetAndUpdateTenant", runtimeID, ctx).Return(nil)

		//when
		status, err := provisioner.HibernateRuntime(ctx, runtimeID)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)
		require.Empty(t, status)
	})
}

func oidcInput() *gqlschema.OIDCConfigInput {
	return &gqlschema.OIDCConfigInput{
		ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb2222",
		GroupsClaim:    "groups",
		IssuerURL:      "https://kymatest.accounts400.ondemand.com",
		SigningAlgs:    []string{"RS257"},
		UsernameClaim:  "sup",
		UsernamePrefix: "-",
	}
}

func dnsInput() *gqlschema.DNSConfigInput {
	return &gqlschema.DNSConfigInput{
		Providers: []*gqlschema.DNSProviderInput{
			&gqlschema.DNSProviderInput{
				DomainsInclude: []string{"devtest.kyma.ondemand.com"},
				Primary:        true,
				SecretName:     "aws_dns_domain_secrets_test_inresolver",
				Type:           "route53_type_test",
			},
		},
	}
}
