# kcp login
Performs OIDC login required by all commands.

## Synopsis

Initiates OIDC login to obtain the ID token which is required by all CLI commands.
By default, without any options, the OIDC authorization code flow is executed. It prompts the user to navigate to a local address in the browser and get redirected to the OIDC Authentication Server login page.
Service accounts can execute the resource owner credentials flow by specifying the `--username` and `--password` options.

```bash
kcp login [flags]
```

## Options

```
  -p, --password string   Password to use for the resource owner credentials flow.
  -u, --username string   Username to use for the resource owner credentials flow.
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

