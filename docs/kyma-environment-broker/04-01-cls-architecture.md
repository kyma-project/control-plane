---
title: Cloud Logging Service Architecture
type: Architecture
---

Cloud Logging Service (CLS) is a managed logging service for shipping logs from Runtimes. It uses the Elastic Stack (open source logging platform Elasticsearch, Logstash, Kibana) to store, parse, and visualize the application log data coming from Kubernetes applications.

 Kyma environment broker provisions a CLS instance for a global account, creates a binding and passes the CLS credentials to the provisioner as SKR overrides so that fluent bit in the SKR can be properly configured to push the logs to the CLS instance. The architecture diagram below explains the complete process

![CLS diagram](./assets/cls-arch.svg)

1. CSI sends the request to KEB to provision a new KEB.
2. If there is not CLS instance present for the global account, the it provisions a new instance for CLS for the global account.

    a. If the CLS instance is already provisioned for the global account then KEB would create a new binding. see [3.](3)

3. After the CLS instance is provisioned then we create a binding through which we get the credentials to push logs to the CLS instance.
4. After getting the CLS credentials, these credentials are appended to SKR overrides
5. KEB triggers provisioner to provision a new SKR after along the overrides for SKR.
6. Provisioner provisions the SKR and CLS credentials are configured in the fluent-bit plugin.
7. SKR is now able to push logs to the CLS instance.