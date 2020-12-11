package runtimeversion

import (
	"fmt"
	"testing"

	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	cmName             = "config"
	namespace          = "foo"
	versionForGA       = "1.14"
	versionForSA       = "1.15-rc1"
	fixGlobalAccountID = "628ee42b-bd1e-42b3-8a1d-c4726fd2ee62\n"
	fixSubAccountID    = "e083d3a8-5139-4705-959f-8279c86f6fe7\n"
)

func TestAccountVersionMapping_Get(t *testing.T) {
	t.Run("Should get version for SubAccount when both GlobalAccount and SubAccount are provided", func(t *testing.T) {
		// given
		sch := runtime.NewScheme()
		require.NoError(t, coreV1.AddToScheme(sch))
		client := fake.NewFakeClientWithScheme(sch, &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      cmName,
				Namespace: namespace,
			},
			Data: map[string]string{
				fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): versionForGA,
				fmt.Sprintf("%s%s", subaccountPrefix, fixSubAccountID):    versionForSA,
			},
		})


	svc := NewAccountVersionMapping(context.TODO(), client, namespace, cmName, logrus.New())

	// when
	version, origin, found, err := svc.Get(fixGlobalAccountID, fixSubAccountID)
	require.NoError(t, err)

	// then
	assert.True(t, found)
	assert.Equal(t, versionForSA, version)
	assert.Equal(t, internal.SubAccount, origin)
	})

	t.Run("Should get version for GlobalAccount when only GlobalAccount is provided", func(t *testing.T) {
		// given
		sch := runtime.NewScheme()
		require.NoError(t, coreV1.AddToScheme(sch))
		client := fake.NewFakeClientWithScheme(sch, &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      cmName,
				Namespace: namespace,
			},
			Data: map[string]string{
				fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): versionForGA,
			},
		})


		svc := NewAccountVersionMapping(context.TODO(), client, namespace, cmName, logrus.New())

		// when
		version, origin, found, err := svc.Get(fixGlobalAccountID, fixSubAccountID)
		require.NoError(t, err)

		// then
		assert.True(t, found)
		assert.Equal(t, versionForGA, version)
		assert.Equal(t, internal.GlobalAccount, origin)
	})

	t.Run("Should get version for SubAccount when only SubAccount is provided", func(t *testing.T) {
		// given
		sch := runtime.NewScheme()
		require.NoError(t, coreV1.AddToScheme(sch))
		client := fake.NewFakeClientWithScheme(sch, &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      cmName,
				Namespace: namespace,
			},
			Data: map[string]string{
				fmt.Sprintf("%s%s", subaccountPrefix, fixSubAccountID):    versionForSA,
			},
		})


		svc := NewAccountVersionMapping(context.TODO(), client, namespace, cmName, logrus.New())

		// when
		version, origin, found, err := svc.Get(fixGlobalAccountID, fixSubAccountID)
		require.NoError(t, err)

		// then
		assert.True(t, found)
		assert.Equal(t, versionForSA, version)
		assert.Equal(t, internal.SubAccount, origin)
	})

	t.Run("Should not get version when nothing is provided", func(t *testing.T) {
		// given
		sch := runtime.NewScheme()
		require.NoError(t, coreV1.AddToScheme(sch))
		client := fake.NewFakeClientWithScheme(sch, &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      cmName,
				Namespace: namespace,
			},
			Data: map[string]string{
				"not-existing": "version-mapping-1.0",
			},
		})


		svc := NewAccountVersionMapping(context.TODO(), client, namespace, cmName, logrus.New())

		// when
		version, origin, found, err := svc.Get(fixGlobalAccountID, fixSubAccountID)
		require.NoError(t, err)

		// then
		assert.False(t, found)
		assert.Empty(t, version)
		assert.Empty(t, origin)
	})
}

//func TestConfigMapGlobalAccountVersionMapping_ForGlobalAccount(t *testing.T) {
//	// given
//	sch := runtime.NewScheme()
//	require.NoError(t, coreV1.AddToScheme(sch))
//	client := fake.NewFakeClientWithScheme(sch, &coreV1.ConfigMap{
//		ObjectMeta: metaV1.ObjectMeta{
//			Name:      cmName,
//			Namespace: namespace,
//		},
//		Data: map[string]string{
//			globalAccountPrefix+"001": "1.14",
//			subaccountPrefix+"002": "1.15-rc1",
//		},
//	})
//
//	svc := NewAccountVersionMapping(context.TODO(), client, namespace, cmName, logrus.New())
//
//	// when
//	v1, origin1, found1, err := svc.Get("ga-001")
//	require.NoError(t, err)
//
//	v2, origin2, found2, err := svc.Get("ga-002")
//	require.NoError(t, err)
//
//	_, origin3, found3, err := svc.Get("not-existing")
//	require.NoError(t, err)
//
//	// then
//	assert.Equal(t, "1.14", v1)
//	assert.Equal(t, "1.15-rc1", v2)
//	assert.True(t, found1)
//	assert.True(t, found2)
//	assert.False(t, found3)
//}
