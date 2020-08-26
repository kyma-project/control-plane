---
title: Orchestration
type: Details
---

## Handlers 

Orchestration handlers allows to fetch orchestration status and to run upgrade of kyma or cluster.

The handlers are as follows:

- `GET /orchestrations/{orchestration_id}`

**Responds** with the [orchestration](https://github.com/kyma-project/control-plane/blob/master/components/kyma-environment-broker/internal/model.go) object. 
Check the below example. 

```
{
  "state": "InProgress",
  "description": "Blabla bla",
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

- `POST /upgrade/kyma`

Specified with the following **body**:

```
{
  "targets": {
    "include": [
      { "target": "all" }
    ],
    "exclude": [
      {"globalAccount": "....", "subAccount": "..."},
    ]
  },
  "strategy": {
    "schedule": "immediate" | "maintenanceWindow",
    "parallel": {
      "workers": 4,
    }
  }
}
```

You can find its structure [here](https://github.com/kyma-project/control-plane/blob/master/components/kyma-environment-broker/internal/model.go).

**Responds** with newly created orchestrationID:

```
{
  "orchestration_id": "uuid"
}
```