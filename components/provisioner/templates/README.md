# Templates

This directory contains templates for `Shoot` and `ClusterBom` to install Kyma directly on Gardener without using the Provisioner but with similar configuration.

## Usage

To make it easier to provision Shoot cluster and install Kyma from templates, Provisioner contains small binary to do so.

To render the templates with required values, run:
```bash
go run ./cmd/template render --shoot=my-shoot --project=my-gardener-project --secret=my-azure-secret
``` 

To Provision the cluster apply generated files to the Gardener Project:
```bash
kubectl apply -f ./templates-rendered/
```

To see additional parameters that can be used with `render` command, run:
```bash
go run ./cmd/template render -h
``` 


## Development 

When default Provisioning parameters change or there are any modifications to `Shoot` resources created by the Provisioner, the `shoot.yaml` template need to be regenerated.

To regenerate the template, run:
```bash
go run ./cmd/template generate
```
