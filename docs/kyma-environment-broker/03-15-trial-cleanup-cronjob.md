# Trial Cleanup Job

Trial Cleanup Job is an application that makes SKR instances with trial plan expire if those are 14 days old.
Expiration means that SKR instance is suspended and `expired` flag is set.

## Details

The job can be run either in dry run mode or in production mode:

- In `dry run mode` information about candidate instances (i.e. instances meeting configured criteria) is logged. No changes are made.
- In `production mode` for each instance meeting criteria PATCH request is sent to KEB. This instance is marked as `expired` and if is in `succeeded` state suspension process is started. 
In case the instance is already in `suspended` state this instance is only marked as `expired`. 

## Prerequisites

Trial Cleanup requires access to:
- Database to get the IDs of instances with `trial plan` which are not expired yet. 
- Kyma Environment Broker to initiate SKR instance suspension.

## Configuration

The job is a cronjob with schedule that can be configured in `management-plan-config`. 
Default settings (every day at 1:15 am) is as follows:
```  kyma-environment-broker.trialCleanup.schedule: "15 1 * * *"
```

Use the following environment variables to configure the application:

| Environment variable | Description                                                                                                                   | Default value                            |
|---|-------------------------------------------------------------------------------------------------------------------------------|------------------------------------------|
| **APP_DRY_RUN** | Specifies the mode - dry run only logs candidate instances                                                                    | `true`                                   |
| **APP_EXPIRATION_PERIOD** | Specifies expiration period - instances with `trial plan` and older than expiration period are made expired                   | `336h`                                    |
| **APP_DATABASE_USER** | Specifies the username for the database.                                                                                      | `postgres`                               |
| **APP_DATABASE_PASSWORD** | Specifies the user password for the database.                                                                                 | `password`                               |
| **APP_DATABASE_HOST** | Specifies the host of the database.                                                                                           | `localhost`                              |
| **APP_DATABASE_PORT** | Specifies the port for the database.                                                                                          | `5432`                                   |
| **APP_DATABASE_NAME** | Specifies the name of the database.                                                                                           | `provisioner`                            |
| **APP_DATABASE_SSL** | Activates the SSL mode for PostgrSQL. See all the possible values [here](https://www.postgresql.org/docs/9.1/libpq-ssl.html). | `disable`                                |
| **APP_BROKER_URL**  | Specifies the Kyma Environment Broker URL.                                                                                    | `https://kyma-env-broker.kyma.local`     |
| **APP_BROKER_TOKEN_URL** | Specifies the Kyma Environment Broker OAuth token endpoint.                                                                   | `https://oauth.2kyma.local/oauth2/token` |
| **APP_BROKER_CLIENT_ID** | Specifies the username for the OAuth2 authentication in KEB.                                                                  | None                                     |
| **APP_BROKER_CLIENT_SECRET** | Specifies the password for the OAuth2 authentication in KEB.                                                                  | None                                     |
| **APP_BROKER_SCOPE** | Specifies the scope for the OAuth2 authentication in KEB.                                                                     | None                                     |
