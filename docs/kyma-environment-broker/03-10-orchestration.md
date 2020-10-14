---
title: Orchestration
type: Details
---

Orchestration is a mechanism that allows you to upgrade Kyma Runtimes. To create an orchestration [fpllow this tutorial](#tutorials-upgrade-kyma-runtime-using-keb). After sending the request, the orchestration is processed by `KymaUpgradeManager`. It lists Shoots (Kyma Runtimes) in the Gardener cluster and filters the Runtime IDs that you have specified in the request body. Then, `KymaUpgradeManager` performs the [upgrade steps](#details-runtime-operations) logic on the Runtimes that you have specified.

If the KEB is restarted, it reprocess orchestration with `in progress` state. 

>**NOTE:** You need a token with the `broker-upgrade:write` authorization scope to create an orchestration, and a token with the `broker-upgrade:read` scope to fetch the orchestrations.

Orchestration API consist of the following handlers:

- `GET /orchestrations` - exposes data about all orchestrations.
- `GET /orchestrations/{orchestration_id}` - exposes data about a single orchestration status.
- `GET /orchestrations/{orchestration_id}/operations` - exposes data about operations scheduled by the orchestration with a given ID.
- `GET /orchestrations/{orchestration_id}/operations/{operation_id}` - exposes the detailed data about a single operation with a given ID.
- `POST /upgrade/kyma` - schedules the orchestration. It requires specifying a request body.

For more details about the API, check the [Swagger schema](https://app.swaggerhub.com/apis/kempski/kyma-orchestration_api/0.4).
