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

1. Export these values as environment variables:

   ```bash
   export RUNTIME_ID={RUNTIME_ID}
   ```

2. [Get the access token](#details-authorization). Export this variable based on the token you got from the OAuth client:

   ```bash
   export AUTHORIZATION_HEADER="Authorization: Bearer $ACCESS_TOKEN"
   ```

3. Make a call to the Kyma Environment Broker to orchestrate the upgrade.

>**NOTE:** The **dry** parameter specified in the request body is set to `true`. This causes that the upgrade is executed in a fake mode. It means that all actions are not executed against selected Runtimes but the Orchestration status is still available.

   ```bash
   curl --request POST "https://$BROKER_URL/upgrade/kyma" \
   --header "$AUTHORIZATION_HEADER" \
   --header 'Content-Type: application/json' \
   --data-raw "{
       \"targets\": {
           \"include\": {\
               \"runtime_id\": "uuid-sdasd-sda23t-efs",
            },
       },
        \"dry\": false,
   }"
   ```

A successful call returns the orchestration ID:

   ```json
   {
       "orchestration_id":"8a7bfd9b-f2f5-43d1-bb67-177d2434053c"
   }
   ```

4. [Check the operation status](#tutorials-check-orchestration-status).
