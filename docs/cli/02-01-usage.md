---
title: Usage
---

The kcp CLI comes with a set of commands, each of which has its own specific set of flags.

For the commands and flags to work, they need to follow this syntax:

```bash
kcp {COMMAND} {FLAGS}
```

- **{COMMAND}** specifies the operation you want to perform, such as displaying runtimes.
- **{FLAGS}** specifies optional flags you can use to enrich your command.

See the example:

```bash
kcp runtimes --region westeurope
```

The CLI supports configuration file for common, global options needed for all commands. The config file will be looked up in this order:

  - `--config <PATH>` option
  - `KCPCONFIG` environment variable which contains the path
  - `$HOME/.kcp/config.yaml` (default path)

The configuration file is in YAML format and supports the following global options:
|     Option            |  Description  | Example |
|-----------------------|---------------|---------|
| `oidc-issuer-url`     | OIDC authentication server URL to use for login | `"https://accounts.sap.com"` |
| `oidc-client-id`      | OIDC client ID to use for login | `"my-client-id"` |
| `oidc-client-secret`  | OIDC client secret to use for login | `"my-client-secret"` |
| `keb-api-url`         | Kyma Environment Broker API URL to use for all commands | `"https://kyma-env-broker.kyma.local"` |
| `kubeconfig-api-url`  | OIDC Kubeconfig Service API URL, used by the [kcp kubeconfig](commands/kcp_kubeconfig.md) and [kcp taskrun](commands/kcp_taskrun.md) commands | `"https://kubeconfig-service.kyma.local"` |
| `gardener-kubeconfig` | Optional path to the corresponding Gardener project kubeconfig file which have permissions to list/get shoots. Needed by [kcp taskrun](commands/kcp_taskrun.md) command | `"path/to/gardener.kubeconfig.yaml"` |

See [the full list of commands and flags](commands/kcp.md).

|     Command        | Child commands   |  Description  | Example |
|--------------------|----------------|---------------|---------|
| [`kubeconfig`](commands/kcp_kubeconfig.md) | None | Downloads kubeconfig for given Kyma runtime. | `kcp kubeconfig -c a1fb2d35` |
| [`login`](commands/kcp_login.md) | None | Performs OIDC login required by all commands. | `kcp login` |
| [`orchestrations`](commands/kcp_orchestrations.md) | None | Displays Kyma control plane orchestrations and corresponding operations details. | `kcp orchestrations` |
| [`runtimes`](commands/kcp_runtimes) | None | Displays Kyma runtimes based on various filters. | `kcp runtimes --region westeurope` |
| [`taskrun`](commands/kcp_taskrun.md) | None | Runs generic tasks on one or more Kyma runtimes. | `kcp taskrun --target all kubectl get nodes` |
| [`upgrade`](commands/kcp_upgrade.md) | [`kyma`](commands/kcp_upgrade_kyma.md) | Performs upgrade operations on Kyma runtimes. Currently only kyma upgrade is supported. | `kcp upgrade kyma --target all` |
