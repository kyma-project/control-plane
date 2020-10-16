---
title: kcp taskrun
---
Run generic tasks on one or more Kyma runtimes

## Synopsis

Runs a command (which can be a script or a program with arbitrary arguments) on targets of Kyma runtimes.
The specified command will be executed locally in parallel in separate subprocesses for each runtime, where the number of parallel executions are controlled by the --parallelism option.

For each subprocess, the following runtime specific data are passed as environment variables:
  KUBECONFIG       : Path to the kubeconfig file for the specific runtimme
  GLOBALACCOUNT_ID : Global Account ID of the runtime
  SUBACCOUNT_ID    : Subaccount ID of the runtime
  RUNTIME_NAME     : Shoot cluster name
  RUNTIME_ID       : Runtime ID of the runtime

The exit status is zero (0) if all subprocesses exit successfully with zero status code. If one or more subprocesses exit with non-zero status, the command will also exit with non-zero status.

```bash
kcp taskrun --target <TARGET SPEC> ... [--target-exclude <TARGET SPEC> ...] COMMAND [ARGS ...] [flags]
```

## Examples

```bash
  kcp taskrun --target all kubectl patch deployment valid-deployment -p '{"spec":{"template":{"spec":{"containers":[{"name":"kubernetes-serve-hostname","image":"new image"}]}}}}'
    Execute a kubectl patch operation for all runtimes
  kcp taskrun --target account=CA4836781TID000000000123456789 /usr/local/bin/awesome-script.sh
    Run a maintenance script for all runtimes of a given Global Account
  kcp taskrun --target all helm upgrade -i -n kyma-system my-kyma-addon --values overrides.yaml
    Deploy or a helm chart on all runtimes
```

## Options

```bash
      --keep                         Keep downloaded kubeconfigs after execution for caching purpose
      --kubeconfig-dir string        Directory to download runtime kubeconfigs to. By default it is a random-generated directory in the OS specific default temporary directory (e.g. /tmp in Linux)
  -p, --parallelism int              Number of parallel commands to execute (default 8)
  -t, --target stringArray           List of runtime target specifiers to include (the option can be specified multiple times).
                                     A target specifier is a comma separated list of the following selectors:
                                       all                 : all runtimes provisioned successfully and not deprovisioning
                                       account=<REGEXP>    : Regex pattern to match against the runtime's GlobalAccount field. E.g. CA50125541TID000000000741207136, "CA.*"
                                       subaccount=<REGEXP> : Regex pattern to match against the runtime's SubAccount field. E.g. 0d20e315-d0b4-48a2-9512-49bc8eb03cd1
                                       region=<REGEXP>     : Regex pattern to match against the shoot cluster's Region field (not SCP platform-region). E.g. "europe|eu-"
                                       runtime-id=<ID>     : Runtime ID is used to indicate a specific runtime
  -e, --target-exclude stringArray   List of runtime target specifiers to exclude (the option can be specified multiple times).
                                     A target specifier is a comma separated list of the selectors described under --target option
```

## Options inherited from parent commands

```bash
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

