---
title: Usage
---

The skr CLI comes with a set of commands, each of which has its own specific set of flags.

For the commands and flags to work, they need to follow this syntax:

```bash
skr {COMMAND} {FLAGS}
```

- **{COMMAND}** specifies the operation you want to perform, such as displaying runtimes.
- **{FLAGS}** specifies optional flags you can use to enrich your command.

See the example:

```bash
skr runtimes --region westeurope
```

The CLI supports configuration file for common, global options needed for all commands. The config file will be looked up in this order:
  --config <PATH> option
  SKRCONFIG environment variable which contains the path
  $HOME/.skr/config.yaml (default path)

The configuration file is in YAML format and supports the following global options:
|     Option          |  Description  | Example |
|---------------------|---------------|---------|
| oidc-issuer-url     | OIDC authentication server URL to use for login | `"https://accounts.sap.com"` |
| oidc-client-id      | OIDC client ID to use for login | `"my-client-id"` |
| oidc-client-secret  | OIDC client secret to use for login | `"my-client-secret"` |
| keb-api-url         | Kyma Environment Broker API URL to use for all commands | `"https://kyma-env-broker.kyma.local"` |
| kubeconfig-api-url  | OIDC Kubeconfig Service API URL, used by the [skr kubeconfig](commands/skr_kubeconfig.md) and [skr taskrun](commands/skr_taskrun.md) commands | `"https://kubeconfig-service.kyma.local"` |
| gardener-kubeconfig | Optional path to the corresponding Gardener project kubeconfig file which have permissions to list/get shoots. Needed by [skr taskrun](commands/skr_taskrun.md) command | `"path/to/gardener.kubeconfig.yaml"` |

See [the full list of commands and flags](commands/skr.md).

|     Command        | Child commands   |  Description  | Example |
|--------------------|----------------|---------------|---------|
| [`kubeconfig`](commands/skr_kubeconfig.md) | None | Downloads kubeconfig for given Kyma runtime. | `skr kubeconfig -c a1fb2d35` |
| [`login`](commands/skr_login.md) | None | Performs OIDC login required by all commands. | `skr login` |
| [`orchestrations`](commands/skr_orchestrations.md) | None | Displays Kyma control plane orchestrations and corresponding operations details. | `skr orchestrations` |
| [`runtimes`](commands/skr_runtimes) | None | Displays Kyma runtimes based on various filters. | `skr runtimes --region westeurope` |
| [`taskrun`](commands/skr_taskrun.md) | None | Runs generic tasks on one or more Kyma runtimes. | `skr taskrun --target all kubectl get nodes` |
| [`upgrade`](commands/skr_upgrade.md) | [`kyma`](commands/skr_upgrade_kyma.md) | Performs upgrade operations on Kyma runtimes. Currently only kyma upgrade is supported. | `skr upgrade kyma --target all` |
