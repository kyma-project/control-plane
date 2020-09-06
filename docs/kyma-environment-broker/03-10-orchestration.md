---
title: Orchestration
type: Details
---

Orchestration is a mechanism that allows you to upgrade Kyma Runtimes. To create an orchestration, [send a request to the proper KEB handler](#tutorials-upgrade-kyma-runtime-using-keb).

After sending the request, the orchestration is processed by `KymaUpgradeManager` which lists Shoots (Kyma Runtimes) in the Gardener cluster and filters the Runtime IDs that you have specified for the upgrade in the request body. For now, KEB supports only a single Runtime's upgrade per request.

Then, the `KymaUpgradeManager` proceeds with the orchestration and performs the [upgrade steps](#details-runtime-operations) logic on the Runtimes that you have specified. 

To check the status of the orchestration, use one of the `GET` endpoints.

Use the Swagger to [check how to consume the Kyma Orchestration API](https://app.swaggerhub.com/apis/kempski/kyma-orchestration_api/0.1).

>**NOTE:** You need a token with the `broker-upgrade:write` authorization scope to create an orchestration, and a token with the `broker-upgrade:read` scope to fetch the orchestrations.
