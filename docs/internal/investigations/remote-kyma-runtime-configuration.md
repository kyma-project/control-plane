# Remote Kyma Runtime configuration


We need to configure Dapr sidecars on a Kyma Runtime from one central Kyma Control Plane.

The proposed solution is to create one central component that would hold configurations for all Runtimes. Each Runtime will have a new component called Control Plane Runtime Agent that will periodically fetch the configuration and apply it to a Kyma Runtime. However, the Dapr 
sidecars have an [issue](https://github.com/dapr/dapr/issues/1172) as the whole Pod needs to restart for the configuration to be applied.

## Solutions

There are a few solutions for this problem:

### Control Plane Runtime Agent will restart every Pod with a Dapr sidecar
This solution may be the simplest, but it could add too many responsibilities for one component. A single Control Plane Runtime Agent would fetch the configuration, apply it and then restart all Pods.

Pros:
- No new components in the Runtime 
- Less resource-consuming

Cons:
- Too many responsibilites for one component

### New component responsible for restarting Pods
This solution adds a new component that will be triggered by the Agent and will restart Pods mentioned in the configuration.

Pros:
- Responsibilites distributed between components

Cons:
- Another component will definitely mean more resource consumption, which we want to avoid

### Dapr contribution that would allow configuration reloading
This solution will surely be the most time-consuming. It seems like a really big change to the Dapr ecosystem and the approach to that is still being [discussed](https://github.com/dapr/dapr/issues/1172#issuecomment-610568718). We would have to create a proposal, wait for it to be accepted, and then develop the feature.

Pros:
- Configuration reload out-of-the-box

Cons:
- Tremendous amount of work needed
