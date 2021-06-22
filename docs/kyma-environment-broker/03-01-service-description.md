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
| `azure_ha` | Installs Kyma Runtime on the Azure cluster with multiple Availability Zones. |
| `aws` | Installs Kyma Runtime on the AWS cluster. |
| `aws_ha` | Installs Kyma Runtime on the AWS cluster with multiple Availability Zones. |
| `openstack` | Installs Kyma Runtime on the Openstack cluster. |
| `gcp` | Installs Kyma Runtime on the GCP cluster. |
| `trial` | Installs Kyma Trial on Azure, AWS or GCP. |
| `freemium` | Installs Kyma Freemium on Azure or AWS. |

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
     
| Parameter name | Type | Description | Required | Default value |
| ---------------|-------|-------------|:----------:|---------------|
| **machineType** | string | Specifies the provider-specific virtual machine type. | No | `Standard_D8_v3` |
| **volumeSizeGb** | int | Specifies the size of the root volume. | No | `50` |
| **region** | string | Defines the cluster region. | No | `westeurope` |
| **zones** | string | Defines the list of zones in which Runtime Provisioner creates a cluster. | No | `["1"]` |
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
    
| Parameter name | Type | Description | Required | Default value |
| ---------------|-------|-------------|:----------:|---------------|
| **machineType** | string | Specifies the provider-specific virtual machine type. | No | `Standard_D4_v3` |
| **volumeSizeGb** | int | Specifies the size of the root volume. | No | `50` |
| **region** | string | Defines the cluster region. | No | `westeurope` |
| **zones** | string | Defines the list of zones in which Runtime Provisioner creates a cluster. | No | `["1"]` |
| **autoScalerMin** | int | Specifies the minimum number of virtual machines to create. | No | `3` |
| **autoScalerMax** | int | Specifies the maximum number of virtual machines to create. | No | `4` |
| **maxSurge** | int | Specifies the maximum number of virtual machines that are created during an update. | No | `4` |
| **maxUnavailable** | int | Specifies the maximum number of VMs that can be unavailable during an update. | No | `1` |

 </details>
 </div>

These are the provisioning parameters for AWS that you can configure:
<div tabs name="aws-plans" group="aws-plans">
  <details>
  <summary label="aws-plan">
  AWS
  </summary>

| Parameter name | Type | Description | Required | Default value |
| ---------------|-------|-------------|:----------:|---------------|
| **machineType** | string | Specifies the provider-specific virtual machine type. | No | `Standard_D8_v3` |
| **volumeSizeGb** | int | Specifies the size of the root volume. | No | `50` |
| **region** | string | Defines the cluster region. | No | `westeurope` |
| **zones** | string | Defines the list of zones in which Runtime Provisioner creates a cluster. | No | `["1"]` |
| **autoScalerMin** | int | Specifies the minimum number of virtual machines to create. | No | `3` |
| **autoScalerMax** | int | Specifies the maximum number of virtual machines to create. | No | `10` |
| **maxSurge** | int | Specifies the maximum number of virtual machines that are created during an update. | No | `4` |
| **maxUnavailable** | int | Specifies the maximum number of VMs that can be unavailable during an update. | No | `1` |

  </details>
  <details>
  <summary label="aws-ha-plan">
  AWS HA
  </summary>

| Parameter name | Type | Description | Required | Default value |
| ---------------|-------|-------------|:----------:|---------------|
| **machineType** | string | Specifies the provider-specific virtual machine type. | No | `Standard_D4_v3` |
| **volumeSizeGb** | int | Specifies the size of the root volume. | No | `50` |
| **region** | string | Defines the cluster region. | No | `westeurope` |
| **zones** | string | Defines the list of zones in which Runtime Provisioner creates a cluster. | No | `["1"]` |
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
    
| Parameter name | Type | Description | Required | Default value |
| ---------------|-------|-------------|:----------:|---------------|
| **machineType** | string | Specifies the provider-specific virtual machine type. | No | `n1-standard-4` |
| **volumeSizeGb** | int | Specifies the size of the root volume. | No | `30` |
| **region** | string | Defines the cluster region. | No | `europe-west4` |
| **zones** | string | Defines the list of zones in which Runtime Provisioner creates a cluster. | No | `["a"]` |
| **autoScalerMin** | int | Specifies the minimum number of virtual machines to create. | No | `3` |
| **autoScalerMax** | int | Specifies the maximum number of virtual machines to create. | No | `4` |
| **maxSurge** | int | Specifies the maximum number of virtual machines that are created during an update. | No | `4` |
| **maxUnavailable** | int | Specifies the maximum number of VMs that can be unavailable during an update. | No | `1` |
 
 </details>
 </div>

     
## Trial plan

Trial plan allows you to install Kyma on Azure, AWS or GCP. The Trial plan assumptions are as follows:
- Kyma is uninstalled after 30 days and the Kyma cluster is deprovisioned after this time.
- It's possible to provision only one Kyma Runtime per global account.

To reduce the costs, the Trial plan skips some of the [provisioning steps](./03-03-runtime-operations.md#provisioning).
- `Provision_Azure_Event_Hubs`
- `AVS External Evaluation` (part of the post actions during the `Initialisation` step)

### Provisioning parameters

These are the provisioning parameters for the Trial plan that you can configure:
  
<div tabs name="trial-plan" group="trial-plan">
  <details>
  <summary label="trial-plan">
  Trial plan
  </summary>
    
| Parameter name | Type | Description | Required | Possible values| Default value |  
| ---------------|-------|-------------|----------|---------------|---------------|  
| **name** | string | Specifies the name of the Kyma Runtime. | Yes | Any string| None |  
| **region** | string | Defines the cluster region. | No | `europe`,`us`, `asia` | Calculated from the platform region |  
| **provider** | string | Specifies the cloud provider used during provisioning. | No | `Azure`, `AWS`, `GCP` | `Azure` |
 
The **region** parameter is optional. If not specified, the region is calculated from platform region specified in this path:
```shell
/oauth/{platform-region}/v2/service_instances/{instance_id}
```
The mapping between the platform region and the provider region (Azure, AWS or GCP) is defined in the configuration file in the **APP_TRIAL_REGION_MAPPING_FILE_PATH** environment variable. If the platform region is not defined, the default value is `europe`.

 </details>
 </div>

