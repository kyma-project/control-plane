# Dapr Proof of Concept scenario

This document describes the scenario that will be developed to prove that [Dapr](https://dapr.io/) sidecars can be configured from a central place.

## Reasons

Kyma Runtimes must consume fewer resources. It can be achieved by delegating some of Kyma's responsibilities
to Dapr. Dapr injects sidecars into selected Pods where they fulfill their given tasks, such as state management or pub-sub.
Configuration for these sidecars can be held in one central place, from where Kyma Runtimes could fetch them and configure the
sidecars accordingly.

## Known issues

Dapr sidecars need a restart to apply changes in configuration. We are still waiting for the decision on the Dapr side if
they will stay with that approach.

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
