# Provision Kyma Runtime using KEB

This tutorial shows how to provision Kyma Runtime on Azure using Kyma Environment Broker.

## Prerequisites

- Compass with:
  * Runtime Provisioner [configured](../provisioner/08-02-provisioning-gardener.md) for Azure provider
  * Kyma Environment Broker configured and chosen [overrides](https://kyma-project.io/docs/kyma/latest/04-operation-guides/operations/03-change-kyma-config-values/) set up

## Steps

1. Export these values as environment variables:

   ```bash
   export BROKER_URL={KYMA_ENVIRONMENT_BROKER_URL}
   export INSTANCE_ID={INSTANCE_ID}
   export GLOBAL_ACCOUNT_ID={GLOBAL_ACCOUNT_ID}
   export NAME={RUNTIME_NAME}
   export USER_ID={USER_ID}
   ```

   > **NOTE:** INSTANCE_ID and NAME must be unique. It is recommended to use UUID as an INSTANCE_ID.

2. Get the [access token](03-05-authorization.md#get-the-access-token). Export this variable based on the token you got from the OAuth client:

   ```bash
   export AUTHORIZATION_HEADER="Authorization: Bearer $ACCESS_TOKEN"
   ```

3. Make a call to the Kyma Environment Broker to create a Runtime on Azure. Find the list of possible request parameters [here](03-01-service-description.md).

   ```bash
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
           \"user_id\": \"$USER_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\"
       }
   }"
   ```

A successful call returns the operation ID:

   ```json
   {
       "operation":"8a7bfd9b-f2f5-43d1-bb67-177d2434053c"
   }
   ```

4. Check the operation status as described [here](08-03-operation-status.md).

## BTP Operator

If you need a BTP Operator component installed, please obtain [BTP Operator access credentials](https://github.com/SAP/sap-btp-service-operator/blob/v0.2.5/README.md#setup), then provide it in the provisioning request, for example:
 ```bash
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\",
           \"user_id\": \"$USER_ID\",
           \"sm_operator_credentials\": {
             \"clientid\": \"$clientid\",
             \"clientsecret\": \"$clientsecret\",
             \"sm_url\": \"$sm_url\",
             \"url\": \"$url\",
             \"xsappname\": \"$xsappname\"
		   },
       },
       \"parameters\": {
           \"name\": \"$NAME\"
       }
   }"
   ```

```json
"sm_operator_credentials": {
    "clientid": "testClientID",
						  "clientsecret": "testClientSecret",
						  "sm_url": "https://service-manager.kyma.com",
						  "url": "https://test.auth.com",
						  "xsappname": "testXsappname"
						},
```