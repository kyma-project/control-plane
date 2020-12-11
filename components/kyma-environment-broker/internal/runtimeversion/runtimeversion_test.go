package runtimeversion

import (
	"context"
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
}

func Test_NewRuntimeVersionConfigurator_ForProvisioning_FromDefaults(t *testing.T) {
	// given
	runtimeVer := "1.1.1"
	parameters := internal.ProvisioningParameters{}
	operation := internal.ProvisioningOperation{}
	rvc := NewRuntimeVersionConfigurator(runtimeVer, fixGlobalAccountVersionMapping(t))

	// when
	ver, err := rvc.ForProvisioning(operation, parameters)

	// then
	require.NoError(t, err)
	require.Equal(t, runtimeVer, ver.Version)
	require.Equal(t, internal.Defaults, ver.Origin)
}

func Test_NewRuntimeVersionConfigurator_ForProvisioning_FromGlobalAccount(t *testing.T) {
	// given
	runtimeVer := "1.14"
	gaID := "ga-001"
	parameters := internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{GlobalAccountID: gaID},
	}
	operation := internal.ProvisioningOperation{}
	rvc := NewRuntimeVersionConfigurator(runtimeVer, fixGlobalAccountVersionMapping(t))

	// when
	ver, err := rvc.ForProvisioning(operation, parameters)

	// then
	require.NoError(t, err)
	require.Equal(t, runtimeVer, ver.Version)
	require.Equal(t, internal.GlobalAccount, ver.Origin)
}

func fixGlobalAccountVersionMapping(t *testing.T) *AccountVersionMapping {
	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))
	client := fake.NewFakeClientWithScheme(sch, &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"ga-001": "1.14",
			"ga-002": "1.15-rc1",
		},
	})

	return NewAccountVersionMapping(context.TODO(), client, namespace, cmName, logrus.New())
}
