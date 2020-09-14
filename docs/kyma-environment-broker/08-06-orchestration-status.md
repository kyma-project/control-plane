---
title: Check orchestration status
type: Tutorials
---

This tutorial shows how to check the orchestration status for the cluster and Kyma orchestrations. You can either:
- Fetch a single orchestration.
- Fetch all orchestrations.


## Fetch a single orchestration

1. Export the orchestration ID that you obtained during the upgrade call as an environment variable:

   ```bash
   export ORCHESTRATION_ID={OBTAINED_ORCHESTRATION_ID}
   ```

2. Make a call to the Kyma Environment Broker with a proper **Authorization** [request header](#details-authorization) to verify that the orchestration succeeded.

   ```bash
   curl --request GET "https://$BROKER_URL/orchestrations/$ORCHESTRATION_ID--header "$AUTHORIZATION_HEADER""
   ```

A successful call returns the orchestration object:

   ```json
    {
      "state": "InProgress",
      "description": "scheduled 5 operations",
      "parameters": {
        "targets": {
          "include": [
            {
              "target": "all",
            },
          ],
          "exclude": [
            {
              "runtimeID": "uuid",
              "globalAccountId": "uuid",
              "subAccountId": "uuid",
              "region": "region",
            },
          ],
        },
        "strategy": ...
      }
      "runtimeOperations": [
        {
          "instanceID": "054ac2c2-318f-45dd-855c-eee41513d40d",
          "runtimeID": "44a57cbd-5271-4d68-8cf9-9dabbb9f1c44",
          "globalAccountID": "",
          "subAccountID": "",
          "clusterName": "c-084befc"
          "operationID": "f683e77c-7d24-4aee-91af-4208bcfc480f",
          "state": "InProgress" / "Pending" / "Succeeded" / "Failed"
        }
        [...]
      ]
    }
   ```

## Fetch all orchestrations

Make a call to the Kyma Environment Broker with a proper **Authorization** [request header](#details-authorization) to verify that the orchestration succeeded.

   ```bash
   curl --request GET "https://$BROKER_URL/orchestrations --header "$AUTHORIZATION_HEADER""
   ```

A successful call returns a list of all orchestrations:

   ```json
    [{
      "state": "InProgress",
      "description": "scheduled 5 operations",
      "parameters": {
        "targets": {
          "include": [
            {
              "target": "all",
            },
          ],
          "exclude": [
            {
              "runtimeID": "uuid",
              "globalAccountId": "uuid",
              "subAccountId": "uuid",
              "region": "region",
            },
          ],
        },
        "strategy": ...
      }
      "runtimeOperations": [
        {
          "instanceID": "054ac2c2-318f-45dd-855c-eee41513d40d",
          "runtimeID": "44a57cbd-5271-4d68-8cf9-9dabbb9f1c44",
          "globalAccountID": "",
          "subAccountID": "",
          "clusterName": "c-084befc"
          "operationID": "f683e77c-7d24-4aee-91af-4208bcfc480f",
          "state": "InProgress" / "Pending" / "Succeeded" / "Failed"
        }
        [...]
      ]
    }]
   ```
