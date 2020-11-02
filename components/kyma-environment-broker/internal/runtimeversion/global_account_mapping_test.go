package runtimeversion

import (
	"testing"

	"context"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	cmName    = "config"
	namespace = "foo"
)

func TestConfigMapGlobalAccountVersionMapping_ForGlobalAccount(t *testing.T) {
	// given
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

	svc := NewGlobalAccountVersionMapping(context.TODO(), client, namespace, cmName, logrus.New())

	// when
	v1, found1, err := svc.Get("ga-001")
	require.NoError(t, err)

	v2, found2, err := svc.Get("ga-002")
	require.NoError(t, err)

	_, found3, err := svc.Get("not-existing")
	require.NoError(t, err)

	// then
	assert.Equal(t, "1.14", v1)
	assert.Equal(t, "1.15-rc1", v2)
	assert.True(t, found1)
	assert.True(t, found2)
	assert.False(t, found3)
}
