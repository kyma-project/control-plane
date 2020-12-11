package runtimeversion

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
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
		parameters := internal.ProvisioningParameters{Parameters: internal.ProvisioningParametersDTO{KymaVersion: runtimeVer}}
		rvc := NewRuntimeVersionConfigurator("not-relevant", &AccountVersionMapping{})

		// when
		ver, err := rvc.ForProvisioning(internal.ProvisioningOperation{}, parameters)

		// then
		require.NoError(t, err)
		require.Equal(t, runtimeVer, ver.Version)
		require.Equal(t, internal.Parameters, ver.Origin)
	})
	t.Run("should return version from Defaults when version not provided", func(t *testing.T) {
		// given
		runtimeVer := "1.1.1"
		parameters := internal.ProvisioningParameters{}
		operation := internal.ProvisioningOperation{}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, &AccountVersionMapping{})

		// when
		ver, err := rvc.ForProvisioning(operation, parameters)

		// then
		require.NoError(t, err)
		require.Equal(t, runtimeVer, ver.Version)
		require.Equal(t, internal.Defaults, ver.Origin)
	})
	t.Run("should return version from GlobalAccount mapping when only GlobalAccount mapping provided", func(t *testing.T) {
		// given
		runtimeVer := "1.12"
		parameters := internal.ProvisioningParameters{
			ErsContext: internal.ERSContext{GlobalAccountID: fixGlobalAccountID, SubAccountID: versionForSA},
		}
		operation := internal.ProvisioningOperation{}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): versionForGA,
		}))

		// when
		ver, err := rvc.ForProvisioning(operation, parameters)

		// then
		require.NoError(t, err)
		require.Equal(t, versionForGA, ver.Version)
		require.Equal(t, internal.GlobalAccount, ver.Origin)
	})
	t.Run("should return version from SubAccount mapping when both GA and SA mapping provided", func(t *testing.T) {
		// given
		runtimeVer := "1.12"
		parameters := internal.ProvisioningParameters{
			ErsContext: internal.ERSContext{GlobalAccountID: fixGlobalAccountID,
			SubAccountID: fixSubAccountID},
		}
		operation := internal.ProvisioningOperation{}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): versionForGA,
			fmt.Sprintf("%s%s", subaccountPrefix, fixSubAccountID):    versionForSA,
		}))

		// when
		ver, err := rvc.ForProvisioning(operation, parameters)

		// then
		require.NoError(t, err)
		require.Equal(t, versionForSA, ver.Version)
		require.Equal(t, internal.SubAccount, ver.Origin)
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
