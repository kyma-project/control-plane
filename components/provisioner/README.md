# Provisioner

## Overview

Runtime Provisioner is a Kyma Control Plane component responsible for provisioning, installing, and deprovisioning clusters. When provisioning a cluster, you have an option to provision a cluster with Kyma (Kyma Runtime), or without it. To provision a Kyma Runtime, provide its configuration as **kymaConfig**.

For more details, see the Runtime Provisioner [documentation](https://github.com/kyma-project/control-plane/tree/main/docs/provisioner).

## Prerequisites

Before you can run Runtime Provisioner, you have to configure access to the PostgreSQL database. For development purposes, you can run a PostgreSQL instance in the Docker container executing the following command:

```bash
$ docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=password postgres
```

Afterwards, the database must be created, and migrations must be run.

Runtime Provisioner also needs a kubeconfig for a garden and auth data for Director.

## Development

### GraphQL schema

After you introduce changes in the GraphQL schema, run the `gqlgen.sh` script.

### Database schema

For tests to run properly, update the database schema in `./assets/database/provisioner.sql`. Provide the new migration in the Schema Migrator component in `resources/kcp/charts/provisioner/migrations`.

### Run Provisioner

To run Runtime Provisioner, use the following command:
```bash
go run ./cmd/
```

### Environment Variables

This table lists the environment variables, their descriptions, and default values:


| Parameter                                                     | Description                                                                                               | Default value                                                           |
|:--------------------------------------------------------------|:----------------------------------------------------------------------------------------------------------|:------------------------------------------------------------------------|
| APP_ADDRESS                                                   | Runtime Provisioner's address with the port                                                               | `127.0.0.1:3000`                                                        |
| APP_API_ENDPOINT                                              | Endpoint for the GraphQL API                                                                              | `/graphql`                                                              |
| APP_DATABASE_NAME                                             | Database name                                                                                             | `provisioner`                                                           |
| APP_DATABASE_PASSWORD                                         | Database user password                                                                                    | `password`                                                              |
| APP_DATABASE_PORT                                             | Database port                                                                                             | `5432`                                                                  |
| APP_DATABASE_SECRET_KEY                                       |                                                                                                           | optional                                                                |
| APP_DATABASE_SSL_MODE                                         | SSL Mode for PostgrSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html) | `disable`                                                               |
| APP_DATABASE_SSL_ROOT_CERT                                    |                                                                                                           | optional                                                                |
| APP_DATABASE_USER                                             | Database username                                                                                         | `postgres`                                                              |
| APP_DATABSE_HOST                                              | Database host                                                                                             | `localhost`                                                             |
| APP_DEPROVISIONING_NO_INSTALL_TIMEOUT                         |                                                                                                           |                                                                         |
| APP_DEPROVISIONING_TIMEOUT                                    |                                                                                                           |                                                                         |
| APP_DIRECTOR_OAUTH_PATH                                       | Path to a YAML file with Director's OAUTH data. Format described below                                    | `./dev/director.yaml`                                                   |
| APP_DIRECTOR_URL                                              | Director URL                                                                                              | `http://compass-director.compass-system.svc.cluster.local:3000/graphql` |
| APP_DOWNLOAD_PRE_RELEASES                                     |                                                                                                           | `true`                                                                  |
| APP_ENQUEUE_IN_PROGRESS_OPERATIONS                            | Specifies whether operations in the `InProgress` state should be enqueued on the application startup      | `true`                                                                  |
| APP_GARDENER_AUDIT_LOGS_POLICY_CONFIG_MAP                     | Name of the ConfigMap containing the audit logs policy                                                    | optional                                                                |
| APP_GARDENER_AUDIT_LOGS_TENANT_CONFIG_PATH                    |                                                                                                           | optional                                                                |
| APP_GARDENER_CLUSTER_CLEANUP_RESOURCE_SELECTOR                |                                                                                                           | `https://service-manager.`                                              |
| APP_GARDENER_DEFAULT_ENABLE_KUBERNETES_VERSION_AUTO_UPDATE    |                                                                                                           | `false`                                                                 |
| APP_GARDENER_DEFAULT_ENABLE_MACHINE_IMAGE_VERSION_AUTO_UPDATE |                                                                                                           | `false`                                                                 |
| APP_GARDENER_KUBECONFIG_PATH                                  | Filepath for the Gardener kubeconfig                                                                      | `./dev/kubeconfig.yaml`                                                 |
| APP_GARDENER_MAINTENANCE_WINDOW_CONFIG_PATH                   |                                                                                                           | optional                                                                |
| APP_GARDENER_PROJECT                                          | Name of the Gardener project connected to the service account                                             | `gardenerProject`                                                       |
| APP_HIBERNATION_TIMEOUT                                       |                                                                                                           |                                                                         |
| APP_LATEST_DOWNLOADED_RELEASES                                |                                                                                                           | `5`                                                                     |
| APP_LOG_LEVEL                                                 |                                                                                                           | `info`                                                                  |
| APP_METRICS_ADDRESS                                           | Runtime Provisioner Metrics' address with the port                                                        | `127.0.0.1:9000`                                                        |
| APP_OPERATOR_ROLE_BINDING                                     |                                                                                                           |                                                                         |
| APP_PLAYGROUND_API_ENDPOINT                                   | Endpoint for the API playground                                                                           | `/graphql`                                                              |
| APP_PROVISIONING_NO_INSTALL_TIMEOUT                           |                                                                                                           |                                                                         |
| APP_PROVISIONING_TIMEOUT                                      |                                                                                                           |                                                                         |
| APP_RUN_AWS_CONFIG_MIGRATION                                  | TODO: Remove after data migration                                                                         | `false`                                                                 |
| APP_SKIP_DIRECTOR_CERT_VERIFICATION                           | Flag to skip certificate verification for Director                                                        | `false`                                                                 |

Director OAUTH config should look like this:
```yaml
data:
  client_id: <client id>
  client_secret: <client secret>
  tokens_endpoint: https://example.com/oauth2/token
```
