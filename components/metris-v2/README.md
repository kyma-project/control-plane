### Metris

#### Description
Metris scrapes all Kyma clusters and uses Shoot information to generate metrics as event streams. The generated event streams are POST-ed to an events collecting system.

#### Usage

- `Metris` comes with the following command line argument flags:

    | Flag | Description | Default Value   |
    | ----- | ------------ | --------------- |
    | `gardener-secret-path` | The path to the secret which contains kubeconfig of the Gardener MPS cluster. | `/gardener/kubeconfig` |
    | `gardener-namespace` | The namespace in gardener cluster where information on Kyma clusters are. | `garden-kyma-dev`    |
    | `scrape-interval` | The wait duration of the interval between 2 executions of metrics generation. | `3m`         |
    | `worker-pool-size` | The number of workers in the pool. | `5` |
    | `log-level` | The log-level of the application. E.g. fatal, error, info, debug etc. | `info` |
    | `listen-addr` | The application starts server in this port to cater to the metrics and health endpoints. | `8080` |
    | `debug-port` | The custom port to debug when needed. `0` will disable debugging server. | `0` |

- `Metris` comes with the following environment variables:
     
     | Variable | Description | Default Value   |
     | ----- | ------------ | ------------- |
     | `PUBLIC_CLOUD_SPECS` | The specification contains the CPU, Network and Disk information for all machine types from a public cloud provider.  | `-` |
     | `KEB_URL` | The KEB URL where Metris fetches runtime information. | `-` |
     | `KEB_TIMEOUT` | The timeout governs the connections from Metris to KEB | `30s` |
     | `KEB_RETRY_COUNT` | The number of retries Metris will do when connecting to KEB fails. | 5 |
     | `KEB_POLL_WAIT_DURATION` | The wait duration for Metris between each execution of polling KEB for runtime information. | `10m` |
     | `EDP_URL` | The EDP base URL where Metris will ingest event-stream to. | `-` |
     | `EDP_TOKEN` | The token used to connect to EDP. | `-` |
     | `EDP_NAMESPACE` | The namespace in EDP where Metris will ingest event-stream to.| `kyma-dev` |
     | `EDP_DATASTREAM_NAME` | The datastream in EDP where Metris will ingest event-stream to. | `consumption-metrics` |
     | `EDP_DATASTREAM_VERSION` | The datastream version which Metris will use. | `1` |
     | `EDP_DATASTREAM_ENV` | The datastream environment which Metris will use.  | `dev` |
     | `EDP_TIMEOUT` | The timeout for Metris connections to EDP. | `30s` |
     | `EDP_RETRY` | The number of retries for Metris connections to EDP. | `3` |

#### Development
- Run a deployment in currently configured k8s cluster

```
ko apply -f dev/  
```

- Resolve all dependencies
```
make gomod-vendor
```

- Run tests
```
make tests
```

- Run tests and publish a test coverage report
```
make publish-test-results
```

#### Troubleshooting
- Check logs
```
kubectl logs -l app=metrisv2 -n kcp-system -c metrisv2 -f
```

#### Data collection

Metris collects information about billable hyperscaler usage and sends it to EDP. This data has to adhere to the following schema:

