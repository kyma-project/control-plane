# Kyma Environment Broker

## Overview

Kyma Environment Broker (KEB) is a component that allows you to provision Kyma as a Runtime on clusters provided by third-party providers. It uses the Runtime Provisioner's API to install Kyma on a given cluster.

For more information, read the [documentation](../../docs/kyma-environment-broker) where you can find information on:

- [Architecture](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/02-01-architecture.md)
- [Service description](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/03-01-service-description.md)
- [Runtime components](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/03-02-runtime-components.md)
- [Runtime provisioning and deprovisioning](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/03-03-runtime-provisioning-and-deprovisioning.md)
- [Hyperscaler account pool](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/03-04-hyperscaler-account-pool.md)
- [Authorization](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/03-05-authorization.md)
- [Runtime overrides](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/03-06-runtime-overrides.md)
- [Provisioning Kyma environment](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/08-01-provisioning-kyma-environment.md)
- [Deprovisioning Kyma environment](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/08-02-deprovisioning-kyma-environment.md)
- [Operation status](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/08-03-operation-status.md)
- [Instance details](https://github.com/kyma-project/control-plane/blob/master/docs/kyma-environment-broker/08-04-instance-details.md)

## Configuration

KEB binary allows you to override some configuration parameters. You can specify the following environment variables:

| Name | Description | Default value |
|-----|---------|:--------:|
| **APP_PORT** | Specifies the port on which the HTTP server listens. | `8080` |
| **APP_PROVISIONING_DEFAULT_GARDENER_SHOOT_PURPOSE** | Specifies the purpose of the created cluster. The possible values are: `development`, `evaluation`, `production`, `testing`. | `development` |
| **APP_PROVISIONING_URL** | Specifies a URL to the Runtime Provisioner's API. | None |
| **APP_PROVISIONING_SECRET_NAME** | Specifies the name of the Secret which holds credentials to the Runtime Provisioner's API. | None |
| **APP_PROVISIONING_GARDENER_PROJECT_NAME** | Defines the Gardener project name. | `true` |
| **APP_PROVISIONING_GCP_SECRET_NAME** | Defines the name of the Secret which holds credentials to GCP. | None |
| **APP_PROVISIONING_AWS_SECRET_NAME** | Defines the name of the Secret which holds credentials to AWS. | None |
| **APP_PROVISIONING_AZURE_SECRET_NAME** | Defines the name of the Secret which holds credentials to Azure. | None |
| **APP_AUTH_USERNAME** | Specifies the Kyma Environment Service Broker authentication username. | None |
| **APP_AUTH_PASSWORD** | Specifies the Kyma Environment Service Broker authentication password. | None |
| **APP_DIRECTOR_URL** | Specifies the Director's URL. | `http://compass-director.compass-system.svc.cluster.local:3000/graphql` |
| **APP_DIRECTOR_OAUTH_TOKEN_URL** | Specifies the URL for OAuth authentication. | None |
| **APP_DIRECTOR_OAUTH_CLIENT_ID** | Specifies the client ID for OAuth authentication. | None |
| **APP_DIRECTOR_OAUTH_SECRET** | Specifies the client secret for OAuth authentication. | None |
| **APP_DIRECTOR_OAUTH_SCOPE** | Specifies the scopes for OAuth authentication. | `runtime:read runtime:write` |
| **APP_DATABASE_USER** | Defines the database username. | `postgres` |
| **APP_DATABASE_PASSWORD** | Defines the database user password. | `password` |
| **APP_DATABASE_HOST** | Defines the database host. | `localhost` |
| **APP_DATABASE_PORT** | Defines the database port. | `5432` |
| **APP_DATABASE_NAME** | Defines the database name. | `broker` |
| **APP_DATABASE_SSL** | Specifies the SSL Mode for PostgrSQL. See all the possible values [here](https://www.postgresql.org/docs/9.1/libpq-ssl.html).  | `disable`|
| **APP_KYMA_VERSION** | Specifies the default Kyma version. | None |
| **APP_ENABLE_ON_DEMAND_VERSION** | If set to `true`, a user can specify a Kyma version in a provisioning request. | `false` |
| **APP_VERSION_CONFIG_NAMESPACE** | Defines the Namespace with the ConfigMap that contains Kyma versions for global accounts configuration. | None |
| **APP_VERSION_CONFIG_NAME** | Defines the name of the ConfigMap that contains Kyma versions for global accounts configuration. | None |
| **APP_PROVISIONING_MACHINE_IMAGE** | Defines the Gardener machine image used in a provisioned node. | None |
| **APP_PROVISIONING_MACHINE_IMAGE_VERSION** | Defines the Gardener image version used in a provisioned cluster. | None |
| **APP_TRIAL_REGION_MAPPING_FILE_PATH** | Defines a path to the file which contains a mapping between the platform region and the Trial plan region. | None |
| **APP_MAX_PAGINATION_PAGE** | Defines the maximum number of objects that can be queried in one page using the endpoints that use pagination. | `100` |
| **APP_LMS_URL** | Defines the URL for the LMS system. | None |
| **APP_LMS_CLUSTER_TYPE** | Defines the cluster type for the LMS system. | `single-node` |
| **APP_LMS_ENVIRONMENT** | Specifies the environment for the LMS system. | `dev` |
| **APP_LMS_SAML_TENANT** | Defines the SAML tenant for the LMS system. | None |
| **APP_LMS_ENABLED_FOR_GLOBAL_ACCOUNTS** | An LMS instance gets provisioned for the specified Global Accounts. Possible values are `all`, `none`, `{global-account-ID-1}, {global-account-ID-2}, ...` | `all` |
| **APP_LMS_MANDATORY** | Defines whether failing LMS activation will break provisioning. | `true` |
| **APP_LMS_REGION** | Defines the region for the LMS system. If set, this region is always used. If empty, the region is mapped from the OSB API request. | None |
| **APP_LMS_TOKEN** | Specifies the token for the LMS system. | None |
