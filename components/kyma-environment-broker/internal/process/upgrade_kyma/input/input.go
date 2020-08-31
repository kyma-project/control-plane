package input

import "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

type UpgradeKymaInput struct {
	input gqlschema.UpgradeRuntimeInput
}

func (u *UpgradeKymaInput) Create() (gqlschema.UpgradeRuntimeInput, error) {
	updateString(&u.input.KymaConfig.Version, )
	return u.input, nil
}

func updateString(toUpdate *string, value *string) {
	if value != nil {
		*toUpdate = *value
	}
}