# Check orchestration status

This tutorial shows how to check the orchestration status. Using this API, you can fetch data about:
- A single orchestration
- All orchestrations
- Upgrade operations scheduled by a given orchestration
- A single operation with details, such as parameters sent to Runtime Provisioner

## Fetch a single orchestration status

1. Export the orchestration ID that you obtained during the upgrade call as an environment variable:

   ```bash
   export ORCHESTRATION_ID={OBTAINED_ORCHESTRATION_ID}
   ```

2. Make a call to the Kyma Environment Broker with a proper **Authorization** [request header](03-10-orchestration.md) to verify that the orchestration succeeded.

   ```bash
   curl --request GET "https://$BROKER_URL/orchestrations/$ORCHESTRATION_ID --header "$AUTHORIZATION_HEADER""
   ```

   A successful call returns the orchestration object:
   ```json
      {
          "orchestrationID": "07089b96-8e31-49a4-96d0-f8288253c804",
          "state": "in progress",
          "description": "Scheduled 2 operations",
          "createdAt": "2020-10-12T19:27:22.281515Z",
          "updatedAt": "2020-10-12T19:27:22.281515Z",
          "parameters": {
              "targets": {
                  "include": [
                      {
                          "target": "all"
                      }
                  ]
              },
              "strategy": {
                  "type": "parallel",
                  "schedule": "immediate",
                  "parallel": {
                      "workers": 1
                  }
              },
              "dryRun": true
          }
      }
   ```

## Fetch all orchestrations status

Make a call to the Kyma Environment Broker with a proper **Authorization** [request header](03-10-orchestration.md) to verify that the orchestration succeeded.

   ```bash
   curl --request GET "https://$BROKER_URL/orchestrations --header "$AUTHORIZATION_HEADER""
   ```

A successful call returns the list of all orchestrations:

   ```json
[
  {
      "orchestrationID": "07089b96-8e31-49a4-96d0-f8288253c804",
      "state": "in progress",
      "description": "Scheduled 2 operations",
      "createdAt": "2020-10-12T19:27:22.281515Z",
      "updatedAt": "2020-10-12T19:27:22.281515Z",
      "parameters": {
          "targets": {
              "include": [
                  {
                      "target": "all"
                  }
              ]
          },
          "strategy": {
              "type": "parallel",
              "schedule": "immediate",
              "parallel": {
                  "workers": 1
              }
          },
          "dryRun": true
      }
  }
]
   ```

## List upgrade operations scheduled by an orchestration

1. Export the orchestration ID that you obtained during the upgrade call as an environment variable:

   ```bash
   export ORCHESTRATION_ID={OBTAINED_ORCHESTRATION_ID}
   ```

2. Make a call to the Kyma Environment Broker with a proper **Authorization** [request header](03-10-orchestration.md) to fetch the list of the upgrade operations for a given orchestration.

   ```bash
   curl --request GET "https://$BROKER_URL/orchestrations/$ORCHESTRATION_ID/operations --header "$AUTHORIZATION_HEADER""
   ```

   A successful call returns the list of upgrade operations:

      ```json
   {
       "data": [
           {
               "operationID": "c4aa1f4b-be2a-4e8d-90e6-edd00194aaa9",
               "runtimeID": "5791e81d-8959-4b78-82e4-7e4edea45683",
               "globalAccountID": "3e64ebae-38b5-46a0-b1ed-9ccee153a0ae",
               "subAccountID": "39b19a66-2c1a-4fe4-a28e-6e5db434084e",
               "orchestrationID": "17089b96-8e31-49a4-96d0-f8288253c804",
               "servicePlanID": "ca1e5357-707f-4565-bbbd-b3ab732597c6",
               "servicePlanName": "gcp",
               "dryRun": true,
               "shootName": "c-3a3xdaf",
               "maintenanceWindowBegin": "0000-01-01T04:00:00Z",
               "maintenanceWindowEnd": "0000-01-01T08:00:00Z"
           },
           {
               "operationID": "669c1644-44c2-349d-a3c5-8bc63dceff93",
               "runtimeID": "8071f119-29a5-3e81-bae3-04751881f317",
               "globalAccountID": "3e64ebae-68b5-46a0-b1ed-9ccee153a0ae",
               "subAccountID": "A791EFE6-6121-1714-9933-E2D3D8CA2992",
               "orchestrationID": "a7089b96-8e31-49a4-96d0-f8288253c804",
               "servicePlanID": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
               "servicePlanName": "azure",
               "dryRun": true,
               "shootName": "c-5d2xd83",
               "maintenanceWindowBegin": "0000-01-01T22:00:00Z",
               "maintenanceWindowEnd": "0000-01-01T02:00:00Z"
           }
       ],
       "count": 2,
       "totalCount": 2
   }
      ```

## Fetch the detailed operation status

1. Export the following values as the environment variables:

   ```bash
   export ORCHESTRATION_ID={OBTAINED_ORCHESTRATION_ID}
   export OPERATION_ID={SCHEDULED_OPERATION_ID}
   ```

2. Make a call to the Kyma Environment Broker with a proper **Authorization** [request header](03-10-orchestration.md) to fetch the details for given operation.

   ```bash
   curl --request GET "https://$BROKER_URL/orchestrations/$ORCHESTRATION_ID/operations/$OPERATION_ID --header "$AUTHORIZATION_HEADER""
   ```

   A successful call returns the upgrade operation object with the **kymaConfig** and **clusterConfig** fields:

      ```json
   {
       "operationID": "c4aadf4b-be2a-4e8d-90e6-edd00194aaa9",
       "runtimeID": "5791e84d-8959-4b78-82e4-7e4edea45683",
       "globalAccountID": "3e64ebae-38b5-46a0-b1ed-9ccee153a0ae",
       "subAccountID": "39ba9a66-2c1a-4fe4-a28e-6e5db434084e",
       "orchestrationID": "07089b96-8e31-49a4-96d0-f8288253c804",
       "servicePlanID": "ca6e5357-707f-4565-bbbd-b3ab732597c6",
       "servicePlanName": "gcp",
       "dryRun": true,
       "shootName": "c-3a3e0af",
       "maintenanceWindowBegin": "0000-01-01T04:00:00Z",
       "maintenanceWindowEnd": "0000-01-01T08:00:00Z",
       "kymaConfig": {
           "version": "1.15.1",
           "components": [],
           "configuration": []
       },
       "clusterConfig": {}
   }
      ```
