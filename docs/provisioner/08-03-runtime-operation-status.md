---
title: Check Runtime Operation Status
type: Tutorials
---

This tutorial shows how to check the Runtime operation status for the operations of Runtime provisioning and deprovisioning. 

## Steps

> **NOTE:** To access Runtime Provisioner, forward the port on which the GraphQL server is listening.

Make a call to Runtime Provisioner with a **tenant** header to verify that provisioning/deprovisioning succeeded. Pass the ID of the operation as `id`.

```graphql
query { 
  runtimeOperationStatus(id: "e9c9ed2d-2a3c-4802-a9b9-16d599dafd25") { 
    operation 
    state 
    message 
    runtimeID 
  }
}
```

A successful call returns a response which includes the status of the (de)provisioning operation (`state`) and the id of the (de)provisioned Runtime (`runtimeID`):

```json
{
  "data": {
    "runtimeOperationStatus": {
      "operation": "{"Provision"|"Deprovision"}",
      "state": "Succeeded",
      "message": "Operation succeeded.",
      "runtimeID": "309051b6-0bac-44c8-8bae-3fc59c12bb5c"
    }
  }
}
```

The `Succeeded` status means that the provisioning/deprovisioning was successful and the cluster was created/deleted.

If you get the `InProgress` status, it means that the (de)provisioning has not yet finished. In that case, wait a few moments and check the status again.