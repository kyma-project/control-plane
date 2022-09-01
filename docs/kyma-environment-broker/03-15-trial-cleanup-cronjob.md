# Trial Cleanup Job

Trial Cleanup Job is a Job that makes the SKR instances with the `trial` plan expire 14 days after their creation.
Expiration means that the SKR instance is suspended and the `expired` flag is set.

## Details

For each instance meeting the criteria, a PATCH request is sent to Kyma Environment Broker (KEB). This instance is marked as `expired`, and if it is in the `succeeded` state, the suspension process is started. 
If the instance is already in the `suspended` state, this instance is just marked as `expired`. 

### Dry-run mode
If you need to test the Job, you can run it in the `dry-run` mode.
In that mode, the Job only logs the information about the candidate instances (i.e. instances meeting the configured criteria). The instances are not affected.

## Prerequisites

The Trial Cleanup Job requires access to:
- KEB database, to get the IDs of the instances with `trial` plan which are not expired yet. 
- KEB, to initiate the SKR instance suspension.

## Configuration

The Job is a CronJob with a schedule that can be [configured](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax) as a parameter in the `management-plane-config` repository.
By default, the CronJob is set to run every day at 1:15 am:
```yaml  
kyma-environment-broker.trialCleanup.schedule: "15 1 * * *"
```

Use the following environment variables to configure the Job:

| Environment variable | Description                                                                                                               | Default value                            |
|---|---------------------------------------------------------------------------------------------------------------------------|------------------------------------------|
| **APP_DRY_RUN** | Specifies whether to run the Job in the [`dry-run` mode](#details).                                                       | `true`                                   |
| **APP_EXPIRATION_PERIOD** | Specifies the [expiration period](#trial-cleanup-job) for the instances with the `trial` plan.                            | `336h`                                    |
| **APP_DATABASE_USER** | Specifies the username for the database.                                                                                  | `postgres`                               |
| **APP_DATABASE_PASSWORD** | Specifies the user password for the database.                                                                             | `password`                               |
| **APP_DATABASE_HOST** | Specifies the host of the database.                                                                                       | `localhost`                              |
| **APP_DATABASE_PORT** | Specifies the port for the database.                                                                                      | `5432`                                   |
| **APP_DATABASE_NAME** | Specifies the name of the database.                                                                                       | `provisioner`                            |
| **APP_DATABASE_SSL** | Activates the SSL mode for PostgreSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html). | `disable`                                |
| **APP_BROKER_URL**  | Specifies the KEB URL.                                                                                                    | `https://kyma-env-broker.kyma.local`     |
| **APP_BROKER_TOKEN_URL** | Specifies the KEB OAuth token endpoint.                                                                                   | `https://oauth.2kyma.local/oauth2/token` |
| **APP_BROKER_CLIENT_ID** | Specifies the username for the OAuth2 authentication in KEB.                                                              | None                                     |
| **APP_BROKER_CLIENT_SECRET** | Specifies the password for the OAuth2 authentication in KEB.                                                              | None                                     |
| **APP_BROKER_SCOPE** | Specifies the scope for the OAuth2 authentication in KEB.                                                                 | None                                     |
