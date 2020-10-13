---
title: skr runtimes
---
Display Kyma runtimes

## Synopsis

Display Kyma runtimes and their primary attributes, such as identifiers, region, states, etc.
The command supports filtering runtimes based on various attributes, see the list of options below.

```bash
skr runtimes [flags]
```

## Examples

```bash
  skr runtimes                                           Display table overview about all runtimes
  skr rt -c c-178e034 -o json                            Display all details about one runtime identified by Shoot name in JSON format
  skr runtimes --account CA4836781TID000000000123456789  Display all runtimes of a given Global Account
```

## Options

```bash
  -g, --account strings      Filter by Global Account ID. Multiple values can be provided, either separated as a comma (e.g GAID1,GAID2), or by specifying the option multiple times
  -o, --output string        Output type of displayed runtime(s). Possible values: table, json, yaml (default "table")
  -r, --region strings       Filter by Region. Multiple values can be provided, either separated as a comma (e.g cf-eu10,cf-us10), or by specifying the option multiple times
  -i, --runtime-id strings   Filter by Runtime ID. Multiple values can be provided, either separated as a comma (e.g ID1,ID2), or by specifying the option multiple times
  -c, --shoot strings        Filter by Shoot cluster name. Multiple values can be provided, either separated as a comma (e.g shoot1,shoot2), or by specifying the option multiple times
  -s, --subaccount strings   Filter by Subaccount ID. Multiple values can be provided, either separated as a comma (e.g SAID1,SAID2), or by specifying the option multiple times
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

