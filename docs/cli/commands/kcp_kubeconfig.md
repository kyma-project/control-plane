# kcp kubeconfig
Downloads the kubeconfig file for a given Kyma Runtime

## Synopsis

Downloads the kubeconfig file for a given Kyma Runtime.
The Runtime can be specified by one of the following:
  - Global account / subaccount pair with the `--account` and `--subaccount` options
  - Global account / Runtime ID pair with the `--account` and `--runtime-id` options
  - Shoot cluster name with the `--shoot` option.

By default, the kubeconfig file is saved to the current directory. The output file name can be specified using the `--output` option.

```bash
kcp kubeconfig [flags]
```

## Examples

```
  kcp kubeconfig -g GAID -s SAID -o /my/path/runtime.config  Downloads the kubeconfig file using global account ID and subaccount ID.
  kcp kubeconfig -g GAID -r RUNTIMEID                    Downloads the kubeconfig file using global account ID and Runtime ID.
  kcp kubeconfig -c c-178e034                            Downloads the kubeconfig file using a Shoot cluster name.
```

## Options

```
  -g, --account string      Global account ID of the specific Kyma Runtime.
  -o, --output string       Path to the file to save the downloaded kubeconfig to. Defaults to {CLUSTER NAME}.yaml in the current directory if not specified.
  -r, --runtime-id string   Runtime ID of the specific Kyma Runtime.
  -c, --shoot string        Shoot cluster name of the specific Kyma Runtime.
  -s, --subaccount string   Subccount ID of the specific Kyma Runtime.
```

## Global Options

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

* [kcp](kcp.md)	 - Day-two operations tool for Kyma Runtimes.

