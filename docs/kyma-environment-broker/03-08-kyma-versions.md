---
title: Configure Kyma version
type: Details
---

Kyma Environment Broker is configured with a default Kyma Version (the APP_KYMA_VERSION environment variable). 
You can also specify a default version per Global Account using a config map, for example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kyma-versions
  namespace: "kcp-system"
data:
  3e64ebae-38b5-46a0-b1ed-9ccee153a0ae: "1.15-rc1"
```

The ConfigMap contains default versions for Global Account. The key is a Global Account ID, the value is the version.
If the global account is not configured in the ConfigMap, the default value set by APP_KYMA_VERSION environment variable is be used.

A user can also specify Kyma version using a provisioning parameter `kymaVersion`, for example:

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
This feature is enabled by the APP_ENABLE_ON_DEMAND_VERSION environment variable.
