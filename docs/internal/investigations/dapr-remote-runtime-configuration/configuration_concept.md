# Passing configuration to runtimes concept

## Overview

This document describes the concept of passing a configuration to runtimes from the central point with the usage of [DAPR](https://dapr.io).

![Concept image](assets/concept.png?raw=true "Concept")

## Idea

The main idea is to have a component in Kyma Control Plane (Runtime Director on image), which would be a point of communication with Runtimes through Agents and provide a configuration for them. The configuration would be for example an URL and credentials to some external service (e.g. logging service). After each runtime fetch the new configuration, the DAPR bindings should be updated and populated to the DAPR sidecars.

## Example

Let's consider integration between Redis services and Kyma Runtimes. Assume that we have two Redis services on the external cluster/managed service and we want to provide a configuration on the Kyma Runtimes to allow them to smoothly communicate with those Redis instances.

1. Kyma Control Plane and Runtimes are configured to use Redis service no. 1.
2. Kyma Control Plane administrator changes configuration to use Redis service no. 2.
3. Agents from the Runtimes fetches the new configuration and updates DAPR bindings.
4. DAPR sidecars use a new configuration.
5. Applications connected to runtime can reach Redis service through the localhost, by using the DAPR sidecar.

## Drawbacks so far

- The DAPR sidecars do not reload component config when applied, to do so whole pod has to be restarted (https://github.com/dapr/dapr/issues/1172).
