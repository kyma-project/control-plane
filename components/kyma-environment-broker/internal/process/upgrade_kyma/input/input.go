package input

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/vburenin/nsync"
)

type UpgradeKymaInput struct {
	input gqlschema.UpgradeRuntimeInput
	mutex *nsync.NamedMutex

	desiredKymaVersion string
}

func (u *UpgradeKymaInput) Create() (gqlschema.UpgradeRuntimeInput, error) {
	return u.input, nil
}

func (u *UpgradeKymaInput) SetDesiredKymaVersion(kymaVersion string) internal.UpgradeKymaInputCreator {
	u.input.KymaConfig.Version = kymaVersion

	return u
}
