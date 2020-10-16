---
title: kcp
---
Day-two operations tool for Kyma Runtimes

## Synopsis

The kcp CLI (a.k.a. Kyma Control Plane CLI) is a day-two operations tool for Kyma runtimes, which allows to view and manage the runtimes in scale.
It is possible to list and observe attributes and state of each Kyma runtime and perform various operations on them, e.g. upgrading the Kyma version.
You can find the complete list of possible operations as commands below.

The CLI supports configuration file for common, global options needed for all commands. The config file will be looked up in this order:
  --config <PATH> option
  KCPCONFIG environment variable which contains the path
  $HOME/.kcp/config.yaml (default path)

The configuration file is in YAML format and supports the following global options: oidc-issuer-url, oidc-client-id, oidc-client-secret, keb-api-url, kubeconfig-api-url, gardener-kubeconfig.

## Options

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

* [kcp kubeconfig](kcp_kubeconfig.md)	 - Download kubeconfig for given Kyma runtime
* [kcp login](kcp_login.md)	 - Perform OIDC login required by all commands
* [kcp orchestrations](kcp_orchestrations.md)	 - Display Kyma control plane orchestrations
* [kcp runtimes](kcp_runtimes.md)	 - Display Kyma runtimes
* [kcp taskrun](kcp_taskrun.md)	 - Run generic tasks on one or more Kyma runtimes
* [kcp upgrade](kcp_upgrade.md)	 - Perform upgrade operations on Kyma runtimes

