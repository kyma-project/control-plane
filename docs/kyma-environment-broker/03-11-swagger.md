# Check API using Swagger

Using the Swagger UI, you can visualize Kyma Environment Broker's (KEB's) APIs on a single page.

The Swagger UI static files are copied from the [official source](https://github.com/swagger-api/swagger-ui/tree/master/dist) and then they are injected into KEB's container which exposes them on the root endpoint.

KEB uses a Swagger schema file that is mounted as a volume to the Pod. You can find that schema [here](https://github.com/kyma-project/control-plane/blob/main/resources/kcp/charts/kyma-environment-broker/files/swagger.yaml). You can use templates in the Swagger schema file to configure it.

You can either use Virtual Service or port-forward the Pod to expose and use the Swagger UI.

## Use Virtual Service

Open the following website:

   ```
   https://$BROKER_URL/
   ```

To use the `Try it out` feature on Virtual Service, you need this [Chrome browser extension](https://chrome.google.com/webstore/detail/allow-cors-access-control/lhobafahddgcelffkeicbaginigeejlf).

## Port-forward the Pod

Use the following command to port-forward the Pod:

   ```bash
   kubectl port-forward -n kcp-system svc/kcp-kyma-environment-broker 8888:80
   ```

Open the following website:

   ```
   http://localhost:8888/
   ```
