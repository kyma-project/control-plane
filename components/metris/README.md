# Metris

## Overview

Metris is a metering component that collects data and sends them to EDP.

## Configuration

| CLI argument | Environment variable | Description | Default value |
| ------------ | -------------------- | ----------- | ------------- |
| `--edp-url` | **EDP_URL** | EDP base URL | `https://input.yevents.io` |
| `--edp-token` | **EDP_TOKEN** | EDP source token | None |
| `--edp-namespace` | **EDP_NAMESPACE** | EDP Namespace | None |
| `--edp-data-stream` | **EDP_DATASTREAM_NAME** | EDP data stream name | None |
| `--edp-data-stream-version` | **EDP_DATASTREAM_VERSION** | EDP data stream version | None |
| `--edp-data-stream-env` | **EDP_DATASTREAM_ENV** | EDP data stream environment | None |
| `--edp-timeout` | **EDP_TIMEOUT** | Time limit for requests made by the EDP client | `30s` |
| `--edp-buffer` | **EDP_BUFFER** | Number of events that the buffer can have | `100` |
| `--edp-workers` | **EDP_WORKERS** | Number of workers to send metrics | `5` |
| `--edp-event-retry` | **EDP_RETRY** | Number of retries for sending an event | `5` |
| `--provider-poll-interval` | **PROVIDER_POLLINTERVAL** | Interval at which metrics are fetched | `5m` |
| `--provider-poll-max-interval` | **PROVIDER_POLLMAXINTERVAL** | maximum Interval at which metrics are fetch | `15m` |
| `--provider-poll-duration` | **PROVIDER_POLLDURATION** | Time limit for requests made by the provider client | `5m` |
| `--provider-max-retries` | **PROVIDER_MAXRETRIES** | Maximum number of retries before a cluster is removed from the cache if it is not found on the provider. NOTE: This will stop sending events for the removed cluster | `20` |
| `--provider-workers` | **PROVIDER_WORKERS** | Number of workers to fetch metrics | `10` |
| `--provider-buffer` | **PROVIDER_BUFFER** | Number of clusters that the buffer can have | `100` |
| `--listen-addr` | **METRIS_LISTEN_ADDRESS** | Address and port the metrics and health HTTP endpoints will bind to | None |
| `--debug-port` | **METRIS_DEBUG_PORT** | Port the debug HTTP endpoint will bind to (always listen on localhost) | None |
| `--config-file` | None | Location of the `config` file | None |
| `--kubeconfig` | **METRIS_KUBECONFIG** | Path to the Gardener `kubeconfig` file | None |
| `--log-level` | **METRIS_LOGLEVEL** | Logging level (`debug`,`info`,`warn`,`error`) | `info` |
| `--tracing` | **TRACING_ENABLE** | Enable tracing | `false` |
| `--zipkin-url` | **ZIPKIN_URL** | Zipkin Collector URL | `http://localhost:9411/api/v2/spans` |

## Data collection

Metris collects information about billabe hyperscaler usage and sends it to EDP. This data has to adhere to the following schema:

