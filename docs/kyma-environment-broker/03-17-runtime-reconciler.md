# Runtime reconciler

Runtime reconciler is application which perform some reconcile tasks on runtimes (SKRs).

## Overview

Currently there is one task for runtime reconciler, it is to reconcile BTP Manager secrets on SKRs. It is achived in two ways, first one is implemention with usage of runtime-watcher, which in real time sends event about changes of secret from SKR to KEB. The second method is a job, which periodically loop over all instances from KEB database which have existing assigned runtimes ID, and for each of them, do a check if secret on SKR match with credentials from KEB database.

## Configuration

The application is defined as kubernetes deployment.

Use the following environment variables to configure the application:

| Environment variable                                             | Description                                                                                                                      | Default value |
| ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| **RUNTIME_RECONCILER_WATCHER_ENABLED**                           | Specifies whether application should use watcher to reconcile.                                                                   | `false`        |
| **RUNTIME_RECONCILER_JOB_ENABLED**                               | Specifies whether application should use job to reconcile.                                                                       | `false`        |
| **RUNTIME_RECONCILER_DRY_RUN**                                   | Specifies whether to run the application in the dry-run mode.                                                                    | `true`        |
| **RUNTIME_RECONCILER_BTP_MANAGER_SECRET_WATCHER_ADDR**           | Specifies port of watcher.                                                                                                       | `0`           |
| **RUNTIME_RECONCILER_BTP_MANAGER_SECRET_WATCHER_COMPONENT_NAME** | Specifies company name of watcher.                                                                                               | `NA`          |
| **RUNTIME_RECONCILER_AUTO_RECONCILE_INTERVAL**                   | Specifies in what interval (in hours), the job should run.                                                                       | `24`          |
| **RUNTIME_RECONCILER_DATABASE_SECRET_KEY**                       | Specifies the secret key for the database.                                                                                       | optional      |
| **RUNTIME_RECONCILER_DATABASE_USER**                             | Specifies the username for the database.                                                                                         | `postgres`    |
| **RUNTIME_RECONCILER_DATABASE_PASSWORD**                         | Specifies the user password for the database.                                                                                    | `password`    |
| **RUNTIME_RECONCILER_DATABASE_HOST**                             | Specifies the host of the database.                                                                                              | `localhost`   |
| **RUNTIME_RECONCILER_DATABASE_PORT**                             | Specifies the port for the database.                                                                                             | `5432`        |
| **RUNTIME_RECONCILER_DATABASE_NAME**                             | Specifies the name of the database.                                                                                              | `broker`      |
| **RUNTIME_RECONCILER_DATABASE_SSLMODE**                          | Activates the SSL mode for PostgreSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html).       | `disable`     |
| **RUNTIME_RECONCILER_DATABASE_SSLROOTCERT**                      | Specifies the location of CA cert of PostgreSQL. (Optional)                                                                      |  optional     |
| **RUNTIME_RECONCILER_PROVISIONER_URL**                           | Specifies URL for intergration with Provisioner.                                                                                 |   -           |