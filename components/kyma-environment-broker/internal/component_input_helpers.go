package internal

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	SCMigrationComponentName          = "sc-migration"
	BTPOperatorComponentName          = "btp-operator"
	HelmBrokerComponentName           = "helm-broker"
	ServiceCatalogComponentName       = "service-catalog"
	ServiceCatalogAddonsComponentName = "service-catalog-addons"
	ServiceManagerComponentName       = "service-manager-proxy"
)

func DisableServiceManagementComponents(r ProvisionerInputCreator) {
	r.DisableOptionalComponent(SCMigrationComponentName)
	r.DisableOptionalComponent(HelmBrokerComponentName)
	r.DisableOptionalComponent(ServiceCatalogComponentName)
	r.DisableOptionalComponent(ServiceCatalogAddonsComponentName)
	r.DisableOptionalComponent(ServiceManagerComponentName)
	r.DisableOptionalComponent(BTPOperatorComponentName)
}

func CreateBTPOperatorProvisionInput(r ProvisionerInputCreator, creds *ServiceManagerOperatorCredentials) {
	overrides := []*gqlschema.ConfigEntryInput{
		{
			Key:    "manager.secret.clientid",
			Value:  creds.ClientID,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "manager.secret.clientsecret",
			Value:  creds.ClientSecret,
			Secret: ptr.Bool(true),
		},
		{
			Key:   "manager.secret.url",
			Value: creds.ServiceManagerURL,
		},
		{
			Key:   "manager.secret.tokenurl",
			Value: creds.URL,
		},
	}
	r.AppendOverrides(BTPOperatorComponentName, overrides)
}

func CreateBTPOperatorUpdateInput(r ProvisionerInputCreator, creds *ServiceManagerOperatorCredentials) error {
	// TODO: get this from
	// https://github.com/kyma-project/kyma/blob/dba460de8273659cd8cd431d2737015a1d1909e5/tests/fast-integration/skr-svcat-migration-test/test-helpers.js#L39-L42
	overrides := []*gqlschema.ConfigEntryInput{
		{
			Key:   "cluster.id",
			Value: "",
		},
	}
	r.AppendOverrides(BTPOperatorComponentName, overrides)
	return nil
}
