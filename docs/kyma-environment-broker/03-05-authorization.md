# Authorization

Kyma Environment Broker provides OAuth2 authorization. For this purpose, Kyma Environment Broker uses the [ApiRule](https://kyma-project.io/docs/kyma/latest/05-technical-reference/00-custom-resources/apix-01-apirule/) custom resource which generates a [VirtualService](https://istio.io/docs/reference/config/networking/virtual-service/) and uses  [Oathkeeper Access Rules](https://www.ory.sh/docs/oathkeeper/api-access-rules) to allow or deny access.
To authorize with the Kyma Environment Broker, use an OAuth2 client registered through the [Hydra Maester controller](https://github.com/ory/k8s/blob/master/docs/helm/hydra-maester.md).

To access the Kyma Environment Broker endpoints, use the `/oauth` prefix before OSB API paths. For example:

```shell
/oauth/{region}/v2/catalog
```

You must also specify the `Authorization: Bearer` request header:

```shell
Authorization: Bearer {ACCESS_TOKEN}
```

## Get the access token

Follow these steps to obtain a new access token:

1. Export these values as environment variables:

```shell
export CLIENT_ID={CLIENT_ID}
export CLIENT_SECRET={CLIENT_SECRET}
export TOKEN_URL={TOKEN_URL}
export SCOPE="broker:write cld:read"
```

2. Encode your client credentials and export them as environment variables:

```shell
export ENCODED_CREDENTIALS=$(echo -n "$CLIENT_ID:$CLIENT_SECRET" | base64)
```

3. Get the access token:

```shell
curl -ik -X POST "$TOKEN_URL" -H "Authorization: Basic $ENCODED_CREDENTIALS" -F "grant_type=client_credentials" -F "scope=broker:write"
```
