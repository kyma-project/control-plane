# kcp taskrun
Runs generic tasks on one or more Kyma Runtimes.

## Synopsis

Runs a command, which can be a script or a program with arbitrary arguments, on targets of Kyma Runtimes.
The specified command is executed locally. It is executed in separate subprocesses for each Runtime in parallel, where the number of parallel executions is controlled by the `--parallelism` option.

For each subprocess, the following Runtime-specific data are passed as environment variables:
  - KUBECONFIG       : Path to the kubeconfig file for the specific Runtime, unless `--no-kubeconfig` option is passed
  - GLOBALACCOUNT_ID : Global account ID of the Runtime
  - SUBACCOUNT_ID    : Subaccount ID of the Runtime
  - RUNTIME_NAME     : Shoot cluster name
  - RUNTIME_ID       : Runtime ID of the Runtime

	If all subprocesses finish successfully with the zero status code, the exit status is zero (0). If one or more subprocesses exit with a non-zero status, the command will also exit with a non-zero status.

```bash
kcp taskrun --target {TARGET SPEC} ... [--target-exclude {TARGET SPEC} ...] COMMAND [ARGS ...] [flags]
```

## Examples

```
  kcp taskrun --target all kubectl patch deployment valid-deployment -p '{"metadata":{"labels":{"my-label": "my-value"}}}'
    Execute a kubectl patch operation for all Runtimes.
  kcp taskrun --target account=CA4836781TID000000000123456789 /usr/local/bin/awesome-script.sh
    Run a maintenance script for all Runtimes of a given global account.
  kcp taskrun --target all helm upgrade -i -n kyma-system my-kyma-addon --values overrides.yaml
    Deploy a Helm chart on all Runtimes.
```

## Options

```
      --keep-kubeconfig              Option that allows you to keep downloaded kubeconfig files after execution for caching purposes.
      --kubeconfig-dir string        Directory to download Runtime kubeconfig files to. By default, it is a random-generated directory in the OS-specific default temporary directory (e.g. /tmp in Linux).
      --no-kubeconfig                Option to turn off the downloading and exposure of kubeconfig for each runtime.
      --no-prefix-output             Option to omit prefixing each output line with the Runtime name. By default, all output lines are prepended for better traceability.
  -p, --parallelism int              Number of parallel commands to execute. (default 4)
  -t, --target stringArray           List of Runtime target specifiers to include. You can specify this option multiple times.
                                     A target specifier is a comma-separated list of the following selectors:
                                       all                 : All Runtimes provisioned successfully and not deprovisioning
                                       account={REGEXP}    : Regex pattern to match against the Runtime's global account field, e.g. "CA50125541TID000000000741207136", "CA.*"
                                       subaccount={REGEXP} : Regex pattern to match against the Runtime's subaccount field, e.g. "0d20e315-d0b4-48a2-9512-49bc8eb03cd1"
                                       region={REGEXP}     : Regex pattern to match against the Runtime's provider region field, e.g. "europe|eu-"
                                       runtime-id={ID}     : Specific Runtime by Runtime ID
                                       plan={NAME}         : Name of the Runtime's service plan. The possible values are: azure, azure_lite, trial, gcp
  -e, --target-exclude stringArray   List of Runtime target specifiers to exclude. You can specify this option multiple times.
                                     A target specifier is a comma-separated list of the selectors described under the --target option.
```

## Global Options

```
      --config string                Path to the KCP CLI config file. Can also be set using the KCPCONFIG environment variable. Defaults to $HOME/.kcp/config.yaml .
      --gardener-kubeconfig string   Path to the kubeconfig file of the corresponding Gardener project which has permissions to list/get Shoots. Can also be set using the KCP_GARDENER_KUBECONFIG environment variable.
      --gardener-namespace string    Gardener namespace (project) to use. Can also be set using the KCP_GARDENER_NAMESPACE environment variable.
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

