---
title: kcp login
---
Perform OIDC login required by all commands

## Synopsis

Initiates OIDC login to obtain ID token, which is required by all CLI commands.
By default without any options, the OIDC authorization code flow is executed, which prompts the user to navigate to a local address in the browser and get redirected to the OIDC Authentication Server login page.
Service accounts can execute the resource owner credentials flow by specifying the --username and --password options.

```bash
kcp login [flags]
```

## Options

```
  -p, --password string   Password to use for resource owner credentials flow
  -u, --username string   Username to use for resource owner credentials flow
```

## Options inherited from parent commands

```
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

* [kcp](kcp.md)	 - Day-two operations tool for Kyma Runtimes

