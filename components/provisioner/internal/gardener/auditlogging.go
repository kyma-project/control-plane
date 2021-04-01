package gardener

import (
	"encoding/json"
	"errors"
	"fmt"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"os"
)

type AuditLogConfigurator interface {
	CanEnableAuditLogsForShoot(seedName string) bool
	SetAuditLogAnnotation(shoot *gardener_types.Shoot, seed gardener_types.Seed) (bool, error)
}

type auditLogConfigurator struct {
	auditLogTenantConfigPath string
}

func NewAuditLogConfigurator(auditLogTenantConfigPath string) AuditLogConfigurator {
	return &auditLogConfigurator{
		auditLogTenantConfigPath: auditLogTenantConfigPath,
	}
}

func (a *auditLogConfigurator) CanEnableAuditLogsForShoot(seedName string) bool {
	return seedName != "" && a.auditLogTenantConfigPath != ""
}

func (a *auditLogConfigurator) SetAuditLogAnnotation(shoot *gardener_types.Shoot, seed gardener_types.Seed) (bool, error) {
	data, err := a.getConfigFromFile()
	if err != nil {
		return false, err
	}

	provider := getProviderType(seed)

	providerConfig := data[provider]

	if providerConfig == nil {
		return false, errors.New(fmt.Sprintf("cannot find config for provider %s", provider))
	}

	seedRegion := getSeedRegion(seed)

	tenant := providerConfig[seedRegion]

	if tenant == "" {
		return false, errors.New(fmt.Sprintf("tenant for region %s is empty", seedRegion))
	} else if tenant == shoot.Annotations[auditLogsAnnotation] {
		return false, nil
	}

	annotate(shoot, auditLogsAnnotation, tenant)
	return true, nil
}

func (a *auditLogConfigurator) getConfigFromFile() (map[string]map[string]string, error) {
	file, err := os.Open(a.auditLogTenantConfigPath)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	var data map[string]map[string]string
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func getSeedRegion(seed gardener_types.Seed) string {
	return seed.Spec.Provider.Region
}

func getProviderType(seed gardener_types.Seed) string {
	return seed.Spec.Provider.Type
}
