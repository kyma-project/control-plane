# kcp runtimes

Displays Kyma Runtimes.

## Synopsis

Displays Kyma Runtimes and their primary attributes, such as identifiers, region, or states.
The command supports filtering Runtimes based on various attributes. See the list of options for more details.

```bash
kcp runtimes [flags]
```

## Examples

```
  kcp runtimes                                           Display table overview about all Runtimes.
  kcp rt -c c-178e034 -o json                            Display all details about one Runtime identified by a Shoot name in the JSON format.
  kcp runtimes --account CA4836781TID000000000123456789  Display all Runtimes of a given global account.
  kcp runtimes -c bbc3ee7 -o custom="INSTANCE ID:instanceID,SHOOTNAME:shootName"
                                                         Display the custom fields about one Runtime identified by a Shoot name.
  kcp runtimes -o custom="INSTANCE ID:instanceID,SHOOTNAME:shootName,runtimeID:runtimeID,STATUS:{status.provisioning}"
                                                         Display all Runtimes with specific custom fields.
```

## Options

```
  -g, --account strings       Filter by global account ID. You can provide multiple values, either separated by a comma (e.g. GAID1,GAID2), or by specifying the option multiple times.
  -i, --instance-id strings   Filter by instance ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.
  -o, --output string         Output type of displayed Runtime(s). The possible values are: table, json, custom(e.g. custom=<header>:<jsonpath-field-spec>. (default "table")
  -p, --plan strings          Filter by service plan name. You can provide multiple values, either separated by a comma (e.g. azure,trial), or by specifying the option multiple times.
  -R, --region strings        Filter by provider region. You can provide multiple values, either separated by a comma (e.g. westeurope,northeurope), or by specifying the option multiple times.
  -r, --runtime-id strings    Filter by Runtime ID. You can provide multiple values, either separated by a comma (e.g. ID1,ID2), or by specifying the option multiple times.
  -c, --shoot strings         Filter by Shoot cluster name. You can provide multiple values, either separated by a comma (e.g. shoot1,shoot2), or by specifying the option multiple times.
  -S, --state strings         Filter by Runtime state. The possible values are: succeeded, failed, provisioning, deprovisioning, upgrading, suspended, all. Suspended Runtimes are filtered out unless the "all" or "suspended" values are provided. You can provide multiple values, either separated by a comma (e.g. succeeded,failed), or by specifying the option multiple times.
  -s, --subaccount strings    Filter by subaccount ID. You can provide multiple values, either separated by a comma (e.g. SAID1,SAID2), or by specifying the option multiple times.
```

## Global Options

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

* [kcp](kcp.md)	 - Day-two operations tool for Kyma Runtimes.

