# Metris

## Overview

Metris is a metering component that collects data and sends them to EDP.

## Configuration

| CLI argument | Environment variable | Description | Default value |
| -- | -- | -- | -- |
| `--edp-url` | **EDP_URL** | EDP base URL | `https://input.yevents.io` |
| `--edp-token` | **EDP_TOKEN** | EDP source token | None |
| `--edp-namespace` | **EDP_NAMESPACE** | EDP Namespace | None |
| `--edp-data-stream` | **EDP_DATASTREAM_NAME** | EDP data stream name | None |
| `--edp-data-stream-version` | **EDP_DATASTREAM_VERSION** | EDP data stream version | None |
| `--edp-data-stream-env` | **EDP_DATASTREAM_ENV** | EDP data stream environment | None |
| `--edp-timeout` | **EDP_TIMEOUT** | Time limit for requests made by the EDP client | `30s` |
| `--edp-buffer` | **EDP_BUFFER** | Number of events that the buffer can have. | `100` |
| `--edp-workers` | **EDP_WORKERS** | Number of workers to send metrics. | `5` |
| `--edp-event-retry` | **EDP_RETRY** | Number of retries for sending event. | `5` |
| `--provider-poll-interval` | **PROVIDER_POLLINTERVAL** | Interval at which metrics are fetch. | `1m` |
| `--provider-workers` | **PROVIDER_WORKERS** | Number of workers to fetch metrics. | `10` |
| `--provider-buffer` | **PROVIDER_BUFFER** | Number of cluster that the buffer can have. | `100` |
| `--listen-addr` | **METRIS_LISTEN_ADDRESS** | Address and port the metrics and health HTTP endpoints will bind to. | None |
| `--debug-port` | **METRIS_DEBUG_PORT** | Port the debug HTTP endpoint will bind to. Always listen on localhost. | None |
| `--config-file` | None | Location of the config file. | None |
| `--kubeconfig` | **METRIS_KUBECONFIG** | Path to the Gardener kubeconfig file. | None |
| `--log-level` | **METRIS_LOGLEVEL** | Logging level. (debug,info,warn,error) | `info` |
