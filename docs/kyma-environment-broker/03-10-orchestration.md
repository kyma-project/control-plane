---
title: Orchestration
type: Details
---

Orchestration is a mechanism that allows you to upgrade Kyma Runtimes. To create an orchestration, [follow this tutorial](#tutorials-orchestrate-kyma-upgrade). After sending the request, the orchestration is processed by `KymaUpgradeManager`. It lists Shoots (Kyma Runtimes) in the Gardener cluster and narrows them to the IDs that you have specified in the request body. Then, `KymaUpgradeManager` performs the [upgrade steps](#details-runtime-operations) logic on the selected Runtimes.

If Kyma Environment Broker is restarted, it reprocesses the orchestration with the `IN PROGRESS` state.

>**NOTE:** You need an OIDC ID token in the JWT format issued by a (configurable) OIDC provider which is trusted by Kyma Environment Broker. The `groups` claim must be present in the token, and furthermore the user must belong to the configurable admin group (`runtimeAdmin` by default) to create an orchestration. To fetch the orchestrations, the user must belong to the configurable operator group (`runtimeOperator` by default).

Orchestration API consist of the following handlers:

- `GET /orchestrations` - exposes data about all orchestrations.
- `DELETE /orchestrations` - cancels in progress orchestration.
- `GET /orchestrations/{orchestration_id}` - exposes status of a single orchestration.
- `DELETE /orchestrations/{orchestration_id}` - cancels in progress orchestration with a given ID.
- `GET /orchestrations/{orchestration_id}/operations` - exposes data about operations scheduled by the orchestration with a given ID.
- `GET /orchestrations/{orchestration_id}/operations/{operation_id}` - exposes the detailed data about a single operation with a given ID.
- `POST /upgrade/kyma` - schedules the orchestration. It requires specifying a request body.

For more details, follow the tutorial on how to [check API using Swagger](#tutorials-check-api-using-swagger).

## Strategies

To change the behavior of the orchestration, you can specify a **strategy** in the request body.
For now, there is only one **parallel** strategy with two types of schedule:

- Immediate - schedules the upgrade operations instantly.
- MaintenanceWindow - schedules the upgrade operations with the maintenance time windows specified for a given Runtime.

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

## Cancellation

You may cancel any in progress orchestration using one of two DELETE endpoints described above.

After you cancel an orchestration, the KEB sets its state to `Canceled`. Orchestration with such state is not scheduling any new operations.

To provide consistency, cancelled orchestration waits for already processed operations to finish. When operations are finished the orchestration queue processing becomes unblocked.