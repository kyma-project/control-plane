package avs

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

type ModelConfigurator interface {
	ProvideSuffix() string
	ProvideTesterAccessId(pp internal.ProvisioningParameters) int64
	ProvideGroupId(pp internal.ProvisioningParameters) int64
	ProvideParentId(pp internal.ProvisioningParameters) int64
	ProvideTags() []*Tag
	ProvideNewOrDefaultServiceName(defaultServiceName string) string
	ProvideCheckType() string
}
