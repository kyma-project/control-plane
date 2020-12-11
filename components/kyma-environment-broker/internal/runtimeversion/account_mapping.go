package runtimeversion

import (
	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AccountVersionMapping struct {
	ctx       context.Context
	k8sClient client.Client

	namespace string
	name      string

	log logrus.FieldLogger
}

const (
	globalAccountPrefix = "GA_"
	subaccountPrefix    = "SA_"
)

func NewAccountVersionMapping(ctx context.Context, cli client.Client,
	namespace, name string, log logrus.FieldLogger) *AccountVersionMapping {

	return &AccountVersionMapping{
		ctx:       ctx,
		namespace: namespace,
		name:      name,
		k8sClient: cli,
		log:       log,
	}
}

// Get retrieves Kyma version from ConfigMap for given accounts IDs
func (m *AccountVersionMapping) Get(globalAccountID, subaccountID string) (string, internal.RuntimeVersionOrigin, bool, error) {
	config := &v1.ConfigMap{}
	key := client.ObjectKey{Namespace: m.namespace, Name: m.name}
	err := m.k8sClient.Get(m.ctx, key, config)

	switch {
	case apierr.IsNotFound(err):
		m.log.Infof("Kyma Version per Account configuration %s/%s not found", m.namespace, m.name)
		return "", "", false, nil
	case err != nil:
		return "", "", false, errors.Wrap(err, "while getting kyma version config map")
	}

	// SubAccount version mapping has higher priority than GlobalAccount version
	ver, found := config.Data[subaccountPrefix+subaccountID]
	if !found {
		ver, found = config.Data[globalAccountPrefix+globalAccountID]
		if !found {
			return ver, "", found, nil
		}
		return ver, internal.GlobalAccount, found, nil
	}

	return ver, internal.SubAccount, found, nil
}
