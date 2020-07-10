# Remote Kyma Runtime configuration

## Introduction

There is a need for configuring Dapr sidecars on a Kyma Runtime from one central Control Plane.

The proposed solution is to create one central component that would hold configurations for all the Runtimes. The Runtime will have a new component, a Control Plane Runtime Agent, that will periodically fetch the configuration and apply it on the Kyma Runtime. However, the Dapr 
sidecar have a shortcoming. As described [here](https://github.com/dapr/dapr/issues/1172), for the configuration to be applied
the whole pod needs to restart.

## Solutions

There are a few solutions for this problem:

**Control Plane Runtime Agent will restart every pod with a dapr sidecar**
This solution may be the simplest, but it could add to many responsibilities for one component, it would fetch the configuration, apply it and then restart all the pods
Pros:
- No new components on the Runtime, less resource consuming

Cons:
- Too many responsibilites for one component

**New component responsible for restarting the pods**
This solution adds a new component that will be triggered by the Agent and will restart the pods mentioned in the configuration.
Pros:
- Responsibilites distributed between components

Cons:
- Another component will definitely mean more resource consumption, which we want to avoid

**Dapr contribution that would allow configuration reloading**
This solution will surely need the most amount work. It seems like a really big change to the Dapr ecosystem and the approach for that is still being [discussed](https://github.com/dapr/dapr/issues/1172#issuecomment-610568718). We would have to create a proposal, wait for it to be accepted and then develop the feature.
Pros:
- Configuration reload out of the box

Cons:
- Tremendous amount of work needed