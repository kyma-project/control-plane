package gardener

import (
	"encoding/json"
	"path/filepath"
	"testing"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscaling "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

		t.Log(shoot.Spec.Extensions)
		//when
		annotated, err := auditLogConfigurator.SetAuditLogAnnotation(shoot, seed)

		expected := `
            {
                "providerConfig": {
                    "apiVersion": "service.auditlog.extensions.gardener.cloud/v1alpha1",
                    "kind": "AuditlogConfig",
                    "secretReferenceName": "auditlog-credentials",
                    "serviceURL": "https://auditlog.example.com:3000",
                    "tenantID": "a9be5aad-f855-4fd1-a8c8-e95683ec786b",
                    "type": "standard"
                },
                "type": "shoot-auditlog-service"
            }`

		//then
		require.NoError(t, err)
		assert.True(t, annotated)
		t.Log(shoot.Spec.Extensions)

		actual, _ := json.Marshal(shoot.Spec.Extensions[0])
		assert.JSONEq(t, expected, string(actual))
		// assert.Equal(t, expected, shoot.Spec.Extensions[0])
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
		shoot.Spec.Extensions = []gardener_types.Extension{
			{
				Type: "shoot-auditlog-service",
				ProviderConfig: &runtime.RawExtension{
					Raw: []byte(`
{
                    "apiVersion": "service.auditlog.extensions.gardener.cloud/v1alpha1",
                    "kind": "AuditlogConfig",
                    "secretReferenceName": "auditlog-credentials",
                    "serviceURL": "https://auditlog.example.com:3000",
                    "tenantID": "a9be5aad-f855-4fd1-a8c8-e95683ec786b",
                    "type": "standard"
}
`),
				},
			},
		}

		shoot.Spec.Resources = []gardener_types.NamedResourceReference{
			{
				Name: "auditlog-credentials",
				ResourceRef: autoscaling.CrossVersionObjectReference{
					Kind:       "Secret",
					Name:       "auditlog-secret",
					APIVersion: "v1",
				},
			},
		}
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

		expected := `
            {
                "providerConfig": {
                    "apiVersion": "service.auditlog.extensions.gardener.cloud/v1alpha1",
                    "kind": "AuditlogConfig",
                    "secretReferenceName": "auditlog-credentials",
                    "serviceURL": "https://auditlog.example.com:3000",
                    "tenantID": "a9be5aad-f855-4fd1-a8c8-e95683ec786b",
                    "type": "standard"
                },
                "type": "shoot-auditlog-service"
            }`

		require.NoError(t, err)
		assert.False(t, notAnnotated)
		actual, _ := json.Marshal(shoot.Spec.Extensions[0])
		assert.JSONEq(t, expected, string(actual))
	})
}
