# Orchestrate Kyma upgrade

This tutorial shows how to upgrade Kyma Runtime using Kyma Environment Broker.

## Prerequisites

- Compass with:
  * Runtime Provisioner [configured](../provisioner/08-02-provisioning-gardener.md) for Azure
  * Kyma Environment Broker configured and chosen [overrides](https://kyma-project.io/docs/kyma/latest/04-operation-guides/operations/03-change-kyma-config-values/) set up

## Steps

1. [Get the OIDC ID token in the JWT format](03-10-orchestration.md). Export this variable based on the token you got from the OIDC client:

   ```bash
   export AUTHORIZATION_HEADER="Authorization: Bearer $ID_TOKEN"
   ```

2. Make a call to the Kyma Environment Broker to orchestrate the upgrade. You can select specific Runtimes to upgrade using the following selectors:

- `target` - use the `target: "all"` selector to select all Runtimes
- `globalAccount` - use it to select Runtimes with the specified global account ID
- `subAccount` - use it to select Runtimes with the specified subaccount ID
- `runtimeID` - use it to select Runtimes with the specified Runtime ID
- `planName` - use it to select Runtimes with the specified plan name
- `region` - use it to select Runtimes located in the specified region

   ```bash
   curl --request POST "https://$BROKER_URL/upgrade/kyma" \
   --header "$AUTHORIZATION_HEADER" \
   --header 'Content-Type: application/json' \
   --data-raw "{\
       \"targets\": {\
           \"include\": [{\
               \"runtimeID\": \"uuid-sdasd-sda23t-efs\",\
               \"globalAccount\": \"uuid-sdasd-sda23t-efs\",\
               \"subAccount\": \"uuid-sdasd-sda23t-efs\",\
               \"planName\": \"azure\",\
               \"region\": \"europewest\"\
            }]\
       },\
       \"dryRun\": false\
   }"
   ```

>**NOTE:** If the **dryRun** parameter specified in the request body is set to `true`, the upgrade is executed but the upgrade request is not sent to Runtime Provisioner.

3. If you want to configure [the strategy of your orchestration](03-10-orchestration.md#strategies), use the following request example:

```bash
curl --request POST "https://$BROKER_URL/upgrade/kyma" \
--header "$AUTHORIZATION_HEADER" \
--header 'Content-Type: application/json' \
--data-raw "{\
    \"targets\": {\
        \"include\": [{\
            \"runtimeID\": \"uuid-sdasd-sda23t-efs\",\
            \"globalAccount\": \"uuid-sdasd-sda23t-efs\",\
            \"subAccount\": \"uuid-sdasd-sda23t-efs\",\
            \"planName\": \"azure\",\
            \"region\": \"europewest\",\
         }]\
    },\
    \"strategy\": {\
        \"type\": \"parallel\",\
            \"schedule\": \"maintenanceWindow\",\
            \"parallel\": {\
              \"workers\": 5\
            },\
    },\
    \"dryRun\": false\
}"
```

>**NOTE:** By default, the orchestration is configured with the parallel strategy, using the immediate type of schedule with only one worker.

A successful call returns the orchestration ID:

   ```json
   {
       "orchestrationID":"8a7bfd9b-f2f5-43d1-bb67-177d2434053c"
   }
   ```

4. [Check the orchestration status](08-06-orchestration-status.md).

>**NOTE:** Only one orchestration request can be processed at the same time. If KEB is already processing an orchestration, the newly created request waits for processing with the `PENDING` state.
