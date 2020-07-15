# Remote Runtime management concept


This document describes the concept of Runtime management using [Dapr](https://dapr.io).

## Architecture

![Concept image](assets/concept.png?raw=true "Concept")

This diagram shows the future architecture of the Kyma environment.
There are two central places, Compass and Kyma Control Plane, along with Runtimes that communicate with them.

### Compass
Compass is responsible for connecting applications to Runtimes, along with controlling and monitoring them.
The System Broker will be responsible for fetching data from different types of runtimes (e.g. Cloud Foundry or customer Kubernetes cluster with service catalog installed on it).  

### Kyma Control Plane
Kyma Control Plane is responsible for managing Kyma instances. It allows you to provision Kyma Runtimes and it's also a central place where Kyma instances can fetch their configuration from. 
The Runtime Director component exposes the API for the Kyma Runtimes configuration. 

### Runtimes
As shown in the diagram, these Runtimes will be supported:
- Cloud Foundry 
- customer own Kubernetes cluster with Service Catalog installed
- managed Kyma Runtime
- standalone Kyma Runtime
Each supported Kyma instance has a dedicated Agent component that is responsible for handling the communication with the Kyma Central Plane. The Agent component fetches the configuration from the Runtime Director and applies it to the desired environment.

## Kyma Runtime configuration from the Control Plane
The main idea is to have a component in Kyma Control Plane that will be a point of communication for Agents to manage the Runtimes. This component would be the image of Runtime Director. It would also allow for configuring Runtimes from one central place.

For example, the configuration could be passed as an URL with credentials to some external service, such as logging service which is used by some of the registered Runtimes. After each Runtime fetches the new configuration, the Dapr bindings would be updated and populated to the Dapr sidecars.
