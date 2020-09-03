---
title: Orchestration
type: Details
---

Orchestration is an abstraction that allows you to aggregate Kyma and cluster upgrades. To create an orchestration, [send a request to the proper KEB handler](#tutorials-upgrade-kyma-runtime-using-keb).

After sending the request, the orchestration is processed by KymaUpgradeManager which resolves Shoots in the Gardener cluster and filters them using input provided in the request body.

When Orchestration knows its targets, it starts to process them in a separate queue as the Upgrade Operation using the steps logic. 

You can check the status of the Orchestration and its operations using one of the `GET` endpoints.

>**NOTE:** For now only Kyma upgrade is supported

To create an Orchestrations, use a token with the `broker-upgrade:write` authorization scope. To fetch Orchestrations, use the `broker-upgrade:read` scope.

Use the Swagger to [check how to consume the Kyma Orchestration API](https://app.swaggerhub.com/apis/kempski/kyma-orchestration_api/0.1).
