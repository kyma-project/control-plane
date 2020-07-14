# Remote runtime management concept

## Overview

This document describes the concept of runtime management with the usage of [Dapr](https://dapr.io).

## Architecture

![Concept image](assets/concept.png?raw=true "Concept")

The diagram above shows the future architecture of Kyma environment.
There are two central places, Compass and Kyma Control Plane along with runtimes which will communicate with them.

### Compass
It is responsible for connecting applications to runtimes, along with controlling and monitoring them.
The System Broker will be responsible for fetching data from different types of runtimes (e.g. Cloud Foundry or customer Kubernetes cluster with service catalog installed on it).  

### Kyma Control Plane
The Kyma Control Plane is responsible for managing the Kyma instances. It allows to provision Kyma Runtimes as well as it is a central place where the Kyma instances can fetch their configuration from. 
The Runtime Director will be a service that will be responsible for managing and controlling runtimes through runtime agents.

### Runtimes
The diagram presents some possible runtimes, which will be supported in the future. It includes Cloud Foundry runtime, Customer own cluster with Service Catalog installed, Kyma Managed Runtime, and Kyma Standalone Runtime. 
Each of them beside Kyma Standalone will have some implementation of Agent, that will communicate with Control Plane.

## Kyma Runtime configuration from the Control Plane
The main idea is to have a component in Kyma Control Plane (Runtime Director on image), which would be a point of communication with Runtimes through Agents to manage them. It would allow also to configure them from one central point.

The configuration could be for example an URL and credentials to some external service (e.g. logging service) which is used by some of the registered Runtimes. After each runtime fetches the new configuration, the Dapr bindings would be updated and populated to the Dapr sidecars.
