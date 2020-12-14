---
title: Configure Kyma version
type: Details
---

Kyma Environment Broker is configured with a default Kyma version specified in the **APP_KYMA_VERSION** environment variable. This means that each Kyma Runtime provisioned by Kyma Environment Broker in a given global account is installed in the default Kyma version.
You can also specify a different Kyma version for a global account or subaccount using a ConfigMap. To specify a version for a given account, use the following prefixes in ConfigMap keys:
- `GA_` for a global account
- `SA_` for a subaccount

The subaccount version has a higher priority than the global account mapping. When provided, the version for a subaccount is used as long as the subaccount ID in the provisioning request matches the subaccount ID in the ConfigMap. See the example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kyma-versions
  namespace: "kcp-system"
data:
  GA_3e64ebae-38b5-46a0-b1ed-9ccee153a0ae: "1.15.0-rc1"
  SA_df29c526-0c3d-4e2c-ba41-8b69cde41961: "PR-3721"
```

This ConfigMap contains a default version for a global account and a subaccount. The **3e64ebae-38b5-46a0-b1ed-9ccee153a0ae** parameter is a global account ID, and the value is the Kyma version specified for this global account. The **df29c526-0c3d-4e2c-ba41-8b69cde41961** parameter is a subaccount ID, and the value is the Kyma version specified for this subaccount.

You can also specify a Kyma version using the **kymaVersion** provisioning parameter, for example:

```bash
   export VERSION=1.15
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --header 'Content-Type: application/json' \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"kymaVersion\": \"$VERSION"
       }
   }"
```
The **kymaVersion** provisioning parameter overrides the default settings.
To enable this feature, set the **APP_ENABLE_ON_DEMAND_VERSION** environment variable to `true`.
