---
title: Overview
type: Overview
---

Runtime Provisioner is a Kyma Control Plane component responsible for provisioning, installing, and deprovisioning clusters. When provisioning a cluster, you can choose whether to provision a cluster with Kyma (Kyma Runtime), by providing **kymaConfig**, or not. Runtime Provisioner is registered in Compass in Director as an Integration System.

It allows you to provision the clusters in the following ways:
- [through Gardener](#tutorials-provision-clusters-through-gardener) on:
    * GCP
    * Microsoft Azure
    * Amazon Web Services (AWS).

During the operation of provisioning, you can pass a list of Kyma components you want installed on the provisioned Runtime with their custom configuration, as well as a custom Runtime configuration. To install a customized version of a given component, you can also provide an [external URL as the installation source](/root/kyma#configuration-install-components-from-user-defined-ur-ls) for the component. See the [provisioning tutorial](#tutorials-provision-clusters-through-gardener) for more details.

Note that the operations of provisioning and deprovisioning are asynchronous. The operation of provisioning returns the Runtime Operation Status containing the Runtime ID and the operation ID. The operation of deprovisioning returns the operation ID. You can use the operation ID to [check the Runtime Operation Status](#tutorials-check-runtime-operation-status) and the Runtime ID to [check the Runtime Status](#tutorials-check-runtime-status).

Runtime Provisioner also provides extensions that let you leverage Gardener [DNS](https://github.com/gardener/external-dns-management) and [certificate management](https://github.com/gardener/cert-management). See the respective documentation for more details.

Runtime Provisioner exposes an API to manage cluster provisioning, installation, and deprovisioning.

Find the specification of the API [here](https://github.com/kyma-project/control-plane/blob/main/components/provisioner/pkg/gqlschema/schema.graphql).

To access the Runtime Provisioner, forward the port that the GraphQL Server is listening on:

```bash
kubectl -n kcp-system port-forward svc/kcp-provisioner 3000:3000
```

When making a call to the Runtime Provisioner, make sure to attach a tenant header to the request.
