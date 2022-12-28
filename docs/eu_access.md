# EU access

## Overview

EU Access requires, among others, that Data Residency is in the European Economic Area or Switzerland. 
For more information, see [EU Access Overview](https://wiki.one.int.sap/wiki/display/IntBusComp/EU+Access+Overview). 

BTP Kyma runtime service supports the BTP `cf-eu11` AWS and BTP `cf-ch20` Azure subaccount regions which are
called EU Access BTP subaccount regions. 
Kyma Control Plane manages `cf-eu11` Kyma runtimes in a separate AWS hyperscaler account pool and 
`cf-ch20` Kyma runtimes in a separate Azure hyperscaler account pool.

When the PlatformRegion is an EU access BTP subaccount region:
-  Kyma Environment Broker (KEB) provides the `euAccess` parameter to Provisioner
- KEB services catalog handler being aware of the platform regions:
  - for `cf-eu11` exposes `eu-central-1` as possible value for the `region` parameter
  - for `cf-ch20` exposes `switzerlandnorth` as possible value for the `region` parameter