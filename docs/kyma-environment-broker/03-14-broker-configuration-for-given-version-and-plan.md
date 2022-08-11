# KEB configuration for given Kyma version and plan

Some processes in Kyma Environment Broker can be configured to deliver different results. KEB requires a ConfigMap with 
a configuration for given Kyma version and plan to process the requests. At least default configuration should be defined 
for chosen Kyma version which KEB recognizes as applicable for all the supported plans. You can also set separate configuration 
for each of the plans. 
  
When processing requests KEB reads configuration from a ConfigMap which holds data about processable Kyma version(s) and 
configurable units for a given plan. Only one ConfigMap can exist for given Kyma version, but it also can be set for 
multiple Kyma versions if the configuration is the same for every targeted version.  

> **NOTE:** Create all configurations in the `kcp-system` Namespace.

> **NOTE:** As this is the first iteration of KEB configuration concept, only additional components list can be configured.

## ConfigMap  

Appropriate ConfigMap is selected by filtering the resources using labels. KEB recognizes ConfigMaps with configuration 
when they contain at least these two labels:

```yaml
keb-config: "true"
runtime-version-{KYMA_VERSION}: "true"
```

You can assign more than one ```runtime-version-{KYMA_VERSION}: "true"``` label as long as the configuration is the 
same for provided Kyma versions.

> **NOTE:** Each ConfigMap that defines configuration must have both labels assigned.

The actual configuration is stored in ConfigMap's `data` object. Add `default` key under `data`object:

```yaml
data:
  default: |-
    additional-components:
      - name: "additional-component1"
        namespace: "kyma-system"
```

You must define a default configuration which is selected when supported plan key is missing. That means if there are no 
other plan keys under `data` object then the default configuration is applicable for all the plans. 

See the example of a ConfigMap with a configuration for Kyma version `2.5.3` and `plan1`, `plan2`, `trial` plans:

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
# no components

```