---
title: Orchestration
type: Details
---

Orchestration is an abstraction which allows to aggregate Kyma and cluster upgrades. User needs to send a request on a [proper KEB's handler](##tutorials-upgrade-kyma-runtime-using-keb) to create an Orchestration.

After creation, orchestration is being processed by KymaUpgradeManager which resolves Shoots on the gardener cluster and filters them using input provided by the User in the request body.

When Orchestration knows its targets, it starts to process them in a separate queue as the Upgrade Operation using the steps logic. 

User can always check the status of the Orchestration and its Operations using one of the `GET` endpoints.

>**NOTE:** For now only Kyma upgrade is supported

To create Orchestrations use token with `broker-upgrade:write` authorization and to fetch them use the `broker-upgrade:read` scope.

Check how to consume the Orchestration API by using [this](https://app.swaggerhub.com/apis/kempski/kyma-orchestration_api/0.1).