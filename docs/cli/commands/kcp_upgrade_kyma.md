# kcp upgrade kyma

Upgrades or reconfigures Kyma on one or more Kyma Runtimes.

## Synopsis

Upgrades or reconfigures Kyma on targets of Runtimes.
The upgrade is performed by Kyma Control Plane (KCP) within a new orchestration asynchronously. The ID of the orchestration is returned by the command upon success.
The targets of Runtimes are specified via the `--target` and `--target-exclude` options. At least one `--target` must be specified.
The Kyma version and configurations to use for the upgrade are taken from Kyma Control Plane during the processing of the orchestration.

```bash
kcp upgrade kyma --target {TARGET SPEC} ... [--target-exclude {TARGET SPEC} ...] [flags]
```

## Examples

```
  kcp upgrade kyma --target all --schedule maintenancewindow     Upgrade Kyma on all Runtimes in their next respective maintenance window hours.
  kcp upgrade kyma --target "account=CA.*"                       Upgrade Kyma on Runtimes of all global accounts starting with CA.
  kcp upgrade kyma --target all --target-exclude "account=CA.*"  Upgrade Kyma on Runtimes of all global accounts not starting with CA.
  kcp upgrade kyma --target "region=europe|eu|uk"                Upgrade Kyma on Runtimes whose region belongs to Europe.
```

## Options

```
      --dry-run                      Perform the orchestration without executing the actual upgrage operations for the Runtimes. The details can be obtained using the "kcp orchestrations" command.
      --parallel-workers int         Number of parallel workers to use in parallel orchestration strategy. By default the amount of workers will be auto-selected on control plane server side.
      --schedule string              Orchestration schedule to use. Possible values: "immediate", "maintenancewindow". By default the schedule will be auto-selected on control plane server side.
      --strategy string              Orchestration strategy to use. (default "parallel")
  -t, --target stringArray           List of Runtime target specifiers to include. You can specify this option multiple times.
                                     A target specifier is a comma-separated list of the following selectors:
                                       all                 : All Runtimes provisioned successfully and not deprovisioning
                                       account={REGEXP}    : Regex pattern to match against the Runtime's global account field, e.g. "CA50125541TID000000000741207136", "CA.*"
                                       subaccount={REGEXP} : Regex pattern to match against the Runtime's subaccount field, e.g. "0d20e315-d0b4-48a2-9512-49bc8eb03cd1"
                                       region={REGEXP}     : Regex pattern to match against the Runtime's provider region field, e.g. "europe|eu-"
                                       runtime-id={ID}     : Specific Runtime by Runtime ID
                                       plan={NAME}         : Name of the Runtime's service plan. The possible values are: azure, azure_lite, trial, gcp
                                       shoot={NAME}        : Specific Runtime by Shoot cluster name
  -e, --target-exclude stringArray   List of Runtime target specifiers to exclude. You can specify this option multiple times.
                                     A target specifier is a comma-separated list of the selectors described under the --target option.
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

* [kcp upgrade](kcp_upgrade.md)	 - Performs upgrade operations on Kyma Runtimes.

