# kcp

Day-two operations tool for Kyma Runtimes.

## Synopsis

KCP CLI (Kyma Control Plane CLI) is a day-two operations tool for Kyma Runtimes, which allows you to view and manage the Runtimes in scale.
It is possible to list and observe attributes and state of each Kyma Runtime, and perform various operations on them, such as upgrading the Kyma version.
You can find the complete list of possible operations as commands below.

The CLI supports configuration file for common (global) options needed for all commands. The config file will be looked up in this order:
  - `--config {PATH}` option
  - KCPCONFIG environment variable which contains the path
  - $HOME/.kcp/config.yaml (default path).

The configuration file is in YAML format and supports the following global options: oidc-issuer-url, oidc-client-id, oidc-client-secret, keb-api-url, kubeconfig-api-url, gardener-kubeconfig.
See the **Global Options** section of each command for the description of these options.

## Options

```
      --config string                Path to the KCP CLI config file. Can also be set using the KCPCONFIG environment variable. Defaults to $HOME/.kcp/config.yaml .
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

* [kcp completion](kcp_completion.md)	 - Generates completion script
* [kcp kubeconfig](kcp_kubeconfig.md)	 - Downloads the kubeconfig file for a given Kyma Runtime
* [kcp login](kcp_login.md)	 - Performs OIDC login required by all commands.
* [kcp orchestrations](kcp_orchestrations.md)	 - Displays Kyma Control Plane (KCP) orchestrations.
* [kcp runtimes](kcp_runtimes.md)	 - Displays Kyma Runtimes.
* [kcp taskrun](kcp_taskrun.md)	 - Runs generic tasks on one or more Kyma Runtimes.
* [kcp upgrade](kcp_upgrade.md)	 - Performs upgrade operations on Kyma Runtimes.

