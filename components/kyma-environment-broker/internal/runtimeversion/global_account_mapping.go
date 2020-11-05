package runtimeversion

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GlobalAccountVersionMapping struct {
	ctx       context.Context
	k8sClient client.Client

	namespace string
	name      string

	log logrus.FieldLogger
}

func NewGlobalAccountVersionMapping(ctx context.Context, cli client.Client,
	namespace, name string, log logrus.FieldLogger) *GlobalAccountVersionMapping {

	return &GlobalAccountVersionMapping{
		ctx:       ctx,
		namespace: namespace,
		name:      name,
		k8sClient: cli,
		log:       log,
	}
}

func (m *GlobalAccountVersionMapping) Get(globalAccountID string) (string, bool, error) {
	config := &v1.ConfigMap{}
	key := client.ObjectKey{Namespace: m.namespace, Name: m.name}
	err := m.k8sClient.Get(m.ctx, key, config)

	switch {
	case apierr.IsNotFound(err):
		m.log.Infof("Kyma Version per Global Acocunt configuration %s/%s not found", m.namespace, m.name)
		return "", false, nil
	case err != nil:
		return "", false, errors.Wrap(err, "while getting kyma version config map")
	}

	ver, found := config.Data[globalAccountID]
	return ver, found, nil
}
