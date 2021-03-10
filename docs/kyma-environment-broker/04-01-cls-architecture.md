---
title: Cloud Logging Service Integration
type: Architecture
---

The Cloud Logging service (CLS) is a managed logging service for shipping logs from various runtimes such as SKR. It uses the Elastic Stack (open source logging platform Elasticsearch, Logstash, Kibana) to store, parse, and visualize the application log data coming from Kubernetes applications.

 Kyma Environment Broker (KEB) provisions a CLS instance for a global account and creates a binding. It passes the CLS credentials to the provisioner as SKR overrides, so that Fluent Bit in the SKR is properly configured to push the logs to the CLS instance. The following architecture diagram explains the complete process:

![CLS diagram](./assets/cls-arch.svg)

1. CSI sends the request to KEB to provision a new SKR.
  - If there is no CLS instance present for the global account, then KEB provisions a new instance for CLS for the global account.

      - If the CLS instance is already provisioned for the global account, then KEB creates a binding. see [3.](3)

3. After the CLS instance is provisioned, a binding is created, through which the KEB gets the credentials to push logs to the CLS instance.
4. After getting the CLS credentials, these credentials are appended to SKR overrides
5. KEB triggers Provisioner to provision a new SKR after along the overrides for SKR.
6. Provisioner provisions the SKR and the Fluent Bit plugin is configured using the CLS credentials.
7. With this configuration, SKR pushes logs to the CLS instance.
