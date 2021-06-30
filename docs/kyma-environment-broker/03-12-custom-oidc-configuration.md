---
title: Custom OIDC configuration
type: Details
---

To create an SKR with a custom OIDC (Open ID Connect) configuration, specify the additional `oidc` provisioning parameters. See the example:

```bash
   export VERSION=1.15.0
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
           \"oidc\": {
              \"clientID\": \"9bd05ed7-a930-44e6-8c79-e6defeb7dec5\",
              \"issuerURL\": \"https://kymatest.accounts400.ondemand.com\",
              \"groupsClaim\": \"groups\",
              \"signingAlgs\": [\"RS256\"],
              \"usernamePrefix\": \"-\",
              \"usernameClaim\": \"sub\"
           }
       }
   }"
```
>NOTE: `clientID` and `issuerURL` values are mandatory for custom OIDC configuration.

If you do not provide the `oidc` object in your provisioning request or leave all object's properties empty, the default OIDC configuration will be used.
However, if you do the same in update request, it will leave the saved OIDC configuration untouched.
In these particular cases, JSON could look like this:
```json
{
  "service_id" : "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id" : "4deee563-e5ec-4731-b9b1-53b42d855f0c",
  "context" : {
    "globalaccount_id" : {GLOBAL_ACCOUNT_ID}
  },
  "parameters" : {
    "name" : {CLUSTER_NAME}
  }
}
```
OR
```json
{
  "service_id" : "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id" : "4deee563-e5ec-4731-b9b1-53b42d855f0c",
  "context" : {
    "globalaccount_id" : {GLOBAL_ACCOUNT_ID}
  },
  "parameters" : {
    "name" : {CLUSTER_NAME},
    "oidc" : {
      "clientID" : "",
      "issuerURL" : "",
      "groupsClaim" : "",
      "signingAlgs" : [],
      "usernamePrefix" : "",
      "usernameClaim" : ""
    }
  }
}
```
The default OIDC configuration looks like this:
```json
{
  ...
    "oidc" : {
      "clientID" : "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
      "issuerURL" : "https://kymatest.accounts400.ondemand.com",
      "groupsClaim" : "groups",
      "signingAlgs" : ["RS256"],
      "usernamePrefix" : "-",
      "usernameClaim" : "sub"
    }
  ...
}
```

If you would like to update the OIDC configuration, please make sure that you provide values for the mandatory properties. Without these values, validation error will occur.
Update operation will overwrite the OIDC configuration values as provided in JSON. That means empty OIDC properties values are considered as valid values for configuration. Consider the following scenario:

   1. Existing instance has the following OIDC configuration:
   ```
   ClientID: 9bd05ed7-a930-44e6-8c79-e6defeb7dec9
   IssuerURL: https://kymatest.accounts400.ondemand.com
   GroupsClaim: groups
   UsernameClaim: sub
   UsernamePrefix: -
   SigningAlgs: RS256
   ```
   2. User sends update request (HTTP PUT) with the following JSON in the payload:
```json
{
  "service_id" : "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id" : "4deee563-e5ec-4731-b9b1-53b42d855f0c",
  "context" : {
    "globalaccount_id" : {GLOBAL_ACCOUNT_ID}
  },
  "parameters" : {
    "name" : {CLUSTER_NAME},
    "oidc" : {
      "clientID" : "new-client-id",
      "issuerURL" : "https://new-issuer-url.local.com",
      "groupsClaim" : "",
      "signingAlgs" : [],
      "usernamePrefix" : "",
      "usernameClaim" : ""
    }
  }
}
```
   3. The OIDC configuration is updated accordingly to the values of `oidc` object in JSON:
   ```
   ClientID: new-client-id
   IssuerURL: https://new-issuer-url.local.com
   GroupsClaim: 
   UsernameClaim: 
   UsernamePrefix: 
   SigningAlgs: 
   ```