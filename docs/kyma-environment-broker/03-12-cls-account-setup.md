---
title: Cloud Logging Service Account setup
type: Details
---

The CLS instances are created inside a subaccount inside the Kyma global account: When the customer provisions the first SKR instance (SKR 1) for their SKR global account, one CLS instance is created inside the Kyma global account.

The following image explains the account setup for SKR and CLS instances:

![CLS Account diagram](./assets/cls-acc.svg)

The diagram illustrates the following: 
1. When a customer with global account X provisions the first SKR (SKR X-1), a CLS instance (CLS for customer X) is created in the CLS subaccount that is part of the Kyma global account. 

2. After provisioning a CLS instance, a binding is created, and the credentials are passed over to SKR X-1. 
 
3. With this setup, the logs are pushed to the CLS instance.

For any subsequent provisioning of SKR (for example, SKR X-2) within the same global account, no new CLS instance is provisioned. Instead, the existing CLS instance is used by creating a new binding, which creates a new set of credentials to access the CLS instance. The new set of credentials is passed over to SKR X-2, so that it can push logs to CLS instance.

To access Kibana, one must authenticate against an ID provider based on SAML2 protocol.

Learn more about the CLS integration with KEB under [Cloud Logging Service Integration](./02-03-cls-integration.md).
