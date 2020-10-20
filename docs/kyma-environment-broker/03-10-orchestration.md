---
title: Orchestration
type: Details
---

Orchestration is a mechanism that allows you to upgrade Kyma Runtimes. To create an orchestration, [follow this tutorial](#tutorials-orchestrate-kyma-upgrade). After sending the request, the orchestration is processed by `KymaUpgradeManager`. It lists Shoots (Kyma Runtimes) in the Gardener cluster and narrows them to the IDs that you have specified in the request body. Then, `KymaUpgradeManager` performs the [upgrade steps](#details-runtime-operations) logic on the selected Runtimes.

If Kyma Environment Broker is restarted, it reprocesses the orchestration with the `IN PROGRESS` state. 

>**NOTE:** You need a token with the `broker-upgrade:write` authorization scope to create an orchestration, and a token with the `broker-upgrade:read` scope to fetch the orchestrations.

Orchestration API consist of the following handlers:

- `GET /orchestrations` - exposes data about all orchestrations.
- `GET /orchestrations/{orchestration_id}` - exposes data about a single orchestration status.
- `GET /orchestrations/{orchestration_id}/operations` - exposes data about operations scheduled by the orchestration with a given ID.
- `GET /orchestrations/{orchestration_id}/operations/{operation_id}` - exposes the detailed data about a single operation with a given ID.
- `POST /upgrade/kyma` - schedules the orchestration. It requires specifying a request body.

For more details about the API, check the [Swagger schema](https://app.swaggerhub.com/apis/kempski/kyma-orchestration_api/0.4).

### Strategies

To change the behavior of the orchestration, you can specify a strategy in the request body.
For now, we support only the **parallel** strategy with two types of schedule:

- Immediate - schedules the upgrade operations instantly
- MaintenanceWindow - schedules the upgrade operations with the maintenance time windows specified for a given Runtime

You can also configure how many upgrade operations can be executed in parallel to accelerate the process. Specify the **parallel** object in the request body with **workers** field set to the number of concurrent executions for the upgrade operations.

The example strategy configuration looks as follows:

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