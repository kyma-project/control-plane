---
title: Upgrade shoots
type: Tutorials
---

This tutorial shows how to upgrade Gardener shoots for Kyma Runtimes.

## Steps

> **NOTE:** To access the Runtime Provisioner, forward the port on which the GraphQL Server is listening.

To upgrade a Gardener Shoot used to host the Runtime of a given ID, make a call to the Runtime Provisioner with a **tenant** header using a mutation like this:  

```graphql
mutation { 
  upgradeShoot(
    id: "61d1841b-ccb5-44ed-a9ec-45f70cd1b0d3"
    config: {
      gardenerConfig: {
        kubernetesVersion: "1.15.11"
        volumeSizeGB: 35
        machineType: "Standard_D2_v3"
        diskType: "pd-standard"
        purpose: "testing"
        autoScalerMin: 2
        autoScalerMax: 4
        maxSurge: 4
        maxUnavailable: 1
        enableKubernetesVersionAutoUpdate: false
        enableMachineImageVersionAutoUpdate: false
        providerSpecificConfig: { 
          azureConfig: {
            zones: ["1", "2"]
          } 
        }
      }
    }
  ) 
}
```

A successful call returns the ID of the upgrading operation:

```json
{
  "data": {
    "upgradeShoot": "c7e6727f-16b5-4748-ac95-197d8f79d094"
  }
}
```

The operation of upgrading is asynchronous. Use the upgrade operation ID (`upgradeShoot`) to [check the Runtime Operation Status](08-03-runtime-operation-status.md) and verify that the upgrade was successful. Use the Runtime ID (`id`) to [check the Runtime Status](08-04-runtime-status.md). 