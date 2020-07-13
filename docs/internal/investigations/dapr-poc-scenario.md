# Dapr Proof of Concept scenario

## Overview

This document describes the scenario that will be developed to prove that Dapr sidecars can be configured from a central place.

## Reasons

There is a need to make the Kyma Runtimes consume less resources. It can be achieved by delegating some of Kyma's responsibilites
to Dapr. Dapr injects sidecars into selected pods where they fulfill their given tasks, eg. state management or pub-sub.
Configuration for these sidecars can be held in one central place, from where Kyma Runtimes could fetch them and configure the
sidecars accordingly.

## Known Issues

Dapr sidecars need a restart to apply a change in configuration. We are still waiting for the decision on the Dapr side if
they will stay with that approach.

## POC Scenario

The scenario will present two Kyma Runtimes using Dapr HTTP Bindings to reach two different Redis instances.

1. Kyma Control Plane admin sets the configuration for two Kyma Runtimes
2. Runtime Agents from both runtimes fetch the configuration and create the Dapr Bindings
3. Kyma Control Plane admin changes the configuration
4. Agents fetch the new configuration, applies it and restarts the pods
5. Both Runtimes use the new configuration

As it's the Proof of Concept stage we can use some temporary solutions eg.
- Configuration held in memory instead of a DB
- Pods being restarted by the Agent instead of a more complex solution
