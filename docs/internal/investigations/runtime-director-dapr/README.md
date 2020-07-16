# Runtime Director - Proof of concept

This directory contains source code and Helm chart for Kyma Control Plane Runtime Director.

Runtime Director 

## Overview

Runtime Director exposes simple REST API which returns Runtime configurations.
The application periodically reloads configuration from a ConfigMap `runtime-director-config` in `kcp-poc` namespace. The configuration file has the following form:

```yaml
runtimes:
  - id: "1"
    data: # arbitrary data
      foo: bar
  - id: "2"
    data: # arbitrary data
      bar: baz
```

The application reloads configuration every 10 seconds. 

## API

The Application exposes the following endpoints:

- `GET /runtimes`
   
    Returns configuration for all Runtimes.  
    
    Possible error codes: `500`
    
    Example response:
    ```json
    [{"id": "1", "data": {"foo":  "bar"} }, {"id": "2", "data": {"bar":  "baz"}}]    
    ```

- `GET /runtimes/{runtimeId}`
   
    Returns configuration for a given Runtime.  
    
    Possible error codes: `404`, `500`
    
    Example response:
    ```json
    {"id": "1", "data": {"foo":  "bar"} }    
    ```
  
  
## Development  

### Build

To build source code, navigate to the `./component` directory and run:

```bash
make build-image
```

### Run on local machine

To run the app without building Docker image, navigate to the `./component` directory and execute the following command:

```bash
go run cmd/main.go
```

The Runtime Director listens on `127.0.0.1:3001`, reloading file from `hack/config.yaml` path.

## Installation

To install Runtime Director on Kubernetes cluster using Helm 3 in `kcp-poc` namespace, run the following command:

```bash
LOCAL_ENV=false ./deploy.sh
```

The script will add proper an `runtime-director.{DOMAIN}` entry to `/etc/hosts`.

## Configuration

You can use the following environmental variables while running the `deploy.sh` script:
 - `LOCAL_ENV` - Does the script run on local environment with Minikube (default: `true`)
 - `DOMAIN` - Used domain on cluster (default: `kyma.local`)
 - `ISTIO_GATEWAY_NAME` - Istio Gateway name (default: `compass-istio-gateway`)
 - `ISTIO_GATEWAY_NAMESPACE` - Istio Gateway namespace (default: `compass-system`)

For example, to set a different domain and deploy on Kubernetes cluster, and install run the script with the following way:

```bash
LOCAL_ENV=false DOMAIN=foo.bar ./deploy.sh
```

## Verify

To verify if the Runtime Director works properly, execute the following command, which returns all Runtime configurations from the mounted ConfigMap].

```bash
curl https://runtime-director.${DOMAIN}/runtimes
```
