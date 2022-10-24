# Kyma Control Plane

## Overview

Kyma Control Plane (KCP) is a central system to manage Kyma Runtimes.

For more information on KCP and its components, read the [documentation](https://github.com/kyma-project/control-plane/tree/main/docs).

## Prerequisites

- [Docker](https://www.docker.com/get-started)
- [Minikube](https://github.com/kubernetes/minikube) 1.6.2
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) 1.16.3
- [Kyma CLI](https://github.com/kyma-project/cli) stable

## Installation

### Dependencies

Kyma Control Plane depends on [Kyma](https://github.com/kyma-project/kyma) and [Compass](https://github.com/kyma-incubator/compass).
For local development and CI integration jobs, fixed Kyma and Compass versions are used. To change Kyma or Compass version, see the [`README.md`](./installation/resources/README.md) in the `installation/resources` directory. 

### Local installation with Kyma

To install Kyma Control Plane with the minimal Kyma installation, Compass and Kyma Control Plane, run this script:
```bash
./installation/cmd/run.sh
```

You can also specify Kyma version, such as 1.6 or newer:
```bash
./installation/cmd/run.sh {version}
```

## Reconciler

If you want to deploy only the [Reconciler](https://github.com/kyma-incubator/reconciler) for testing purposes, run:

```bash
cd tools/reconciler
make deploy-reconciler
```