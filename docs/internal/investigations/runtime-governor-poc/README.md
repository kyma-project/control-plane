# Runtime Governor - Proof of concept

This directory contains the source code and Helm chart for Kyma Control Plane Runtime Governor.

## Overview

Runtime Governor exposes a simple REST API that returns Runtime configurations.
The application periodically reloads configuration from the `runtime-governor-config` ConfigMap in the `kcp-poc` Namespace. The configuration file has the following form:

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

## Details 

### API

The application exposes the following endpoints:

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

To build the source code, navigate to the `./component` directory and run:

```bash
make build-image
```

### Run on a local machine

To run the application without building a Docker image, navigate to the `./component` directory and execute the following command:

```bash
go run cmd/main.go
```

The Runtime Governor listens on the `127.0.0.1:3001` port and reloads the configuration file from the `hack/config.yaml` path.

## Installation

To install Runtime Governor on a Kubernetes cluster using Helm 3 in the `kcp-poc` Namespace, run the following command:

```bash
LOCAL_ENV=false ./deploy.sh
```

The script will add a proper `runtime-governor.{DOMAIN}` entry to `/etc/hosts`.

## Configuration

You can use the following environment variables while running the `deploy.sh` script:
 - `LOCAL_ENV` - a variable that specifies whether the script runs on a local environment with Minikube (default value: `true`)
 - `DOMAIN` - a domain used on a cluster (default value: `kyma.local`)
 - `ISTIO_GATEWAY_NAME` - Istio Gateway name (default value: `compass-istio-gateway`)
 - `ISTIO_GATEWAY_NAMESPACE` - Istio Gateway Namespace (default value: `compass-system`)

For example, to set a different domain and install Runtime Governor on a Kubernetes cluster, run the script in such a way:

```bash
LOCAL_ENV=false DOMAIN=foo.bar ./deploy.sh
```

## Testing

To verify if Runtime Governor works properly, execute the following command, which returns all Runtime configurations from the mounted ConfigMap:

```bash
curl https://runtime-governor.${DOMAIN}/runtimes
```
