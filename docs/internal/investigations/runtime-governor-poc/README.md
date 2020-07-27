# Runtime Governor - Proof of Concept

This document describes the scenario that will be developed to prove that [Dapr](https://dapr.io/) sidecars can be configured from a central place.

## Reasons

Kyma Runtimes must consume fewer resources. It can be achieved by delegating some of Kyma's responsibilities
to Dapr. Dapr injects sidecars into selected Pods where they fulfill their given tasks, such as state management or pub-sub.
Configuration for these sidecars can be held in one central place, from where Kyma Runtimes could fetch them and configure the
sidecars accordingly.

## PoC scenario

The scenario will present two Kyma Runtimes using [Dapr HTTP Bindings](https://github.com/dapr/docs/blob/master/reference/specs/bindings/http.md) to reach two different Redis instances.

1. Kyma Control Plane admin sets the configuration for two Kyma Runtimes.
2. Runtime Agents from both Runtimes fetch the configuration and create the Dapr Bindings.
3. Kyma Control Plane admin changes the configuration.
4. Agents fetch the new configuration, apply it, and restart the Pods.
5. Both Runtimes use the new configuration.

As it's the Proof of Concept stage, we can use some temporary solutions, such as:
- Configuration held in memory instead of a database
- Pods being restarted by the Agent instead of a more complex solution

## Reloading configuration of Dapr sidecars 

We need to configure Dapr sidecars in a Kyma Runtime from one central Kyma Control Plane. However, he Dapr sidecars have an [issue](https://github.com/dapr/dapr/issues/1172) concerning the fact that the whole Pod needs to restart for the configuration to be applied.

There are a few solutions for this problem:

### Control Plane Runtime Agent will restart every Pod with a Dapr sidecar
This solution may be the simplest, but it could add too many responsibilities for one component. A single Control Plane Runtime Agent would fetch the configuration, apply it, and then restart all Pods.

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

## Related resources

For implementation details of the Runtime Governor, see the [README.md](./governor/README.md) document.