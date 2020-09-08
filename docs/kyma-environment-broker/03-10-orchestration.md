---
title: Orchestration
type: Details
---

Orchestration is a mechanism that allows you to upgrade Kyma Runtimes. To create an orchestration, [send the upgrade request to the proper KEB handler](#tutorials-upgrade-kyma-runtime-using-keb). After sending the request, the orchestration is processed by `KymaUpgradeManager`. It lists Shoots (Kyma Runtimes) in the Gardener cluster and filters the Runtime IDs that you have specified for the upgrade in the request body. Then, the `KymaUpgradeManager` proceeds with the orchestration and performs the [upgrade steps](#details-runtime-operations) logic on the Runtimes that you have specified.

>**NOTE:** You need a token with the `broker-upgrade:write` authorization scope to create an orchestration, and a token with the `broker-upgrade:read` scope to fetch the orchestrations.

>**NOTE:** For now, you can upgrade only one Runtime in a single request in KEB.

Orchestration API consist of the following handlers:

- **GET** `/orchestrations` - exposes data about a single orchestration status.
- **GET** `/orchestrations/{orchestration_id}` - exposes data about all the orchestrations.
- **POST** `/upgrade/kyma` - schedules the orchestration of the Kyma upgrade. Requires specifying a request body.

For more details about API check the [Swagger schema](https://app.swaggerhub.com/apis/kempski/kyma-orchestration_api/0.1).
