# kcp completion

Generates completion script

## Synopsis

Generates completion script for Bash, Zsh, Fish, or Powershell.
### Bash

To load completions for Bash, run:
`$ source <(kcp completion bash)`

To load completions for each session, execute once:

- Linux:
`$ kcp completion bash > /etc/bash_completion.d/kcp`

- MacOS:
`$ kcp completion bash > /usr/local/etc/bash_completion.d/kcp`

### Zsh

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:
`$ echo "autoload -U compinit; compinit" >> ~/.zshrc`

To load completions for each session, execute once:
`$ kcp completion zsh > "${fpath[1]}/_kcp"`

You will need to start a new shell for this setup to take effect.

### Fish

To load completions for Fish, run:
`$ kcp completion fish | source`

To load completions for each session, execute once:
`$ kcp completion fish > ~/.config/fish/completions/kcp.fish`

### Powershell

`PS> kcp completion powershell | Out-String | Invoke-Expression`

To load completions for every new session, run:
`PS> kcp completion powershell > kcp.ps1`

Source this file from your Powershell profile.


```bash
kcp completion [bash|zsh|fish|powershell]
```

## Examples

```
kcp completion bash                            Display completions in bash.
```

## Options

```
      --o string   autocompletion file (default "/home/i349725/kcp_completion")
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

