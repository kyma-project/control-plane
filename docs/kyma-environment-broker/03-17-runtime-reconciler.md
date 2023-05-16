# Runtime Reconciler

Runtime Reconciler is an application that performs reconciliation tasks on SAP BTP, Kyma runtimes (SKRs).

## Overview

Currently, there is one task for Runtime Reconciler. It reconciles BTP Manager Secrets on Kyma Runtimes. It can do it in two ways: 
- by implementation with the usage of Runtime Watcher, which sends events about changes of the Secret from Kyma Runtime to KEB in real time. 
- with a job, which periodically loops over all instances from the KEB database; each instance has an existing assigned Runtime ID; the job checks if the Secret on the Kyma Runtime matches the credentials from the KEB database.

## Configuration

The application is defined as a Kubernetes deployment.

Use the following environment variables to configure the application:

| Environment variable                                             | Description                                                                                                                      | Default value |
| ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| **RUNTIME_RECONCILER_WATCHER_ENABLED**                           | Specifies whether the application should use Runtime Watcher for reconciliation.                                                                   | `false`        |
| **RUNTIME_RECONCILER_JOB_ENABLED**                               | Specifies whether the application should use the job to reconcile.                                                                       | `false`        |
| **RUNTIME_RECONCILER_DRY_RUN**                                   | Specifies whether to run the application in the dry-run mode.                                                                    | `true`        |
| **RUNTIME_RECONCILER_BTP_MANAGER_SECRET_WATCHER_ADDR**           | Specifies Runtime Watcher's port.                                                                                                       | `0`           |
| **RUNTIME_RECONCILER_BTP_MANAGER_SECRET_WATCHER_COMPONENT_NAME** | Specifies the component name for Runtime Watcher.                                                                                               | `NA`          |
| **RUNTIME_RECONCILER_AUTO_RECONCILE_INTERVAL**                   | Specifies at what intervals the job runs  (in hours).                                                                       | `24`          |
| **RUNTIME_RECONCILER_DATABASE_SECRET_KEY**                       | Specifies the secret key for the database.                                                                                       | optional      |
| **RUNTIME_RECONCILER_DATABASE_USER**                             | Specifies the username for the database.                                                                                         | `postgres`    |
| **RUNTIME_RECONCILER_DATABASE_PASSWORD**                         | Specifies the user password for the database.                                                                                    | `password`    |
| **RUNTIME_RECONCILER_DATABASE_HOST**                             | Specifies the host of the database.                                                                                              | `localhost`   |
| **RUNTIME_RECONCILER_DATABASE_PORT**                             | Specifies the port for the database.                                                                                             | `5432`        |
| **RUNTIME_RECONCILER_DATABASE_NAME**                             | Specifies the name of the database.                                                                                              | `broker`      |
| **RUNTIME_RECONCILER_DATABASE_SSLMODE**                          | Activates the SSL mode for PostgreSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html).       | `disable`     |
| **RUNTIME_RECONCILER_DATABASE_SSLROOTCERT**                      | Specifies the location of CA cert of PostgreSQL. (Optional)                                                                      |  optional     |
| **RUNTIME_RECONCILER_PROVISIONER_URL**                           | Specifies URL for intergration with Provisioner.                                                                                 |   -           |