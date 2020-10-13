---
title: skr upgrade kyma
---
Upgrade or reconfigure Kyma on one or more Kyma runtimes

## Synopsis

Upgrades or reconfigures Kyma on targets of SKRs.
The upgrade is performed by the Kyma Control plane within a new orchestration asynchronously, the id of which is returned by the command upon success.
The targets of SKRs are specified via the --target and --target-exclude options. At lease one --target must be specified.
The Kyma version and configurations to use for the upgrade is taken from the Kyma Control Plane during processing of the orchestration.

```bash
skr upgrade kyma --target <TARGET SPEC> ... [--target-exclude <TARGET SPEC> ...] [flags]
```

## Examples

```bash
  skr upgrade kyma --target all --schedule maintenancewindow     Upgrade Kyma on all runtimes in their next respective maintenance window hours
  skr upgrade kyma --target "account=CA.*"                       Upgrade Kyma on runtimes of all Global Accounts starting with CA
  skr upgrade kyma --target all --target-exclude "account=CA.*"  Upgrade Kyma on runtimes of all Global Accounts not starting with CA
  skr upgrade kyma --target "region=europe|eu|uk"                Upgrade Kyma on runtimes whose region belongs to Europe
```

## Options

```bash
      --dry-run                      Perform the orchestration without executing the actual upgrage operations for the runtimes. The details can be obtained using the "skr orchestrations" command
      --parallel-workers int         Number of parallel workers to use in parallel orchestration strategy. By default the amount of workers will be auto-selected on control plane server side
      --schedule string              Orchestration schedule to use. Possible values: "immediate", "maintenancewindow". By default the schedule will be auto-selected on control plane server side
      --strategy string              Orchestration strategy to use. Currently the only supported strategy is parallel (default "parallel")
  -t, --target stringArray           List of runtime target specifiers to include (the option can be specified multiple times).
                                     A target specifier is a comma separated list of the following selectors:
                                       all                 : all SKRs provisioned successfully and not deprovisioning
                                       account=<REGEXP>    : Regex pattern to match against the runtime's GlobalAccount field. E.g. CA50125541TID000000000741207136, "CA.*"
                                       subaccount=<REGEXP> : Regex pattern to match against the runtime's SubAccount field. E.g. 0d20e315-d0b4-48a2-9512-49bc8eb03cd1
                                       region=<REGEXP>     : Regex pattern to match against the shoot cluster's Region field (not SCP platform-region). E.g. "europe|eu-"
                                       runtime-id=<ID>     : Runtime ID is used to indicate a specific runtime
  -e, --target-exclude stringArray   List of runtime target specifiers to exclude (the option can be specified multiple times).
                                     A target specifier is a comma separated list of the selectors described under --target option
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

* [skr upgrade](skr_upgrade.md)	 - Perform upgrade operations on Kyma runtimes

