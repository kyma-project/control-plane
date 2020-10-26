# kcp orchestrations
Displays Kyma Control Plane (KCP) orchestrations.

## Synopsis

Display KCP orchestrations and their primary attributes, such as identifiers, type, state, parameters, or Runtime operations.
The commands has two modes:
  - Without specifying an orchestration ID as an argument, the command lists all orchestrations, or orchestrations matching the `--state` option, if provided.
  - When specifying an orchestration ID as an argument, the command displays details about the specific orchestration.
     If the optional `--operation` is provided, it displays details of the specified Runtime operation within the orchestration.

```bash
kcp orchestrations [id] [flags]
```

## Examples

```
  kcp orchestrations --state inprogress                                   Display all orchestrations which are in progress.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00                  Display details about a specific orchestration.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00 --operation OID  Display details of the specified Runtime operation within the orchestration.
```

## Options

```
      --operation string   Display details of the specified Runtime operation when a given orchestration is selected.
  -o, --output string      Output type of displayed Runtime(s). The possible values are: table, json. (default "table")
  -s, --state string       Filter output by state. The possible values are: pending, inprogress, succeeded, failed.
```

## Global and Inherited Options

```
      --config string                Path to the KCP CLI config file. Can also be set using the KCPCONFIG environment variable. Defaults to $HOME/.kcp/config.yaml .
      --gardener-kubeconfig string   Path to the kubeconfig file of the corresponding Gardener project which has permissions to list/get Shoots. Can also be set using the KCP_GARDENER_KUBECONFIG environment variable.
  -h, --help                         Option that displays help for the CLI.
      --keb-api-url string           Kyma Environment Broker API URL to use for all commands. Can also be set using the KCP_KEB_API_URL environment variable.
      --kubeconfig-api-url string    OIDC Kubeconfig Service API URL used by the kcp kubeconfig and taskrun commands. Can also be set using the KCP_KUBECONFIG_API_URL environment variable.
      --oidc-client-id string        OIDC client ID to use for login. Can also be set using the KCP_OIDC_CLIENT_ID environment variable.
      --oidc-client-secret string    OIDC client secret to use for login. Can also be set using the KCP_OIDC_CLIENT_SECRET environment variable.
      --oidc-issuer-url string       OIDC authentication server URL to use for login. Can also be set using the KCP_OIDC_ISSUER_URL environment variable.
  -v, --verbose int                  Option that turns verbose logging to stderr. Valid values are 0 (default) - 3 (maximum verbosity).
```

## See also

* [kcp](kcp.md)	 - Day-two operations tool for Kyma Runtimes

