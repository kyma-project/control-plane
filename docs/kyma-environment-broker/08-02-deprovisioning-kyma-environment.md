# Deprovision Kyma Runtime using KEB

This tutorial shows how to deprovision Kyma Runtime on Azure using Kyma Environment Broker.

## Steps

1. Ensure that these environment variables are exported:

   ```bash
   export BROKER_URL={KYMA_ENVIRONMENT_BROKER_URL}
   export INSTANCE_ID={INSTANCE_ID_FROM_PROVISIONING_CALL}
   ```

2. Get the [access token](03-05-authorization.md#get-the-access-token). Export this variable based on the token you got from the OAuth client:

   ```bash
   export AUTHORIZATION_HEADER="Authorization: Bearer $ACCESS_TOKEN"
   ```

3. Make a call to the Kyma Environment Broker to delete a Runtime on Azure.

   ```bash
   curl  --request DELETE "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=4deee563-e5ec-4731-b9b1-53b42d855f0c" \
   --header 'X-Broker-API-Version: 2.13' \
   --header "$AUTHORIZATION_HEADER"
   ```

A successful call returns the operation ID:

   ```json
   {
       "operation":"8a7bfd9b-f2f5-43d1-bb67-177d2434053c"
   }
   ```

4. Check the operation status as described [here](08-03-operation-status.md).

## Subaccount Cleanup Job

The standard workflow for [BTP Operator](https://github.com/SAP/sap-btp-service-operator) resources is to keep them untouched by KEB because users may intend to
keep the external services provisioned through the BTP Operator still operational. In this case, when calling deprovisioning in the BTP Cockpit, users are informed
there are still instances provisioned by BTP Operator, and the user is expected to handle the cleanup.

There is one exception, and that is the `subaccount-cleanup` job. [KEB parses the `User-Agent` HTTP header](https://github.com/kyma-project/control-plane/pull/2520) for
`DELETE` call on `/service_instances/${instance_id}` endpoint and forwards it through the operation to the processing step `btp_operator_cleanup` handling
soft delete for existing BTP Operator resources. Because the `subaccount-cleanup` job is triggered automatically and deletes only SKRs where the whole subaccount is 
intended for deletion, it is necessary to execute the BTP Operator cleanup procedure as well.

