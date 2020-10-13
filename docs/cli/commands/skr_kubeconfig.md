---
title: skr kubeconfig
---
Download kubeconfig for given Kyma runtime

## Synopsis

Downloads kubeconfig for given Kyma runtime.
The runtime can be specified by either of the following:
  - Global Account / Subaccount pair with the --account and --subaccount options
  - Global Account / Runtime ID pair with the --account and --runtime-id options
  - Shoot cluster name with the --shoot option

By default the kubeconfig is saved to the current directory. The output file name can be specified using the --output option.

```bash
skr kubeconfig [flags]
```

## Examples

```bash
  skr kubeconfig -g GAID -s SAID -o /my/path/skr.config  Download kubeconfig using Global Account ID and Subaccount ID
  skr kubeconfig -g GAID -r RUNTIMEID                    Download kubeconfig using Global Account ID and Runtime ID
  skr kubeconfig -c c-178e034                            Download kubeconfig using Shoot cluster name
```

## Options

```bash
  -g, --account string      Global Account ID of the specific Kyma Runtime
  -o, --output string       Path to the file to save the downloaded kubeconfig to. Defaults to <CLUSTER NAME>.yaml in the current directory if not specified.
  -r, --runtime-id string   Runtime ID of the specific Kyma Runtime
  -c, --shoot string        Shoot cluster name of the specific Kyma Runtime
  -s, --subaccount string   Subccount ID of the specific Kyma Runtime
```

## Options inherited from parent commands

```bash
      --config string                Path to the skr CLI config file. Can also be set via the SKRCONFIG environment variable. Defaults to $HOME/.skr/config.yaml
      --gardener-kubeconfig string   Path to the corresponding Gardener project kubeconfig file which have permissions to list/get shoots. Can also be set via the SKR_GARDENER_KUBECONFIG environment variable
  -h, --help                         Displays help for the CLI
      --keb-api-url string           Kyma Environment Broker API URL to use for all commands. Can also be set via the SKR_KEB_API_URL environment variable
      --kubeconfig-api-url string    OIDC Kubeconfig Service API URL, used by the skr kubeconfig and taskrun commands. Can also be set via the SKR_KUBECONFIG_API_URL environment variable
      --oidc-client-id string        OIDC client ID to use for login. Can also be set via the SKR_OIDC_CLIENT_ID environment variable
      --oidc-client-secret string    OIDC client Secret to use for login. Can also be set via the SKR_OIDC_CLIENT_SECRET environment variable
      --oidc-issuer-url string       OIDC authentication server URL to use for login. Can also be set the SKR_OIDC_ISSUER_URL environment variable
  -v, --verbose int                  Turn on verbose logging to stderr. Valid values: 0 (default) - 3 (maximum verbosity)
```

## See also

* [skr](skr.md)	 - Day-two operations tool for SAP CP Kyma Runtimes (SKRs)

