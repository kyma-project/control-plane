package runtimeversion

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_RuntimeVersionConfigurator_ForProvisioning_FromParameters(t *testing.T) {
	t.Run("should return version from ProvisioningParameters when version provided", func(t *testing.T) {
		// given
		runtimeVer := "1.1.1"
		rvc := NewRuntimeVersionConfigurator("not-relevant", "", &AccountVersionMapping{})

		// when
		ver, err := rvc.ForProvisioning(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{Parameters: internal.ProvisioningParametersDTO{KymaVersion: runtimeVer}},
			},
		})

		// then
		require.NoError(t, err)
		require.Equal(t, runtimeVer, ver.Version)
		require.Equal(t, internal.Parameters, ver.Origin)
	})
	t.Run("should return version from Defaults when version not provided", func(t *testing.T) {
		// given
		runtimeVer := "1.1.1"
		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{},
			},
		}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, "", fixAccountVersionMapping(t, map[string]string{}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, runtimeVer, ver.Version)
		require.Equal(t, internal.Defaults, ver.Origin)
	})
	t.Run("should return version from GlobalAccount mapping when only GlobalAccount mapping provided", func(t *testing.T) {
		// given
		runtimeVer := "1.12"
		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					ErsContext: internal.ERSContext{GlobalAccountID: fixGlobalAccountID, SubAccountID: versionForSA},
				},
			},
		}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, "", fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): versionForGA,
		}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, versionForGA, ver.Version)
		require.Equal(t, internal.AccountMapping, ver.Origin)
	})
	t.Run("should return version from SubAccount mapping when both GA and SA mapping provided", func(t *testing.T) {
		// given
		runtimeVer := "1.12"
		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					ErsContext: internal.ERSContext{GlobalAccountID: fixGlobalAccountID,
						SubAccountID: fixSubAccountID},
				},
			},
		}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, "", fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): versionForGA,
			fmt.Sprintf("%s%s", subaccountPrefix, fixSubAccountID):       versionForSA,
		}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, versionForSA, ver.Version)
		require.Equal(t, internal.AccountMapping, ver.Origin)
	})
	t.Run("should return previewVersion when preview plan ID and kymaPreviewVersion provided but no kymaVersion", func(t *testing.T) {
		// given
		runtimeVer := ""
		previewVer := "2.0"
		planID := broker.PreviewPlanID
		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, &AccountVersionMapping{})

		// when
		ver, err := rvc.ForProvisioning(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					Parameters: internal.ProvisioningParametersDTO{KymaVersion: runtimeVer},
					PlanID:     planID,
				},
			},
		})

		// then
		require.NoError(t, err)
		require.Equal(t, previewVer, ver.Version)
	})
	t.Run("should return kymaVersion when preview plan ID, kymaPreviewVersion and kymaVersion are provided", func(t *testing.T) {
		// given
		runtimeVer := "1.1.1"
		previewVer := "2.0"
		planID := broker.PreviewPlanID
		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, &AccountVersionMapping{})

		// when
		ver, err := rvc.ForProvisioning(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					Parameters: internal.ProvisioningParametersDTO{KymaVersion: runtimeVer},
					PlanID:     planID,
				},
			},
		})

		// then
		require.NoError(t, err)
		require.Equal(t, runtimeVer, ver.Version)
	})
}

func fixAccountVersionMapping(t *testing.T, mapping map[string]string) *AccountVersionMapping {
	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))
	client := fake.NewFakeClientWithScheme(sch, &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
		},
		Data: mapping,
	})

	return NewAccountVersionMapping(context.TODO(), client, namespace, cmName, logrus.New())
}
