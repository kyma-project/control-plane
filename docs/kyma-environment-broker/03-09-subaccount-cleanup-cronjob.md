---
title: Subaccount Cleanup CronJob
type: Details
---

Each SKR instance in KEB database belongs to global account and subaccount.
Subaccount Cleanup is an application which periodically calls CIS service about SUBACCOUNT_DELETE events 
and based on those events trigger de-provision action on SKR instance which subaccount belongs to.

## Details

All Subaccount Cleanup process is divided into several steps

1. Fetch SUBACCOUNT_DELETE events from CIS service

    a. CIS client makes first call to CIS service, as a response it gets list of events  divided into pages, 
       total number of events and total number of pages
    b. CIS client fetches rest of the events by make call to each page one by one
    c. a subaccount is taken from each event and keep in array
    d. when CIS client ends, display logs with information how many subaccounts fetched

2. Find all instances in KEB database based on subaccounts fetched in step 1

    a. the whole subaccounts pool is divided into pieces
    b. for each pieces query to database is made simultaneously to fetch instances

3. Trigger de-provisioning operation for each found instance in step 2

    a. for each instance found, de-provision action is triggered on broker client
       logs informs for each triggered action:

       ```deprovisioning for instance <InstanceID> (SubAccountID: <SubAccountID>) was triggered, operation: <OperationID>```
    b. at the end, Subaccount Cleanup informs about the end of his work through the appropriate log

## Prerequisites

Subaccount Cleanup requires access to:
- CIS service to receive all SUBACCOUNT_DELETE events
- Database to get an Instance ID for each Subaccount from SUBACCOUNT_DELETE event
- Kyma Environment Broker to trigger SKR instance deprovisioning

## Configuration

| Environment variable                       | Description                                                                                                                        
|--------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------
| **APP_CLIENT_VERSION** | Specifies from which version of the service events will be fetched: v1.0 or v2.0
| **APP_CIS_CLIENT_ID** | Specifies the client id for the OAuth2 authentication in CIS.
| **APP_CIS_CLIENT_SECRET** | Specifies the client secret for the OAuth2 authentication in CIS.
| **APP_CIS_AUTH_URL** | Specifies the CIS OAuth token endpoint.
| **APP_CIS_EVENT_SERVICE_URL** | Specifies the CIS events endpoint.
| **APP_DATABASE_USER** | Specifies the username for the database. 
| **APP_DATABASE_PASSWORD** | Specifies the user password for the database. 
| **APP_DATABASE_HOST** | Specifies the host of the database. 
| **APP_DATABASE_PORT** | Specifies the port for the database. 
| **APP_DATABASE_NAME** | Specifies the name of the database. 
| **APP_DATABASE_SSL_MODE** | Activates the SSL mode for PostgrSQL. See all the possible values [here](https://www.postgresql.org/docs/9.1/libpq-ssl.html).  
| **APP_BROKER_URL**                             | Specifies the Kyma Environment Broker URL.                                                                                         
| **APP_BROKER_TOKEN_URL**                       | Specifies the Kyma Environment Broker OAuth token endpoint.                                                                        
| **APP_BROKER_CLIENT_ID**                       | Specifies the username for the OAuth2 authentication in KEB.                                                                       
| **APP_BROKER_CLIENT_SECRET**                   | Specifies the password for the OAuth2 authentication in KEB.                                                                       
| **APP_BROKER_SCOPE**                           | Specifies the scope for the OAuth2 authentication in KEB.
                                                                          