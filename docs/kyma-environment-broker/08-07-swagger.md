---
title: Check API using swagger
type: Tutorials
---

This tutorial shows how to check the API schema using Swagger.

The Swagger is injected into the application and it's exposes the Swagger UI on the `/swagger/` endpoint.

The Swagger UI static files are copied from the [official source](https://github.com/swagger-api/swagger-ui/tree/master/dist).

The KEB uses swagger schema file mounted as volume to the Pod. You can find that schema [here](https://github.com/kyma-project/control-plane/blob/master/resources/kcp/charts/kyma-environment-broker/files/swagger.yaml).

You can use several ways to expose and use the Swagger UI, but two of them are recommended.

## Use Virtual Service

Open the following website:

   ```
   https://$BROKER_URL/swagger/
   ```

> **NOTE:** Using this option you can't use `Try it out` option due to not configured OAuth2 swagger schema.

## Port-forward the Pod

Use the following command to port-forward the Pod:

   ```bash
   kubectl port-forward -n kcp-system svc/kcp-kyma-environment-broker 8888:80
   ```

Open the following website:

   ```
   http://localhost:8888/swagger/
   ```

