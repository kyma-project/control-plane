# Kyma Environment Broker (KEB) configuration for a given Kyma version and plan

Some processes in KEB can be configured to deliver different results. KEB needs a ConfigMap with a configuration for the given Kyma version and plan to process the requests. 
A default configuration must be defined for the chosen Kyma version. This configuration must be recognized by KEB as applicable for all the supported plans. You can also set a separate configuration for each plan.
  
While processing requests, KEB reads a configuration from a ConfigMap which holds data about processable Kyma versions and configurable units for a given plan. Only one ConfigMap can exist for a given Kyma version, but it also can be set for multiple Kyma versions if the configuration is the same for every targeted version.

> **NOTE:** Create all configurations in the `kcp-system` Namespace.

> **NOTE:** As this is the first iteration of the KEB configuration concept, only the additional components list can be configured.

## ConfigMap  

The appropriate ConfigMap is selected by filtering the resources using labels. KEB recognizes the ConfigMaps with configurations when they contain these two labels:

```yaml
keb-config: "true"
runtime-version-{KYMA_VERSION}: "true"
```

You can assign more than one ```runtime-version-{KYMA_VERSION}: "true"``` label as long as the configuration is the same for the provided Kyma versions.

> **NOTE:** Each ConfigMap that defines a configuration must have both labels assigned.

The actual configuration is stored in ConfigMap's `data` object. Add `default` key under `data` object:

```yaml
data:
  default: |-
    additional-components:
      - name: "additional-component1"
        namespace: "kyma-system"
```

You must define a default configuration that is selected when the supported plan key is missing. This means that, for example, if there are no other plan keys under the `data` object, the default configuration applies to all the plans. 

See the example of a ConfigMap with a configuration for Kyma version `2.5.3` and `plan1`, `plan2`, and `trial` plans:

```yaml
# keb-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: keb-config
  labels:
    keb-config: "true"
    runtime-version-2.5.3: "true"
data:
  default: |-
    additional-components:
      - name: "additional-component1"
        namespace: "kyma-system"
      - name: "additional-component2"
        namespace: "kyma-system"
      - name: "additional-component3"
        namespace: "kyma-system"
        source:
          url: "https://example.source.url.local/artifacts/additional-component3-0.0.1.tgz"
  plan1: |-
    additional-components:
      - name: "additional-component1"
        namespace: "kyma-system"
      - name: "additional-component3"
        namespace: "kyma-system"
        source:
          url: "https://example.source.url.local/artifacts/additional-component3-0.0.1.tgz"
  plan2: |-
    additional-components:
      - name: "additional-component2"
        namespace: "kyma-system"
      - name: "additional-component3"
        namespace: "kyma-system"
        source:
          url: "https://example.source.url.local/artifacts/additional-component3-0.0.1.tgz"
  trial: |-
    additional-components:
# No components

```