package provisioning

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/auditlog"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// TODO: delete this step after all SKR clusters are migrated to 1.20!
// the only reason why it's there is the old rigid way of configuring FluentBit (via extra.conf),
// which makes it impossible to decouple CLS overrides from Audit Log overrides (both will end up in the same FluentBit config section)
type ClsAuditLogOverridesStep struct {
	operationManager *process.ProvisionOperationManager
	fs               afero.Fs
	auditLogConfig   auditlog.Config
	secretKey        string
}

func (alo *ClsAuditLogOverridesStep) Name() string {
	return "CLS_Audit_Log_Overrides"
}

func NewClsAuditLogOverridesStep(os storage.Operations, cfg auditlog.Config, secretKey string) *ClsAuditLogOverridesStep {
	fileSystem := afero.NewOsFs()

	return &ClsAuditLogOverridesStep{
		process.NewProvisionOperationManager(os),
		fileSystem,
		cfg,
		secretKey,
	}
}

func (alo *ClsAuditLogOverridesStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	luaScript, err := afero.ReadFile(alo.fs, "/auditlog-script/script")
	if err != nil {
		failureReason := "Unable to read Audit Log config script"
		log.Errorf("%s: %v", failureReason, err)
		return alo.operationManager.OperationFailed(operation, failureReason, log)
	}

	replaceSubAccountID := strings.Replace(string(luaScript), "sub_account_id", operation.ProvisioningParameters.ErsContext.SubAccountID, -1)
	replaceTenantID := strings.Replace(replaceSubAccountID, "tenant_id", alo.auditLogConfig.Tenant, -1)

	auditlogOverrideParams, err := auditlog.PrepareOverrideParams(&alo.auditLogConfig, alo.secretKey, operation.Cls.Overrides)
	if err != nil {
		failureReason := "Unable to prepare Audit Log override parameters"
		log.Errorf("%s: %v", failureReason, err)
		return alo.operationManager.OperationFailed(operation, failureReason, log)
	}

	extraConfTemplate, err := auditlog.GetExtraConfTemplate(operation.RuntimeVersion.Version)
	if err != nil {
		failureReason := "Unable to get Audit Log extra config template"
		log.Errorf("%s: %v", failureReason, err)
		return alo.operationManager.OperationFailed(operation, failureReason, log)
	}

	extraConfOverrides, err := cls.RenderOverrides(auditlogOverrideParams, extraConfTemplate)
	if err != nil {
		failureReason := "Unable to render Audit Log extra config overrides"
		log.Errorf("%s: %v", failureReason, err)
		return alo.operationManager.OperationFailed(operation, failureReason, log)
	}

	operation.InputCreator.AppendOverrides("logging", []*gqlschema.ConfigEntryInput{
		{Key: "fluent-bit.conf.Output.forward.enabled", Value: "false"},
		{Key: "fluent-bit.conf.script", Value: replaceTenantID},
		{Key: "fluent-bit.conf.extra", Value: extraConfOverrides},
		{Key: "fluent-bit.config.outputs.forward.enabled", Value: "false"},
		{Key: "fluent-bit.config.script", Value: replaceTenantID},
		{Key: "fluent-bit.config.extra", Value: extraConfOverrides},
		{Key: "fluent-bit.externalServiceEntry.resolution", Value: "DNS"},
		{Key: "fluent-bit.externalServiceEntry.hosts", Value: fmt.Sprintf(`- %s`, auditlogOverrideParams.Host)},
		{Key: "fluent-bit.externalServiceEntry.ports", Value: fmt.Sprintf(`- number: %s
  name: https
  protocol: TLS`, auditlogOverrideParams.Port)},
	})

	return operation, 0, nil
}
