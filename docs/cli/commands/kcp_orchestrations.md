---
title: kcp orchestrations
---
Display Kyma control plane orchestrations

## Synopsis

Display Kyma control plane orchestrations and their primary attributes, such as identifiers, type, state, parameters, runtime operations.
The commands has two modes:
  1. Without specifying an orchestration id as an argument, it will list all orchestrations, or orchestrations matching the --state if supplied.
  2. When specifying an orchestration id as an argument, it will display details about the specific orchestration.
     If the optional --operation is given, it will display details of the specified runtime operation within the orchestration.

```bash
kcp orchestrations [id] [flags]
```

## Examples

```bash
  kcp orchestrations --state inprogress                                   Display all orchestrations which are in progress
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00                  Display details about a specific orchestration
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00 --operation OID  Display details of the specified runtime operation within the orchestration
```

## Options

```bash
      --operation string   Display details of the specified runtime operation when a given orchestration is selected
  -o, --output string      Output type of displayed runtime(s). Possible values: table, json, yaml (default "table")
  -s, --state string       Filter output by state. Possible values: pending, inprogress, succeeded, failed
```

## Options inherited from parent commands

```bash
      --config string                Path to the kcp CLI config file. Can also be set via the KCPCONFIG environment variable. Defaults to $HOME/.kcp/config.yaml
      --gardener-kubeconfig string   Path to the corresponding Gardener project kubeconfig file which have permissions to list/get shoots. Can also be set via the KCP_GARDENER_KUBECONFIG environment variable
  -h, --help                         Displays help for the CLI
      --keb-api-url string           Kyma Environment Broker API URL to use for all commands. Can also be set via the KCP_KEB_API_URL environment variable
      --kubeconfig-api-url string    OIDC Kubeconfig Service API URL, used by the kcp kubeconfig and taskrun commands. Can also be set via the KCP_KUBECONFIG_API_URL environment variable
      --oidc-client-id string        OIDC client ID to use for login. Can also be set via the KCP_OIDC_CLIENT_ID environment variable
      --oidc-client-secret string    OIDC client Secret to use for login. Can also be set via the KCP_OIDC_CLIENT_SECRET environment variable
      --oidc-issuer-url string       OIDC authentication server URL to use for login. Can also be set the KCP_OIDC_ISSUER_URL environment variable
  -v, --verbose int                  Turn on verbose logging to stderr. Valid values: 0 (default) - 3 (maximum verbosity)
```

## See also

* [kcp](kcp.md)	 - Day-two operations tool for Kyma Runtimes

