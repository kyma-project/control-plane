package gardener

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const auditLogConditionType = "AuditlogServiceAvailability"
const auditInstanceCodePattern = `cf\.[a-z0-9]+`

type AuditLogConfigurator interface {
	CanEnableAuditLogsForShoot(seedName string) bool
	SetAuditLogAnnotation(shoot *gardener_types.Shoot, seed gardener_types.Seed) (bool, error)
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
	// SecretReferenceName is the name name of the reference for the secret containing the auditlog service credentials.
	SecretReferenceName string `json:"secretReferenceName"`
}

func findSecret(shoot *gardener_types.Shoot) *gardener_types.NamedResourceReference {
	for i, e := range shoot.Spec.Resources {
		if e.Name == "auditlog-credentials" {
			return &shoot.Spec.Resources[i]
		}
	}

	return nil
}

func configureSecret(shoot *gardener_types.Shoot, config AuditLogConfig) (changed bool) {
	changed = false

	sec := findSecret(shoot)
	if sec != nil {
		if sec.Name == "auditlog-credentials" &&
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

	sec.Name = "auditlog-credentials"
	sec.ResourceRef.APIVersion = "v1"
	sec.ResourceRef.Kind = "Secret"
	sec.ResourceRef.Name = config.SecretName

	return
}

func findExtension(shoot *gardener_types.Shoot) *gardener_types.Extension {
	for i, e := range shoot.Spec.Extensions {
		if e.Type == "shoot-auditlog-service" {
			return &shoot.Spec.Extensions[i]
		}
	}

	return nil
}

func configureExtension(shoot *gardener_types.Shoot, config AuditLogConfig) (changed bool, err error) {
	changed = false

	ext := findExtension(shoot)
	if ext != nil {
		cfg := AuditlogExtensionConfig{}
		err = json.Unmarshal(ext.ProviderConfig.Raw, &cfg)
		if err != nil {
			return
		}

		if cfg.Kind == "AuditlogConfig" &&
			cfg.Type == "standard" &&
			cfg.TenantID == config.TenantID &&
			cfg.ServiceURL == config.ServiceURL &&
			cfg.SecretReferenceName == "auditlog-credentials" {
			return false, nil
		}
	} else {
		shoot.Spec.Extensions = append(shoot.Spec.Extensions, gardener_types.Extension{
			Type: "shoot-auditlog-service",
		})
		ext = &shoot.Spec.Extensions[len(shoot.Spec.Extensions)-1]
	}

	changed = true

	cfg := AuditlogExtensionConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AuditlogConfig",
			APIVersion: "service.auditlog.extensions.gardener.cloud/v1alpha1",
		},
		Type:                "standard",
		TenantID:            config.TenantID,
		ServiceURL:          config.ServiceURL,
		SecretReferenceName: "auditlog-credentials",
	}

	ext.ProviderConfig = &runtime.RawExtension{}
	ext.ProviderConfig.Raw, err = json.Marshal(cfg)

	return
}

func (a *auditLogConfigurator) SetAuditLogAnnotation(shoot *gardener_types.Shoot, seed gardener_types.Seed) (bool, error) {
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

	defer file.Close()

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
