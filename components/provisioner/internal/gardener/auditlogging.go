package gardener

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"regexp"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	auditLogConditionType    = "AuditlogServiceAvailability"
	auditInstanceCodePattern = `cf\.[a-z0-9]+`
	auditlogSecretReference  = "auditlog-credentials"
	auditlogExtensionType    = "shoot-auditlog-service"
)

type AuditLogConfigurator interface {
	CanEnableAuditLogsForShoot(seedName string) bool
	ConfigureAuditLogs(logger logrus.FieldLogger, shoot *gardener_types.Shoot, seed gardener_types.Seed) (bool, error)
}

type auditLogConfigurator struct {
	auditLogTenantConfigPath       string
	auditInstanceIdentifierPattern *regexp.Regexp
}

func NewAuditLogConfigurator(auditLogTenantConfigPath string) AuditLogConfigurator {
	return &auditLogConfigurator{
		auditLogTenantConfigPath:       auditLogTenantConfigPath,
		auditInstanceIdentifierPattern: regexp.MustCompile(auditInstanceCodePattern),
	}
}

func (a *auditLogConfigurator) CanEnableAuditLogsForShoot(seedName string) bool {
	return seedName != "" && a.auditLogTenantConfigPath != ""
}

// AuditlogConfig configuration resource
type AuditlogExtensionConfig struct {
	metav1.TypeMeta `json:",inline"`

	// Type is the type of auditlog service provider.
	Type string `json:"type"`
	// TenantID is the id of the tenant.
	TenantID string `json:"tenantID"`
	// ServiceURL is the URL of the auditlog service.
	ServiceURL string `json:"serviceURL"`
	// SecretReferenceName is the name of the reference for the secret containing the auditlog service credentials.
	SecretReferenceName string `json:"secretReferenceName"`
}

func FindSecret(shoot *gardener_types.Shoot) *gardener_types.NamedResourceReference {
	for i, e := range shoot.Spec.Resources {
		if e.Name == auditlogSecretReference {
			return &shoot.Spec.Resources[i]
		}
	}

	return nil
}

func configureSecret(shoot *gardener_types.Shoot, config AuditLogConfig) (changed bool) {
	changed = false

	sec := FindSecret(shoot)
	if sec != nil {
		if sec.Name == auditlogSecretReference &&
			sec.ResourceRef.APIVersion == "v1" &&
			sec.ResourceRef.Kind == "Secret" &&
			sec.ResourceRef.Name == config.SecretName {
			return false
		}
	} else {
		shoot.Spec.Resources = append(shoot.Spec.Resources, gardener_types.NamedResourceReference{})
		sec = &shoot.Spec.Resources[len(shoot.Spec.Resources)-1]
	}

	changed = true

	sec.Name = auditlogSecretReference
	sec.ResourceRef.APIVersion = "v1"
	sec.ResourceRef.Kind = "Secret"
	sec.ResourceRef.Name = config.SecretName

	return
}

func FindExtension(shoot *gardener_types.Shoot) *gardener_types.Extension {
	for i, e := range shoot.Spec.Extensions {
		if e.Type == auditlogExtensionType {
			return &shoot.Spec.Extensions[i]
		}
	}

	return nil
}

func configureExtension(shoot *gardener_types.Shoot, config AuditLogConfig) (changed bool, err error) {
	changed = false
	const (
		extensionKind    = "AuditlogConfig"
		extensionVersion = "service.auditlog.extensions.gardener.cloud/v1alpha1"
		extensionType    = "standard"
	)

	ext := FindExtension(shoot)
	if ext != nil {
		cfg := AuditlogExtensionConfig{}
		err = json.Unmarshal(ext.ProviderConfig.Raw, &cfg)
		if err != nil {
			return
		}

		if cfg.Kind == extensionKind &&
			cfg.Type == extensionType &&
			cfg.TenantID == config.TenantID &&
			cfg.ServiceURL == config.ServiceURL &&
			cfg.SecretReferenceName == auditlogSecretReference {
			return false, nil
		}
	} else {
		shoot.Spec.Extensions = append(shoot.Spec.Extensions, gardener_types.Extension{
			Type: auditlogExtensionType,
		})
		ext = &shoot.Spec.Extensions[len(shoot.Spec.Extensions)-1]
	}

	changed = true

	cfg := AuditlogExtensionConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       extensionKind,
			APIVersion: extensionVersion,
		},
		Type:                extensionType,
		TenantID:            config.TenantID,
		ServiceURL:          config.ServiceURL,
		SecretReferenceName: auditlogSecretReference,
	}

	ext.ProviderConfig = &runtime.RawExtension{}
	ext.ProviderConfig.Raw, err = json.Marshal(cfg)

	return
}

// ConfigureAuditLogs sets up fields required for audit log extension in a given shoot.
// If the shoot is in a region without audit logs configured, it returns an error.
// Returns true if shoot was modified.
func (a *auditLogConfigurator) ConfigureAuditLogs(logger logrus.FieldLogger, shoot *gardener_types.Shoot, seed gardener_types.Seed) (bool, error) {
	auditLogConfig, err := a.getConfigFromFile()
	if err != nil {
		return false, err
	}

	provider := getProviderType(seed)

	providerConfig := auditLogConfig[provider]
	if providerConfig == nil {
		return false, errors.New(fmt.Sprintf("cannot find config for provider %s", provider))
	}

	auditID := a.getAuditLogInstanceIdentifier(seed)
	if auditID == "" {
		return false, errors.New("could not find audit identifier")
	}

	tenant, ok := providerConfig[auditID]
	if !ok {
		return false, errors.New(fmt.Sprintf("tenant for audit identifier %s is empty", auditID))
	}

	changedExt, err := configureExtension(shoot, tenant)
	changedSec := configureSecret(shoot, tenant)

	if changedSec {
		logger.Info("Configured auditlog secret")
	}
	if changedExt {
		logger.Info("Configured auditlog extension")
	}
	return changedExt || changedSec, err
}

type AuditLogConfig struct {
	TenantID   string `json:"tenantID"`
	ServiceURL string `json:"serviceURL"`
	SecretName string `json:"secretName"`
}

func (a *auditLogConfigurator) getConfigFromFile() (data map[string]map[string]AuditLogConfig, err error) {
	file, err := os.Open(a.auditLogTenantConfigPath)

	if err != nil {
		return nil, err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func getProviderType(seed gardener_types.Seed) string {
	return seed.Spec.Provider.Type
}

func (a *auditLogConfigurator) getAuditLogInstanceIdentifier(seed gardener_types.Seed) string {
	message := findAuditLogConditionMessage(seed)

	if message == "" {
		return ""
	}

	return a.auditInstanceIdentifierPattern.FindString(message)
}

func findAuditLogConditionMessage(seed gardener_types.Seed) string {
	conditions := seed.Status.Conditions

	for _, condition := range conditions {
		if condition.Type == auditLogConditionType {
			return condition.Message
		}
	}
	return ""
}