```json
{
  "name": "consumption-metrics",
  "jsonSchema": {
    "type": "object",
    "title": "SKR Metering Schema",
    "description": "SKR Metering Schema.",
    "required": [
      "timestamp",
      "compute",
      "networking"
    ],
    "properties": {
      "timestamp": {
        "$id": "#/properties/timestamp",
        "type": "string",
        "format": "date-time",
        "title": "The Timestamp Schema",
        "description": "Event Creation Timestamp",
        "default": "",
        "examples": ["2020-03-25T09:16:41+00:00"]
      },
      "compute": {
        "$id": "#/properties/compute",
        "type": "object",
        "title": "The Compute Schema",
        "description": "Contains Azure Compute metrics",
        "default": {},
        "examples": [
          {
            "provisioned_cpus": 24.0,
            "provisioned_volumes": {
              "size_gb_rounded": 192.0,
              "count": 3.0,
              "size_gb_total": 150.0
            },
            "vm_types": [
              {
                "name": "Standard_D8_v3",
                "count": 3.0
              },
              {
                "name": "Standard_D6_v3",
                "count": 2.0
              }
            ],
            "provisioned_ram_gb": 96.0
          }
        ],
        "required": [
          "vm_types",
          "provisioned_cpus",
          "provisioned_ram_gb",
          "provisioned_volumes"
        ],
        "properties": {
          "vm_types": {
            "$id": "#/properties/compute/properties/vm_types",
            "type": "array",
            "title": "The Vm_types Schema",
            "description": "A list of VM types that have been used for this SKR instance.",
            "default": [],
            "items": {
              "$id": "#/properties/compute/properties/vm_types/items",
              "type": "object",
              "title": "The Items Schema",
              "description": "The Azure instance type name and the provisioned quantity at the time of the event.",
              "default": {},
              "examples": [
                {
                  "name": "Standard_D8_v3",
                  "count": 3.0
                },
                {
                  "name": "Standard_D6_v3",
                  "count": 2.0
                }
              ],
              "required": ["name", "count"],
              "properties": {
                "name": {
                  "$id": "#/properties/compute/properties/vm_types/items/properties/name",
                  "type": "string",
                  "title": "The Name Schema",
                  "description": "Name of the instance type",
                  "default": "",
                  "examples": ["Standard_D8_v3"]
                },
                "count": {
                  "$id": "#/properties/compute/properties/vm_types/items/properties/count",
                  "type": "integer",
                  "title": "The Count Schema",
                  "description": "Quantity of the instances",
                  "default": 0,
                  "examples": [3]
                }
              }
            }
          },
          "provisioned_cpus": {
            "$id": "#/properties/compute/properties/provisioned_cpus",
            "type": "integer",
            "title": "The Provisioned_cpus Schema",
            "description": "The total sum of all CPUs provisioned from all instances (number of instances *  number of CPUs per instance)",
            "default": 0,
            "examples": [24]
          },
          "provisioned_ram_gb": {
            "$id": "#/properties/compute/properties/provisioned_ram_gb",
            "type": "integer",
            "title": "The Provisioned_ram_gb Schema",
            "description": "The total sum of Memory (RAM) of all provisioned instances (number of instances * number of GB RAM per instance).",
            "default": 0,
            "examples": [96]
          },
          "provisioned_volumes": {
            "$id": "#/properties/compute/properties/provisioned_volumes",
            "type": "object",
            "title": "The Provisioned_volumes Schema",
            "description": "Volumes (Disk) provisioned(excluding the Node volumes).",
            "default": {},
            "examples": [
              {
                "size_gb_rounded": 192.0,
                "count": 3.0,
                "size_gb_total": 150.0
              }
            ],
            "required": ["size_gb_total", "count", "size_gb_rounded"],
            "properties": {
              "size_gb_total": {
                "$id": "#/properties/compute/properties/provisioned_volumes/properties/size_gb_total",
                "type": "integer",
                "title": "The Size_gb_total Schema",
                "description": "The total GB disk space requested by a kyma instance",
                "default": 0,
                "examples": [150]
              },
              "count": {
                "$id": "#/properties/compute/properties/provisioned_volumes/properties/count",
                "type": "integer",
                "title": "The Count Schema",
                "description": "The number of disks provisioned.",
                "default": 0,
                "examples": [3]
              },
              "size_gb_rounded": {
                "$id": "#/properties/compute/properties/provisioned_volumes/properties/size_gb_rounded",
                "type": "integer",
                "title": "The Size_gb_rounded Schema",
                "description": "Azure charges disk in 32GB blocks. If one provisions e.g. 16GB, he still pays 32 GB. This value here is rounding up each volume to the next y 32 dividable number and sums these values up.",
                "default": 0,
                "examples": [192]
              }
            }
          }
        }
      },
      "networking": {
        "$id": "#/properties/networking",
        "type": "object",
        "title": "The Networking Schema",
        "description": "Some networking controlling data.",
        "default": {},
        "examples": [
          {
            "provisioned_vnets": 2.0,
            "provisioned_ips": 3.0
          }
        ],
        "required": [
          "provisioned_vnets",
          "provisioned_ips"
        ],
        "properties": {
          "provisioned_vnets": {
            "$id": "#/properties/networking/properties/provisioned_vnets",
            "type": "integer",
            "title": "The Provisioned_vnets Schema",
            "description": "Number of virtual networks",
            "default": 0,
            "examples": [2]
          },
          "provisioned_ips": {
            "$id": "#/properties/networking/properties/provisioned_ips",
            "type": "integer",
            "title": "The Provisioned_ips Schema",
            "description": "Number of IPs",
            "default": 0,
            "examples": [3]
          }
        }
      }
    }
  },
  "version": "1",
  "eventTimeField": "event.timestamp"
}
```

See the example of data sent to EDP:

```json
{
  "compute": {
    "vm_types": [
      {
        "name": "Standard_D8_v3",
        "count": 3
      },
      {
        "name": "Standard_D6_v3",
        "count": 2
      }
    ],
    "provisioned_cpus": 24,  
    "provisioned_ram_gb": 96,
    "provisioned_volumes": {
      "size_gb_total": 150,
      "count": 3,
      "size_gb_rounded": 192
    }
  },
  "networking": {
    "provisioned_vnets": 2,
    "provisioned_ips": 3
  }
}
```
