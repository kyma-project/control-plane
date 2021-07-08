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

If you do not provide the `oidc` object in the provisioning request or leave all object's properties empty, the default OIDC configuration is used.
However, if you do not provide the `oidc` object in the update request or leave all object’s properties empty, the saved OIDC configuration stays untouched.
See the following JSON example without the `oidc` object:
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
See the following JSON example with the `oidc` object whose properties are empty:
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
This is the default OIDC configuration:
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

To update the OIDC configuration, provide values for the mandatory properties. Without these values, a validation error occurs.
The update operation overwrites the OIDC configuration values provided in JSON. It means that OIDC properties with empty values are considered valid. See the following scenario:

   1. An existing instance has the following OIDC configuration:
   ```
   ClientID: 9bd05ed7-a930-44e6-8c79-e6defeb7dec9
   IssuerURL: https://kymatest.accounts400.ondemand.com
   GroupsClaim: groups
   UsernameClaim: sub
   UsernamePrefix: -
   SigningAlgs: RS256
   ```
   2. A user sends an update request (HTTP PUT) with the following JSON in the payload:
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
   3. The OIDC configuration is updated to include the values of the `oidc` object from JSON provided in the update request:
   ```
   ClientID: new-client-id
   IssuerURL: https://new-issuer-url.local.com
   GroupsClaim: 
   UsernameClaim: 
   UsernamePrefix: 
   SigningAlgs: 
   ```
