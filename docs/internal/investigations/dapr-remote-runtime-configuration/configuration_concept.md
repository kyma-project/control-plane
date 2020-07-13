# Remote runtimes configuration concept 

## Overview

This document describes the concept of passing a configuration to runtimes from the central point with the usage of [DAPR](https://dapr.io).

![Concept image](assets/concept.png?raw=true "Concept")

## Idea

The main idea is to have a component in Kyma Control Plane (Runtime Director on image), which would communicate with selected Runtimes through Agent and provide a configuration for them. The configuration would be for example an URL and credentials to some external service (e.g. logging service). After sending it to the Runtimes, each of them would have to populate it to desired services and then use a DAPR binding to communicate with them.

## Proof of concept

For the POC scenario lets consider integration between Redis services and SKRs. Assume that we have two Redis services on the external cluster/managed service and we want to provide a configuration on the SKRs to allow them to smoothly communicate with those Redis instances.

## Drawbacks so far

- The DAPR sidecars do not reload component config when applied, to do so whole pod has to be restarted (https://github.com/dapr/dapr/issues/1172).
