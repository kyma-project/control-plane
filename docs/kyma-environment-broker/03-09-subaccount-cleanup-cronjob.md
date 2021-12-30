# Subaccount Cleanup CronJob


Each SKR instance in Kyma Environment Broker (KEB) database belongs to a global account and to a subaccount.
Subaccount Cleanup is an application that periodically calls the CIS service and notifies about `SUBACCOUNT_DELETE` events.
Based on these events, Subaccount Cleanup triggers the deprovisioning action on the SKR instance to which a given subaccount belongs.

## Details

The Subaccount Cleanup workflow is divided into several steps:

1. Fetch `SUBACCOUNT_DELETE` events from the CIS service.

    a. CIS client makes a call to the CIS service and as a response, it gets a list of events divided into pages.

    b. CIS client fetches the rest of the events by making a call to each page one by one.

    c. A subaccount ID is taken from each event and kept in an array.

    d. When the CIS client ends its workflow, it displays logs with information on how many subaccounts were fetched.

2. Find all instances in the KEB database based on the fetched subaccount IDs.
   The subaccounts pool is divided into pieces. For each piece, a query is made to the database to fetch instances.

3. Trigger the deprovisioning operation for each instance found in step 2.

   Logs inform about the status of each triggered action:
    ```
    deprovisioning for instance <InstanceID> (SubAccountID: <SubAccountID>) was triggered, operation: <OperationID>
    ```
   Subaccount Cleanup also uses logs to inform about the end of the deprovisioning operation.

## Prerequisites

Subaccount Cleanup requires access to:
- CIS service to receive all `SUBACCOUNT_DELETE` events
- Database to get the instance ID for each subaccount ID from the `SUBACCOUNT_DELETE` event
- Kyma Environment Broker to trigger SKR instance deprovisioning

## Configuration

Use the following environment variables to configure the application:

| Environment variable | Description |
|---|---|
| **APP_CLIENT_VERSION** | Specifies the service version from which events are fetched. The possible values are  `v1.0` or `v2.0`.
| **APP_CIS_CLIENT_ID** | Specifies the client ID for the OAuth2 authentication in CIS.
| **APP_CIS_CLIENT_SECRET** | Specifies the client secret for the OAuth2 authentication in CIS.
| **APP_CIS_AUTH_URL** | Specifies the endpoint for the CIS OAuth token.
| **APP_CIS_EVENT_SERVICE_URL** | Specifies the endpoint for CIS events.
| **APP_DATABASE_USER** | Specifies the username for the database.
| **APP_DATABASE_PASSWORD** | Specifies the user password for the database.
| **APP_DATABASE_HOST** | Specifies the host of the database.
| **APP_DATABASE_PORT** | Specifies the port for the database.
| **APP_DATABASE_NAME** | Specifies the name of the database.
| **APP_DATABASE_SSL_MODE** | Activates the SSL mode for PostgreSQL. For reference, see the list of [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html).  
| **APP_BROKER_URL**  | Specifies the Kyma Environment Broker URL. |
| **APP_BROKER_TOKEN_URL** | Specifies the endpoint for Kyma Environment Broker OAuth token. |
| **APP_BROKER_CLIENT_ID** | Specifies the username for the OAuth2 authentication in KEB. |
| **APP_BROKER_CLIENT_SECRET** | Specifies the password for the OAuth2 authentication in KEB. |
| **APP_BROKER_SCOPE** | Specifies the scope of the OAuth2 authentication in KEB. |
