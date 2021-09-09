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
		require.Equal(t, 1, ver.MajorVersion)
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
		require.Equal(t, 1, ver.MajorVersion)
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
		require.Equal(t, 1, ver.MajorVersion)
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
		require.Equal(t, 1, ver.MajorVersion)
		require.Equal(t, internal.AccountMapping, ver.Origin)
	})
	t.Run("should return Kyma Version from ProvisioningParameters even when previewVersion provided", func(t *testing.T) {
		// given
		runtimeVer := "1.0.0"
		previewVer := "2.0"
		planID := broker.PreviewPlanID

		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					Parameters: internal.ProvisioningParametersDTO{KymaVersion: runtimeVer},
					PlanID:     planID,
				},
			},
		}

		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, fixAccountVersionMapping(t, map[string]string{}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, runtimeVer, ver.Version)
		require.Equal(t, 1, ver.MajorVersion)
	})
	t.Run("should return previewVersion when version not provided", func(t *testing.T) {
		// given
		runtimeVer := ""
		previewVer := "2.0"
		planID := broker.PreviewPlanID

		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					Parameters: internal.ProvisioningParametersDTO{KymaVersion: runtimeVer},
					PlanID:     planID,
				},
			},
		}

		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, fixAccountVersionMapping(t, map[string]string{}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, previewVer, ver.Version)
		require.Equal(t, 2, ver.MajorVersion)
	})
	t.Run("should return previewVersion when both GA and SA mapping provided", func(t *testing.T) {
		// given
		runtimeVer := "1.0.0"
		previewVer := "2.0"
		planID := broker.PreviewPlanID

		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					PlanID: planID,
					ErsContext: internal.ERSContext{GlobalAccountID: fixGlobalAccountID,
						SubAccountID: fixSubAccountID},
				},
			},
		}

		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): versionForGA,
			fmt.Sprintf("%s%s", subaccountPrefix, fixSubAccountID):       versionForSA,
		}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, previewVer, ver.Version)
		require.Equal(t, 2, ver.MajorVersion)
	})
	t.Run("should return custom version from ProvisioningParameters and default Kyma major version", func(t *testing.T) {
		// given
		runtimeVer := "1.24.5"
		previewVer := "2.0.0"
		customVer := "PR-123"
		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					Parameters: internal.ProvisioningParametersDTO{
						KymaVersion: customVer,
					},
				},
			},
		}

		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, &AccountVersionMapping{})

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, customVer, ver.Version)
		require.Equal(t, 1, ver.MajorVersion)
	})
	t.Run("should return custom version from ProvisioningParameters and default Kyma preview major version", func(t *testing.T) {
		// given
		runtimeVer := "1.24.5"
		previewVer := "2.0.0"
		customVer := "PR-123"
		planID := broker.PreviewPlanID
		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					PlanID: planID,
					Parameters: internal.ProvisioningParametersDTO{
						KymaVersion: customVer,
					},
				},
			},
		}

		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, &AccountVersionMapping{})

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, customVer, ver.Version)
		require.Equal(t, 2, ver.MajorVersion)
	})
	t.Run("should return custom version from GlobalAccount mapping and default Kyma major version when only GlobalAccount mapping provided", func(t *testing.T) {
		// given
		runtimeVer := "1.24.5"
		previewVer := "2.0.0"
		customVerGA := "PR-123"
		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					ErsContext: internal.ERSContext{GlobalAccountID: fixGlobalAccountID, SubAccountID: versionForSA},
				},
			},
		}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): customVerGA,
		}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, customVerGA, ver.Version)
		require.Equal(t, 1, ver.MajorVersion)
		require.Equal(t, internal.AccountMapping, ver.Origin)
	})
	t.Run("should return version from SubAccount mapping and default Kyma major version when both GA and SA mapping provided", func(t *testing.T) {
		// given
		runtimeVer := "1.24.5"
		previewVer := "2.0.0"
		customVerGA := "PR-123"
		customVerSA := "PR-456"
		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					ErsContext: internal.ERSContext{GlobalAccountID: fixGlobalAccountID,
						SubAccountID: fixSubAccountID},
				},
			},
		}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): customVerGA,
			fmt.Sprintf("%s%s", subaccountPrefix, fixSubAccountID):       customVerSA,
		}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, customVerSA, ver.Version)
		require.Equal(t, 1, ver.MajorVersion)
		require.Equal(t, internal.AccountMapping, ver.Origin)
	})
	t.Run("should return preview version and default Kyma preview major version when only custom GlobalAccount mapping provided", func(t *testing.T) {
		// given
		runtimeVer := "1.24.5"
		previewVer := "2.0.0"
		customVerGA := "PR-123"
		planID := broker.PreviewPlanID
		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					PlanID:     planID,
					ErsContext: internal.ERSContext{GlobalAccountID: fixGlobalAccountID, SubAccountID: versionForSA},
				},
			},
		}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): customVerGA,
		}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, previewVer, ver.Version)
		require.Equal(t, 2, ver.MajorVersion)
		require.Equal(t, internal.AccountMapping, ver.Origin)
	})
	t.Run("should return preview version and Kyma preview major version when both GA and SA mapping provided", func(t *testing.T) {
		// given
		runtimeVer := "1.24.5"
		previewVer := "2.0.0"
		customVerGA := "PR-123"
		customVerSA := "PR-456"
		planID := broker.PreviewPlanID
		operation := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ProvisioningParameters: internal.ProvisioningParameters{
					PlanID: planID,
					ErsContext: internal.ERSContext{GlobalAccountID: fixGlobalAccountID,
						SubAccountID: fixSubAccountID},
				},
			},
		}

		rvc := NewRuntimeVersionConfigurator(runtimeVer, previewVer, fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): customVerGA,
			fmt.Sprintf("%s%s", subaccountPrefix, fixSubAccountID):       customVerSA,
		}))

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, previewVer, ver.Version)
		require.Equal(t, 2, ver.MajorVersion)
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
