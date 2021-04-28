package gardener

import (
	"path/filepath"
	"testing"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAuditLogConfigurator_CanEnableAuditLogsForShoot(t *testing.T) {
	t.Run("should return true when seedName and auditLogTenantConfigPath are not empty", func(t *testing.T) {
		//given
		auditLogConfigurator := NewAuditLogConfigurator("/path")
		seedName := "az-eu3"

		//when
		enable := auditLogConfigurator.CanEnableAuditLogsForShoot(seedName)

		//then
		assert.True(t, enable)
	})

	t.Run("should return false when auditLogTenantConfigPath is empty", func(t *testing.T) {
		//given
		auditLogConfigurator := NewAuditLogConfigurator("")
		seedName := "az-eu3"

		//when
		enable := auditLogConfigurator.CanEnableAuditLogsForShoot(seedName)

		//then
		assert.False(t, enable)
	})

	t.Run("should return false when seedName is empty", func(t *testing.T) {
		//given
		auditLogConfigurator := NewAuditLogConfigurator("/path")
		seedName := ""

		//when
		enable := auditLogConfigurator.CanEnableAuditLogsForShoot(seedName)

		//then
		assert.False(t, enable)
	})
}

func TestAuditLogConfigurator_SetAuditLogAnnotation(t *testing.T) {
	t.Run("should annotate shoot and return true", func(t *testing.T) {
		//given
		shoot := &gardener_types.Shoot{}
		seed := gardener_types.Seed{
			ObjectMeta: v1.ObjectMeta{
				Name: "az-eu",
			},
			Spec: gardener_types.SeedSpec{
				Provider: gardener_types.SeedProvider{
					Type: "azure"},
			},
			Status: gardener_types.SeedStatus{Conditions: []gardener_types.Condition{
				{Type: auditLogConditionType,
					Message: "Auditlog landscape https://api.auditlog.cf.us21.hana.ondemand.com:8081/ successfully attached to the seed.",
				},
			}},
		}

		configPath := filepath.Join("testdata", "config.json")

		auditLogConfigurator := NewAuditLogConfigurator(configPath)

		//when
		annotated, err := auditLogConfigurator.SetAuditLogAnnotation(shoot, seed)

		//then
		require.NoError(t, err)
		assert.True(t, annotated)
		assert.Equal(t, "e7382275-e835-4549-94e1-3b1101e3a1fa", shoot.Annotations[auditLogsAnnotation])
	})

	t.Run("should return error when config for provider is empty", func(t *testing.T) {
		//given
		shoot := &gardener_types.Shoot{}
		seed := gardener_types.Seed{
			ObjectMeta: v1.ObjectMeta{
				Name: "az-eu",
			},
			Spec: gardener_types.SeedSpec{
				Provider: gardener_types.SeedProvider{
					Type: "glazure"},
			},
			Status: gardener_types.SeedStatus{Conditions: []gardener_types.Condition{
				{Type: auditLogConditionType,
					Message: "Auditlog landscape https://api.auditlog.cf.us21.hana.ondemand.com:8081/ successfully attached to the seed.",
				},
			}},
		}

		configPath := filepath.Join("testdata", "config.json")

		auditLogConfigurator := NewAuditLogConfigurator(configPath)

		//when
		annotated, err := auditLogConfigurator.SetAuditLogAnnotation(shoot, seed)

		//then
		require.Error(t, err)
		assert.False(t, annotated)
		assert.Empty(t, shoot.Annotations[auditLogsAnnotation])
	})

	t.Run("should return error when cannot find audit log landscape identifier", func(t *testing.T) {
		//given
		shoot := &gardener_types.Shoot{}
		seed := gardener_types.Seed{
			ObjectMeta: v1.ObjectMeta{
				Name: "az-eu",
			},
			Spec: gardener_types.SeedSpec{
				Provider: gardener_types.SeedProvider{
					Type: "azure"},
			},
			Status: gardener_types.SeedStatus{Conditions: []gardener_types.Condition{
				{Type: auditLogConditionType,
					Message: "",
				},
			}},
		}

		configPath := filepath.Join("testdata", "config.json")

		auditLogConfigurator := NewAuditLogConfigurator(configPath)

		//when
		annotated, err := auditLogConfigurator.SetAuditLogAnnotation(shoot, seed)

		//then
		require.Error(t, err)
		assert.False(t, annotated)
		assert.Empty(t, shoot.Annotations[auditLogsAnnotation])
	})

	t.Run("should return error when cannot open config file", func(t *testing.T) {
		//given
		shoot := &gardener_types.Shoot{}
		seed := gardener_types.Seed{
			ObjectMeta: v1.ObjectMeta{
				Name: "az-eu",
			},
			Spec: gardener_types.SeedSpec{
				Provider: gardener_types.SeedProvider{
					Type: "azure"},
			},
			Status: gardener_types.SeedStatus{Conditions: []gardener_types.Condition{
				{Type: auditLogConditionType,
					Message: "Auditlog landscape https://api.auditlog.cf.us21.hana.ondemand.com:8081/ successfully attached to the seed.",
				},
			}},
		}

		configPath := filepath.Join("testdata", "wrongconfig.json")

		auditLogConfigurator := NewAuditLogConfigurator(configPath)

		//when
		annotated, err := auditLogConfigurator.SetAuditLogAnnotation(shoot, seed)

		//then
		require.Error(t, err)
		assert.False(t, annotated)
		assert.Empty(t, shoot.Annotations[auditLogsAnnotation])
	})

	t.Run("should return false when shoot is already anotated", func(t *testing.T) {
		//given
		shoot := &gardener_types.Shoot{}
		shoot.Annotations = map[string]string{}
		shoot.Annotations[auditLogsAnnotation] = "e7382275-e835-4549-94e1-3b1101e3a1fa"
		seed := gardener_types.Seed{
			ObjectMeta: v1.ObjectMeta{
				Name: "az-eu",
			},
			Spec: gardener_types.SeedSpec{
				Provider: gardener_types.SeedProvider{
					Type: "azure"},
			},
			Status: gardener_types.SeedStatus{Conditions: []gardener_types.Condition{
				{Type: auditLogConditionType,
					Message: "Auditlog landscape https://api.auditlog.cf.us21.hana.ondemand.com:8081/ successfully attached to the seed.",
				},
			}},
		}

		configPath := filepath.Join("testdata", "config.json")

		auditLogConfigurator := NewAuditLogConfigurator(configPath)

		//when
		notAnnotated, err := auditLogConfigurator.SetAuditLogAnnotation(shoot, seed)

		//then
		require.NoError(t, err)
		assert.False(t, notAnnotated)
		assert.Equal(t, "e7382275-e835-4549-94e1-3b1101e3a1fa", shoot.Annotations[auditLogsAnnotation])
	})
}
