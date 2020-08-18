---
title: Service description
type: Details
---

Kyma Environment Broker (KEB) is compatible with the [Open Service Broker API](https://www.openservicebrokerapi.org/) (OSBA) specification. It provides a ServiceClass that provisions Kyma Runtime on a cluster.

## Service plans

The supported plans are as follows:

| Plan name | Description |
|-----------|-------------|
| `azure` | Installs Kyma Runtime on the Azure cluster. |
| `azure_lite` | Installs Kyma Lite on the Azure cluster. |
| `gcp` | Installs Kyma Runtime on the GCP cluster. |
| `trial` | Installs Kyma Trial on chosen infrastructure. |

## Provisioning parameters

There are two types of configurable provisioning parameters: the ones that are compliant for all providers and provider-specific ones.

### Parameters compliant for all providers

These are the provisioning parameters that you can configure:

| Parameter name | Type | Description | Required | Default value |
|----------------|-------|-------------|:----------:|---------------|
| **name** | string | Specifies the name of the cluster. | Yes | None |
| **nodeCount** | int | Specifies the number of Nodes in a cluster. | No | `3` |
| **components** | array | Defines optional components that are installed in a Kyma Runtime. The possible values are `kiali` and `tracing`. | No | [] |
| **kymaVersion** | string | Provides a Kyma version on demand. | No | None |

### Provider-specific parameters

These are the provisioning parameters for Azure that you can configure:

<div tabs name="azure-plans" group="azure-plans">
  <details>
  <summary label="azure-plan">
  Azure
  </summary>
     
| Parameter Name | Type | Description | Required | Default value |
| ---------------|-------|-------------|:----------:|---------------|
| **machineType** | string | Specifies the provider-specific virtual machine type. | No | `Standard_D8_v3` |
| **volumeSizeGb** | int | Specifies the size of the root volume. | No | `50` |
| **region** | string | Defines the cluster region. | No | `westeurope` |
| **zones** | string | Defines the list of zones in which the Runtime Provisioner creates the cluster. | No | `["1", "2", "3"]` |
| **autoScalerMin** | int | Specifies the minimum number of virtual machines to create. | No | `3` |
| **autoScalerMax** | int | Specifies the maximum number of virtual machines to create. | No | `10` |
| **maxSurge** | int | Specifies the maximum number of virtual machines that are created during an update. | No | `4` |
| **maxUnavailable** | int | Specifies the maximum number of VMs that can be unavailable during an update. | No | `1` |
| **providerSpecificConfig.AzureConfig.VnetCidr** | string | Provides configuration variables specific for Azure. | No | `10.250.0.0/19` |

  </details>
  <details>
  <summary label="azure-lite-plan">
  Azure Lite
  </summary>
    
| Parameter Name | Type | Description | Required | Default value |
| ---------------|-------|-------------|:----------:|---------------|
| **machineType** | string | Specifies the provider-specific virtual machine type. | No | `Standard_D4_v3` |
| **volumeSizeGb** | int | Specifies the size of the root volume. | No | `50` |
| **region** | string | Defines the cluster region. | No | `westeurope` |
| **zones** | string | Defines the list of zones in which the Runtime Provisioner creates the cluster. | No | `["1", "2", "3"]` |
| **autoScalerMin** | int | Specifies the minimum number of virtual machines to create. | No | `3` |
| **autoScalerMax** | int | Specifies the maximum number of virtual machines to create. | No | `4` |
| **maxSurge** | int | Specifies the maximum number of virtual machines that are created during an update. | No | `4` |
| **maxUnavailable** | int | Specifies the maximum number of VMs that can be unavailable during an update. | No | `1` |
| **providerSpecificConfig.AzureConfig.VnetCidr** | string | Provides configuration variables specific for Azure. | No | `10.250.0.0/19` |

 </details>
 </div>

These are the provisioning parameters for GCP that you can configure:
  
<div tabs name="gcp-plans" group="gcp-plans">
  <details>
  <summary label="gcp-plan">
  GCP
  </summary>
    
| Parameter Name | Type | Description | Required | Default value |
| ---------------|-------|-------------|:----------:|---------------|
| **machineType** | string | Specifies the provider-specific virtual machine type. | No | `n1-standard-4` |
| **volumeSizeGb** | int | Specifies the size of the root volume. | No | `30` |
| **region** | string | Defines the cluster region. | No | `europe-west4` |
| **zones** | string | Defines the list of zones in which the Runtime Provisioner creates the cluster. | No | `["a", "b", "c"]` |
| **autoScalerMin** | int | Specifies the minimum number of virtual machines to create. | No | `3` |
| **autoScalerMax** | int | Specifies the maximum number of virtual machines to create. | No | `4` |
| **maxSurge** | int | Specifies the maximum number of virtual machines that are created during an update. | No | `4` |
| **maxUnavailable** | int | Specifies the maximum number of VMs that can be unavailable during an update. | No | `1` |
 
 </details>
 </div>

     
## Trial Plan

Trial Plan allows the user to choose the provider where they want to install Kyma. Trial drawbacks are that Kyma will be
uninstalled, and the cluster will be deprovisioned after 30 days. Apart from that it's possible to provision only one Kyma
per Global Account.

These are the provisioning parameters for Trial Plan that you can configure:
  
<div tabs name="trial-plan" group="trial-plan">
  <details>
  <summary label="trial-plan">
  Trial Plan
  </summary>
    
| Parameter Name | Type | Description | Required | Possible values| Default value |
| ---------------|-------|-------------|:----------:|---------------|
| **name** | string | Specifies the provider-specific virtual machine type. | No | Any string| `n1-standard-4` |
| **region** | int | Specifies the size of the root volume. | No | `europe`,`us` | `30` |
| **provider** | string | Defines the cluster region. | No | `Azure`, `GCP` | `europe-west4` |
 
 </details>
 </div>