```json
{
  "name": "consumption-metrics",
  "jsonSchema": {
    "type": "object",
    "title": "SKR Metering Schema",
    "description": "SKR Metering Schema.",
    "required": [
      "timestamp",
      "resource_groups",
      "compute",
      "networking",
      "event_hub"
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
      "resource_groups": {
        "$id": "#/properties/resource_groups",
        "type": "array",
        "title": "The Resource_groups Schema",
        "description": "A list of resource groups that have been used for generating this event. In General these are the resource groups of the Gardener Shoot and the Azure Event Hub for knative.",
        "default": [],
        "items": {
          "$id": "#/properties/resource_groups/items",
          "type": "string",
          "title": "The Items Schema",
          "description": "The name of the resource group",
          "default": "",
          "examples": ["group1", "group2"]
        }
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
            "description": "Volumes (Disk) provisioned.",
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
            "provisioned_loadbalancers": 1.0,
            "provisioned_ips": 3.0
          }
        ],
        "required": [
          "provisioned_loadbalancers",
          "provisioned_vnets",
          "provisioned_ips"
        ],
        "properties": {
          "provisioned_loadbalancers": {
            "$id": "#/properties/networking/properties/provisioned_loadbalancers",
            "type": "integer",
            "title": "The Provisioned_loadbalancers Schema",
            "description": "Number of loadbalancers.",
            "default": 0,
            "examples": [1]
          },
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
      },
      "event_hub": {
        "$id": "#/properties/event_hub",
        "type": "object",
        "title": "The Event_hub Schema",
        "description": "The Azure Event Hub Metrics.",
        "default": {},
        "examples": [
          {
            "number_namespaces": 3.0,
            "max_outgoing_bytes_pt5m": 5600.0,
            "incoming_requests_pt5m": 3.0,
            "max_incoming_bytes_pt5m": 5600.0
          }
        ],
        "required": [
          "number_namespaces",
          "incoming_requests_pt1m",
          "max_incoming_bytes_pt1m",
          "max_outgoing_bytes_pt1m",
          "incoming_requests_pt5m",
          "max_incoming_bytes_pt5m",
          "max_outgoing_bytes_pt5m"
        ],
        "properties": {
          "number_namespaces": {
            "$id": "#/properties/event_hub/properties/number_namespaces",
            "type": "integer",
            "title": "The Number_namespaces Schema",
            "description": "The number of provisioned namespaces.",
            "default": 0,
            "examples": [3]
          },
          "incoming_requests_pt1m": {
            "$id": "#/properties/event_hub/properties/incoming_requests_pt1m",
            "type": "integer",
            "title": "The incoming_requests_pt1m Schema",
            "description": "The number of incoming events counted of the last minute.",
            "default": 0,
            "examples": [3]
          },
          "max_incoming_bytes_pt1m": {
            "$id": "#/properties/event_hub/properties/max_incoming_bytes_pt1m",
            "type": "integer",
            "title": "The Max_incoming_bytes_pt1m Schema",
            "description": "The maximum incoming bytes over last minute.",
            "default": 0,
            "examples": [5600]
          },
          "max_outgoing_bytes_pt1m": {
            "$id": "#/properties/event_hub/properties/max_outgoing_bytes_pt1m",
            "type": "integer",
            "title": "The max_outgoing_bytes_pt1m Schema",
            "description": "The maximum outgoing byte over last minute.",
            "default": 0,
            "examples": [5600]
          },
          "incoming_requests_pt5m": {
            "$id": "#/properties/event_hub/properties/incoming_requests_pt5m",
            "type": "integer",
            "title": "The incoming_requests_pt5m Schema",
            "description": "The number of incoming events counted of the last 5 mins.",
            "default": 0,
            "examples": [3]
          },
          "max_incoming_bytes_pt5m": {
            "$id": "#/properties/event_hub/properties/max_incoming_bytes_pt5m",
            "type": "integer",
            "title": "The Max_incoming_bytes_pt5m Schema",
            "description": "The maximum incoming bytes over last 5 minutes.",
            "default": 0,
            "examples": [5600]
          },
          "max_outgoing_bytes_pt5m": {
            "$id": "#/properties/event_hub/properties/max_outgoing_bytes_pt5m",
            "type": "integer",
            "title": "The Max_outgoing_bytes_pt5m Schema",
            "description": "The maximum outgoing byte over last 5 minutes.",
            "default": 0,
            "examples": [5600]
          }
        }
      }
    }
  },
  "version": "1",
  "eventTimeField": "event.timestamp"
}
```

An example of data sent:

```json
{
  "resource_groups": [
    "group1",
    "group2"
  ],
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
    "provisioned_loadbalancers": 1,
    "provisioned_vnets": 2,
    "provisioned_ips": 3
  },
  "event_hub": {
    "number_namespaces": 3,
    "incoming_requests_pt1m": 3,
    "max_incoming_bytes_pt1m": 5600,
    "max_outgoing_bytes_pt1m": 5600,
    "incoming_requests_pt5m": 0,
    "max_incoming_bytes_pt5m": 0,
    "max_outgoing_bytes_pt5m": 0
  }
}
```

As can be seen, the `event_hub` part of the data is specific for Azure, whereas the other sections can be used with other hyperscalers as well.
