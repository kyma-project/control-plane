[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/control-plane)](https://api.reuse.software/info/github.com/kyma-project/control-plane)

# Kyma Control Plane

## Overview

Kyma Control Plane (KCP) is a central system to manage Kyma Runtimes.

For more information on KCP and its components, read the [KCP documentation](https://github.com/kyma-project/control-plane/tree/main/docs).

## Prerequisites

- [Docker](https://www.docker.com/get-started)
- [Minikube](https://github.com/kubernetes/minikube) 1.6.2
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) 1.16.3
- [Kyma CLI](https://github.com/kyma-project/cli) stable

## Dependencies

Kyma Control Plane depends on [Kyma](https://github.com/kyma-project/kyma) and [Compass](https://github.com/kyma-incubator/compass).

## Reconciler

If you want to deploy only the [Reconciler](https://github.com/kyma-incubator/reconciler) for testing purposes, run:

```bash
cd tools/reconciler
make deploy-reconciler
```