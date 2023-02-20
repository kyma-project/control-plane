# Deprovision Retrigger Job

Deprovision Retrigger Job is a Job that attempts to deprovision an instance once again.

## Overview

During regular deprovisioning, you can omit some steps due to the occurrence of some errors. These errors do not cause the deprovisioning process to fail.
You can ignore some not-severe, temporary errors, proceed with deprovisioning and declare the process successful.  The not-completed steps
can be retried later. Store the list of not-completed steps, and mark the deprovisioning operation by setting `deletedAt` to the current timestamp.
The Job iterates over the instances, and for each one with `deletedAt` appropriately set, sends a DELETE to Kyma Environment Broker (KEB).  

## Prerequisites

Deprovision Retrigger Job requires access to:
- KEB database, to get the IDs of the instances with not completed steps.
- KEB, to request SKR deprovisioning.

## Configuration

The Job is a CronJob with a schedule that can be [configured](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax) as a parameter in the `management-plane-config` repository.
By default, the CronJob is set to run every day at 3:00 am:
```yaml  
kyma-environment-broker.trialCleanup.schedule: "0 3 * * *"
```

> **NOTE**
> If you need to test the Job, you can run it in the dry-run mode.
> In this mode, the Job only logs the information about the candidate instances (i.e. instances meeting the configured criteria). The instances are not affected.


Use the following environment variables to configure the Job:

| Environment variable | Description                                                                                                             | Default value                            |
|---|-------------------------------------------------------------------------------------------------------------------------|------------------------------------------|
| **APP_DRY_RUN** | Specifies whether to run the Job in the [dry-run mode](#details).                                                       | `true`                                   |
| **APP_DATABASE_USER** | Specifies the username for the database.                                                                                | `postgres`                               |
| **APP_DATABASE_PASSWORD** | Specifies the user password for the database.                                                                           | `password`                               |
| **APP_DATABASE_HOST** | Specifies the host of the database.                                                                                     | `localhost`                              |
| **APP_DATABASE_PORT** | Specifies the port for the database.                                                                                    | `5432`                                   |
| **APP_DATABASE_NAME** | Specifies the name of the database.                                                                                     | `provisioner`                            |
| **APP_DATABASE_SSLMODE** | Activates the SSL mode for PostgreSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html). | `disable`                                |
| **APP_DATABASE_SSLROOTCERT** | Specifies the location of CA cert of PostgreSQL. (Optional)                                          | None                                |
| **APP_BROKER_URL**  | Specifies the KEB URL.                                                                                                  | `https://kyma-env-broker.kyma.local`     |
| **APP_BROKER_TOKEN_URL** | Specifies the KEB OAuth token endpoint.                                                                                 | `https://oauth.2kyma.local/oauth2/token` |
| **APP_BROKER_CLIENT_ID** | Specifies the username for the OAuth2 authentication in KEB.                                                            | None                                     |
| **APP_BROKER_CLIENT_SECRET** | Specifies the password for the OAuth2 authentication in KEB.                                                            | None                                     |
| **APP_BROKER_SCOPE** | Specifies the scope for the OAuth2 authentication in KEB.                                                               | None                                     |
