# Service description

Kyma Environment Broker (KEB) is compatible with the [Open Service Broker API](https://www.openservicebrokerapi.org/) (OSBA) specification. It provides a ServiceClass that provisions Kyma Runtime on a cluster.

## Service plans

The supported plans are as follows:

| Plan name | Description |
|-----------|-------------|
| `azure` | Installs Kyma Runtime on the Azure cluster. |
| `azure_lite` | Installs Kyma Runtime Lite on the Azure cluster. |
| `azure_trial` | Installs Kyma Runtime Trial on the Azure cluster. |
| `gcp` | Installs Kyma Runtime on the GCP cluster. |
| `gcp_trial` | Installs Kyma Runtime Trial on the GCP cluster. |

## Provisioning parameters

These are the provisioning parameters can be divided into two types:

- Compliant to all providers:

    | Parameter Name | Type | Description | Required | Default value |
    |----------------|-------|-------------|:----------:|---------------|
    | name | string | Specifies the name of the cluster. | Yes | None |
    | nodeCount | int | Specifies the number of Nodes in a cluster. | No | `3` |
    | volumeSizeGb | int | Specifies the size of the root volume. | No | `50` |
    | zones | string | Defines the list of zones in which the Runtime Provisioner creates the cluster. | No | `["1", "2", "3"]` |
    | purpose | string | Defines the purpose of the created cluster. The possible values are: `development`, `evaluation`, `production`, `testing`. | No | `development` |
    | maxSurge | int | Specifies the maximum number of virtual machines that are created during an update. | No | `4` |
    | maxUnavailable | int | Specifies the maximum number of VMs that can be unavailable during an update. | No | `1` |
    | components | array | Defines optional components that are installed in Kyma Runtime. The possible values are `kiali` and `tracing`. | No | [] |
    | kymaVersion | string | Provides Kyma version on demand. | No | None |

- Provider specific:

<div tabs name="plans" group="plans">
  <details>
    <summary label="plan">
    Azure
    </summary>
    | Parameter Name | Type | Description | Required | Default value |
    |----------------|-------|-------------|:----------:|---------------|
    | machineType | string | Specifies the provider-specific virtual machine type. | No | `Standard_D8_v3` |
    | region | string | Defines the cluster region. | No | `westeurope` |
    | zones | string | Defines the list of zones in which the Runtime Provisioner creates the cluster. | No | `["1", "2", "3"]` |
    | autoScalerMin | int | Specifies the minimum number of virtual machines to create. | No | `2` |
    | autoScalerMax | int | Specifies the maximum number of virtual machines to create. | No | `4` |
    | maxSurge | int | Specifies the maximum number of virtual machines that are created during an update. | No | `4` |
    | maxUnavailable | int | Specifies the maximum number of VMs that can be unavailable during an update. | No | `1` |
    | providerSpecificConfig.AzureConfig.VnetCidr | string | Provides configuration variables specific for Azure. | No | `10.250.0.0/19` |

  </details>
  <details>
    <summary label="plan">
    Azure Lite
    </summary>
        | Parameter Name | Type | Description | Required | Default value |
        |----------------|-------|-------------|:----------:|---------------|
        | machineType | string | Specifies the provider-specific virtual machine type. | No | `Standard_D4_v3` |
        | region | string | Defines the cluster region. | No | `westeurope` |
        | zones | string | Defines the list of zones in which the Runtime Provisioner creates the cluster. | No | `["1", "2", "3"]` |
        | providerSpecificConfig.AzureConfig.VnetCidr | string | Provides configuration variables specific for Azure. | No | `10.250.0.0/19` |
  </details>
  <details>
    <summary label="plan">
    Azure Lite
    </summary>
    
  </details>
  <details>
    <summary label="plan">
    GCP
    </summary>
  </details>
  <details>
    <summary label="plan">
    GCP Trial
    </summary>
  </details>
</div>


