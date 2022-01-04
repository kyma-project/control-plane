# Set a custom list of runtime administrators

Kyma Environment Broker allows you to set a custom list of runtime administrators during SKR provisioning and update operations.
To do so, specify the additional **administrators** parameter in the provisioning or update request.
>**NOTE:** Make sure to provide at least one administrator in the list. Empty list causes a validation error.

In the provisioning request, the **administrators** parameter overwrites the default administrator list that is taken from the **user_id** field.
See the example:

```bash
   export VERSION=1.15.0
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\",
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
           \"user_id\": \"$USER_ID\",
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"administrators\":[\"admin1@test.com\",\"admin2@test.com\"]
       }
   }"
```

In the update request, the **administrators** parameter overwrites the last list of administrators.
See the example:

```bash
   export VERSION=1.15.0
   curl --request PATCH "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\",
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
       },
       \"parameters\": {
           \"administrators\":[\"admin3@test.com\"]
       }
   }"
```

>**NOTE:** You can't use the **user_id** field to overwrite the administrators list. Use the **administrators** parameter instead.
