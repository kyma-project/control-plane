# kcp orchestrations

Displays Kyma Control Plane (KCP) orchestrations.

## Synopsis

Displays KCP orchestrations and their primary attributes, such as identifiers, type, state, parameters, or Runtime operations.
The command has the following modes:
  - Without specifying an orchestration ID as an argument. In this mode, the command lists all orchestrations, or orchestrations matching the `--state` option, if provided.
  - When specifying an orchestration ID as an argument. In this mode, the command displays details about the specific orchestration.
      If the optional `--operation` flag is provided, it displays details of the specified Runtime operation within the orchestration.
  - When specifying an orchestration ID and `operations` or `ops` as arguments. In this mode, the command displays the Runtime operations for the given orchestration.
  - When specifying an orchestration ID and `cancel` as arguments. In this mode, the command cancels the orchestration and all pending Runtime operations.

```bash
kcp orchestrations [id] [ops|operations] [cancel] [flags]
```

## Examples

```
  kcp orchestrations --state inprogress                                   Display all orchestrations which are in progress.
  kcp orchestration -o custom="Orchestration ID:{.OrchestrationID},STATE:{.State},CREATED AT:{.createdAt}"
                                                                          Display all orchestations with specific custom fields.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00                  Display details about a specific orchestration.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00 --operation OID  Display details of the specified Runtime operation within the orchestration.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00 operations       Display the operations of the given orchestration.
  kcp orchestration 0c4357f5-83e0-4b72-9472-49b5cd417c00 cancel           Cancel the given orchestration.
```

## Options

```
      --operation string   Option that displays details of the specified Runtime operation when a given orchestration is selected.
  -o, --output string      Output type of displayed Runtime(s). The possible values are: table, json, custom(e.g. custom=<header>:<jsonpath-field-spec>. (default "table")
  -s, --state strings      Filter output by state. You can provide multiple values, either separated by a comma (e.g. failed,inprogress), or by specifying the option multiple times. The possible values are: canceled, canceling, failed, inprogress, pending, succeeded.
```

## Global Options

```
      --config string                Path to the KCP CLI config file. Can also be set using the KCPCONFIG environment variable. Defaults to $HOME/.kcp/config.yaml . (default "/home/i349725/.kcp/config-prod.yaml")
      --gardener-kubeconfig string   Path to the kubeconfig file of the corresponding Gardener project which has permissions to list/get Shoots. Can also be set using the KCP_GARDENER_KUBECONFIG environment variable.
      --gardener-namespace string    Gardener Namespace (project) to use. Can also be set using the KCP_GARDENER_NAMESPACE environment variable.
  -h, --help                         Option that displays help for the CLI.
      --keb-api-url string           Kyma Environment Broker API URL to use for all commands. Can also be set using the KCP_KEB_API_URL environment variable.
      --kubeconfig-api-url string    OIDC Kubeconfig Service API URL used by the kcp kubeconfig and taskrun commands. Can also be set using the KCP_KUBECONFIG_API_URL environment variable.
      --oidc-client-id string        OIDC client ID to use for login. Can also be set using the KCP_OIDC_CLIENT_ID environment variable.
      --oidc-client-secret string    OIDC client secret to use for login. Can also be set using the KCP_OIDC_CLIENT_SECRET environment variable.
      --oidc-issuer-url string       OIDC authentication server URL to use for login. Can also be set using the KCP_OIDC_ISSUER_URL environment variable.
  -v, --verbose int                  Option that turns verbose logging to stderr. Valid values are 0 (default) - 6 (maximum verbosity).
```

## See also

* [kcp](kcp.md)	 - Day-two operations tool for Kyma Runtimes.

