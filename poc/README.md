# Runtime Director - Proof of concept

This directory contains source code and Helm chart for Kyma Control Plane Runtime Director.


### Build

To build source code, navigate to `./component` directory and run:

```bash
docker build -t {your-username}/kcp-runtime-director:v1 .
```

### Install

To install it on Minikube cluster using Helm 3 in `kcp-poc` namespace, run the following command:

```bash
./deploy.sh
```

The script will add proper an `runtime-director.{DOMAIN}` entry to `/etc/hosts`.

### Configuration

You can use following environmental variables while running the `deploy.sh` script:
 - `DOMAIN` - Used domain on cluster (default: `kyma.local`)
 - `ISTIO_GATEWAY_NAME` - Istio Gateway name (default: `compass-istio-gateway`)
 - `ISTIO_GATEWAY_NAMESPACE` - Istio Gateway namespace (default: `compass-system`)

For example, to set a different domain, run the script with the following command:

```bash
DOMAIN=foo.bar ./deploy.sh
```

### Test it

```bash
curl https://runtime-director.kyma.local/runtimes
```

```bash
curl https://runtime-director.kyma.local/runtimes/1
```
