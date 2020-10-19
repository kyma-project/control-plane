---
title: Upgrade Kyma Runtime using KEB
type: Tutorials
---

This tutorial shows how to upgrade Kyma Runtime using Kyma Environment Broker.

## Prerequisites

- Compass with:
  * Runtime Provisioner [configured](/control-plane/runtime-provisioner/#tutorials-provision-clusters-through-gardener) for Azure
  * Kyma Environment Broker configured and chosen [overrides](#details-set-overrides-for-kyma-runtime) set up

## Steps

1. [Get the access token](#details-authorization). Export this variable based on the token you got from the OAuth client:

   ```bash
   export AUTHORIZATION_HEADER="Authorization: Bearer $ACCESS_TOKEN"
   ```

2. Make a call to the Kyma Environment Broker to orchestrate the upgrade. You can select runtimes to upgrade using a following selectors:

- target - can be used to select all runtimes by specifying it as `target: "all"`
- globalAccount
- subAccount
- runtimeID
- planName
- region

>**NOTE:** If the **dryRun** parameter specified in the request body is set to `true`, the upgrade is executed but the process is not sending the upgrade request to the Provisioner.

   ```bash
   curl --request POST "https://$BROKER_URL/upgrade/kyma" \
   --header "$AUTHORIZATION_HEADER" \
   --header 'Content-Type: application/json' \
   --data-raw "{
       \"targets\": {
           \"include\": {\
               \"runtimeID\": "uuid-sdasd-sda23t-efs",
               \"globalAccount\": "uuid-sdasd-sda23t-efs",
               \"subAccount\": "uuid-sdasd-sda23t-efs",
               \"planName\": "azure",
               \"region\": "europewest",
            },
       },
       \"dryRun\": false
   }"
   ```

>**NOTE:** By default, the orchestration will select all the runtimes using the **all** target.

A successful call returns the orchestration ID:

   ```json
   {
       "orchestrationID":"8a7bfd9b-f2f5-43d1-bb67-177d2434053c"
   }
   ```

4. [Check the orchestration status](#tutorials-check-orchestration-status).

>**NOTE:** Only one orchestration can be processed at the same time. If the KEB is already processing some orchestration the newly created will wait for processing with the **pending** state.

### Strategies

To change the behaviour of the orchestration you can specify a strategy in the request body.

For now we support only the **Parallel** strategy with two types of schedule:

- Immediate - schedules the upgrade operations instantly
- MaintenanceWindow - schedules the upgrade operations respecting the maintenance time windows specified for given runtime

You can also configure how many upgrade operations can be executed in parallel to accelerate the process. Specify the **parallel** object in the request body with **workers** field set to the number of concurrent executions for the upgrade operations.

The example strategy objects looks as follows:

```json
{
  "strategy": {
    "type": "parallel",
    "schedule": "maintenanceWindow",
    "parallel": {
      "workers": 5
    }
  }
}
```

>**NOTE:** By default the orchestration will be executed with Parallel strategy, using the immediate type of schedule with only one worker.
