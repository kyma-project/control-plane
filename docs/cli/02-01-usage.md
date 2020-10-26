# Usage

The Kyma Control Plane (KCP) CLI comes with a set of commands, each of which has its own specific set of flags.

For the commands and flags to work, they must follow this syntax:

```bash
kcp {COMMAND} {FLAGS}
```

- **{COMMAND}** specifies the operation you want to perform, such as displaying Runtimes.
- **{FLAGS}** specifies optional flags you can use to enrich your command.

See the example:

```bash
kcp runtimes --region westeurope
```

The CLI supports configuration file for common, global options needed for all commands. The config file will be looked up in this order:

  - `--config {PATH}` option
  - `KCPCONFIG` environment variable which contains the path
  - `$HOME/.kcp/config.yaml` (default path)


See [the full list of commands, global options and flags](commands/kcp.md).

|     Command        | Child commands   |  Description  | Example |
|--------------------|----------------|---------------|---------|
| [`kubeconfig`](commands/kcp_kubeconfig.md) | None | Downloads kubeconfig for given Kyma Runtime. | `kcp kubeconfig -c a1fb2d35` |
| [`login`](commands/kcp_login.md) | None | Performs OIDC login required by all commands. | `kcp login` |
| [`orchestrations`](commands/kcp_orchestrations.md) | None | Displays KCP orchestrations and corresponding operations details. | `kcp orchestrations` |
| [`runtimes`](commands/kcp_runtimes) | None | Displays Kyma Runtimes based on various filters. | `kcp runtimes --region westeurope` |
| [`taskrun`](commands/kcp_taskrun.md) | None | Runs generic tasks on one or more Kyma Runtimes. | `kcp taskrun --target all kubectl get nodes` |
| [`upgrade`](commands/kcp_upgrade.md) | [`kyma`](commands/kcp_upgrade_kyma.md) | Performs upgrade operations on Kyma Runtimes. Currently, only Kyma upgrade is supported. | `kcp upgrade kyma --target all` |
