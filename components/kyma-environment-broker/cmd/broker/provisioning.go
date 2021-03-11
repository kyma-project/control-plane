package main

import (
	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/lms"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	uaa "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/xsuaa"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

func NewProvisioningProcessingQueue(ctx context.Context, cfg Config, db storage.BrokerStorage, pub event.Publisher,
	directorClient *director.Client,
	provisionerClient provisioner.Client,
	inputFactory input.CreatorForPlan,
	externalEvalCreator *provisioning.ExternalEvalCreator,
	internalEvalUpdater *provisioning.InternalEvalUpdater,
	runtimeVerConfigurator *runtimeversion.RuntimeVersionConfigurator,
	serviceManagerClientFactory *servicemanager.ClientFactory,
	bundleBuilder ias.BundleBuilder,
	lmsTenantManager provisioning.LmsTenantProvider,
	lmsClient lms.Client,
	edpClient *edp.Client,
	runtimeOverrides provisioning.RuntimeOverridesAppender,
	accountProvider hyperscaler.AccountProvider,
	avsDel *avs.Delegator,
	internalEvalAssistant *avs.InternalEvalAssistant,
	logs logrus.FieldLogger) *process.Queue {

	iasTypeSetter := provisioning.NewIASType(bundleBuilder, cfg.IAS.Disabled)

	provisionManager := provisioning.NewManager(db.Operations(), pub, logs.WithField("provisioning", "manager"))

	provisioningInit := provisioning.NewInitialisationStep(db.Operations(), db.Instances(),
		provisionerClient, directorClient, inputFactory, externalEvalCreator, internalEvalUpdater, iasTypeSetter,
		cfg.Provisioning.Timeout, cfg.OperationTimeout, runtimeVerConfigurator, serviceManagerClientFactory)
	provisionManager.InitStep(provisioningInit)

	provisioningSteps := []struct {
		disabled bool
		weight   int
		step     provisioning.Step
	}{
		{
			weight: 1,
			step: provisioning.NewServiceManagerOfferingStep("XSUAA_Offering",
				"xsuaa", "application", func(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
					return &op.XSUAA.Instance
				}, db.Operations()),
			disabled: cfg.XSUAA.Disabled,
		},
		{
			weight: 1,
			step: provisioning.NewServiceManagerOfferingStep("EMS_Offering",
				provisioning.EmsOfferingName, provisioning.EmsPlanName, func(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
					return &op.Ems.Instance
				}, db.Operations()),
			disabled: cfg.Ems.Disabled,
		},
		{
			weight: 2,
			step:   provisioning.NewResolveCredentialsStep(db.Operations(), accountProvider),
		},
		{
			weight: 2,
			step: provisioning.NewXSUAAProvisioningStep(db.Operations(), uaa.Config{
				// todo: set correct values from env variables
				DeveloperGroup:      "devGroup",
				DeveloperRole:       "devRole",
				NamespaceAdminGroup: "nag",
				NamespaceAdminRole:  "nar",
			}),
			disabled: cfg.XSUAA.Disabled,
		},
		{
			weight:   2,
			step:     provisioning.NewEmsProvisionStep(db.Operations()),
			disabled: cfg.Ems.Disabled,
		},
		{
			weight:   2,
			step:     provisioning.NewInternalEvaluationStep(avsDel, internalEvalAssistant),
			disabled: cfg.Avs.Disabled,
		},
		{
			weight: 2,
			step:   provisioning.NewLmsActivationStep(cfg.LMS, provisioning.NewProvideLmsTenantStep(lmsTenantManager, db.Operations(), cfg.LMS.Region, cfg.LMS.Mandatory)),
		},
		{
			weight:   2,
			step:     provisioning.NewEDPRegistrationStep(db.Operations(), edpClient, cfg.EDP),
			disabled: cfg.EDP.Disabled,
		},
		{
			weight: 3,
			step:   provisioning.NewAzureEventHubActivationStep(provisioning.NewProvisionAzureEventHubStep(db.Operations(), azure.NewAzureProvider(), accountProvider, ctx)),
		},
		{
			weight: 3,
			step:   provisioning.NewNatsActivationStep(provisioning.NewNatsStreamingOverridesStep()),
		},
		{
			weight: 3,
			step:   provisioning.NewOverridesFromSecretsAndConfigStep(db.Operations(), runtimeOverrides, runtimeVerConfigurator),
		},
		{
			weight: 3,
			step:   provisioning.NewServiceManagerOverridesStep(db.Operations()),
		},
		{
			weight: 3,
			step:   provisioning.NewAuditLogOverridesStep(db.Operations(), cfg.AuditLog),
		},
		{
			weight: 5,
			step:   provisioning.NewLmsActivationStep(cfg.LMS, provisioning.NewLmsCertificatesStep(lmsClient, db.Operations(), cfg.LMS.Mandatory)),
		},
		{
			weight:   6,
			step:     provisioning.NewIASRegistrationStep(db.Operations(), bundleBuilder),
			disabled: cfg.IAS.Disabled,
		},
		{
			weight:   7,
			step:     provisioning.NewXSUAABindingStep(db.Operations()),
			disabled: cfg.XSUAA.Disabled,
		},
		{
			weight:   7,
			step:     provisioning.NewEmsBindStep(db.Operations(), cfg.Database.SecretKey),
			disabled: cfg.Ems.Disabled,
		},
		{
			weight: 10,
			step:   provisioning.NewCreateRuntimeStep(db.Operations(), db.RuntimeStates(), db.Instances(), provisionerClient),
		},
	}
	for _, step := range provisioningSteps {
		if !step.disabled {
			provisionManager.AddStep(step.weight, step.step)
		}
	}
	provisionQueue := process.NewQueue(provisionManager, logs)
	provisionQueue.Run(ctx.Done(), 5)

	return provisionQueue
}
