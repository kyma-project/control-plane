# EU access

## Overview

EU Access requires, among others, that Data Residency is in the European Economic Area or Switzerland. 
For more information, see [EU Access Overview](https://wiki.one.int.sap/wiki/display/IntBusComp/EU+Access+Overview). 

BTP Kyma runtime service supports the BTP `cf-eu11` AWS and BTP `cf-ch20` Azure subaccount regions which are
called EU Access BTP subaccount regions. 
Kyma Control Plane manages `cf-eu11` Kyma runtimes in a separate AWS hyperscaler account pool and 
`cf-ch20` Kyma runtimes in a separate Azure hyperscaler account pool.

When the PlatformRegion is an EU access BTP subaccount region:
- Kyma Environment Broker (KEB) provides the `euAccess` parameter to Provisioner
- KEB services catalog handler exposes:
  - `eu-central-1` as the only possible value for the `region` parameter for `cf-eu11` 
  - `switzerlandnorth` as the only possible value for the `region` parameter for `cf-ch20`

## Access 
Due to limited availability, the provisioning request for the EU Access only regions can succeed only if GlobalAccountId 
is added to the list of allowed GlobalAccountIds (the whitelist).
The list is configured in [management-plane-config repository](https://github.tools.sap/kyma/management-plane-config) 
in resources/control-plane/<landscape>/values.yaml files.

Here you can find example configuration of the list with two GlobalAccountIds listed:
```yaml
kcp-prod:
  kyma-environment-broker:
    euAccessWhitelistedGlobalAccountIds: |-
      whitelist:
        - 2358e708-68f0-4af0-94b6-cf4e8407aff8
        - 4a8fa8f1-d76b-4682-89ba-84fe7591a07c
```

One has to open a support ticket before attempting to provision Kyma clusters in EU Access only regions to have one's 
GlobalAccountId added to the above-mentioned list.

If the GlobalAccountId for provisioning request is not whitelisted the Kyma Environment Broker will respond 
with http code 400 (Bad Request) and preconfigured in management-plane-config message. 
This message will be presented to the user in the BTP Cockpit.   
```yaml
kcp-prod:
  kyma-environment-broker:
    euAccessRejectionMessage: "due to limited availability you need to open support ticket before attempting to provision Kyma clusters in EU Access only regions"
```

