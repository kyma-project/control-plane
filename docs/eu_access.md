# EU access

## Overview

Eu Access requires (among others) that Data Residency is in European Economic Area or Switzerland 
(see [EU Access overview](https://wiki.one.int.sap/wiki/display/IntBusComp/EU+Access+Overview) for further details). 

BTP Kyma runtime service supports the BTP `cf-eu11` AWS subaccount region and BTP `cf-ch20` Azure subaccount region 
called EU Access BTP subaccount regions. 
Kyma control plane manages `cf-eu11` Kyma runtimes in separate AWS hyperscaler account pool and 
`cf-ch20` Kyma runtimes in separate Azure hyperscaler account pool.

When the PlatformRegion is an EU access BTP subaccount region, Kyma Environment Broker (KEB):
- provides the `euAccess` parameter to the provisioner
- KEB services catalog handler being aware of the platform regions:
  - for `cf-eu11` exposes `eu-central-1` as possible value for the `region` parameter
  - for `cf-ch20` exposes `switzerlandnorth` as possible value for the `region` parameter