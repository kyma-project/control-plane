# Kyma Environment Broker architecture

The diagram and steps describe the Kyma Environment Broker (KEB) workflow and the roles of specific components in this process:

![KEB diagram](./assets/keb-arch.svg)

1. The user sends a request to create a new cluster with [Kyma Runtime](https://github.com/kyma-incubator/compass/blob/master/docs/compass/02-01-components.md#kyma-runtime).

2. KEB proxies the request to create a new cluster to the Runtime Provisioner component.

3. Provisioner creates a new cluster.

4. KEB creates a cluster configuration in the Reconciler (except preview plan).

5. Reconciler installs Kyma (except preview plan). 

6. KEB creates Kyma resource (only for preview plan).

7. Lifecycle Manager manages Kyma modules (only for preview plan).

