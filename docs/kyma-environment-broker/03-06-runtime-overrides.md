# Set overrides for Kyma Runtime

You can set overrides to customize your Kyma Runtime. To provision a cluster with custom overrides, add a Secret or a ConfigMap with a specific label. Kyma Environment Broker uses this Secret and/or ConfigMap to prepare a request to the Runtime Provisioner.

> **NOTE:** Create all overrides in the `kcp-system` Namespace.

## ConfigMap

The overrides mechanism selects ConfigMaps by filtering the resources using labels. You can prepare overrides for three kinds of combination:
1. a given plan and Kyma version using the `overrides-plan-{PLAN_NAME}: "true"` and `overrides-version-{KYMA_VERSION}: "true"` labels.
2. a given global account ID and Kyma version using the `overrides-account-{GLOBAL_ACCOUNT_ID}: "true"` and `overrides-version-{KYMA_VERSION}: "true"` labels.
3. a given subaccount ID and Kyma version using the  `overrides-account-{SUB_ACCOUNT_ID}: "true"` and `overrides-version-{KYMA_VERSION}: "true"` labels.

> **NOTE:** Each ConfigMap that defines overrides must have both labels assigned.

Optionally, you can narrow the scope of the overrides to a specific component. Use the `component: "{COMPONENT_NAME}"` label to indicate the component.

The overrides lookup mechanism requires at least one ConfigMap present for each plan and version pair. Otherwise, it fails.

See the examples of a ConfigMap：
1. Configmap with global overrides for the `trial` plan and versions `1.15.1` and `1.16.0`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    overrides-plan-trial: "true"
    overrides-version-1.15.1: "true"
    overrides-version-1.16.0: "true"
  name: global-overrides
  namespace: kcp-system
data:
  global.disableLegacyConnectivity: "true"
```

2. Configmap with global overrides for the `global account` ID and versions `1.15.1` and `1.16.0`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    overrides-account-d9994f8f-7e46-42a8-b2c1-1cddd9d2ef06: "true"
    overrides-version-1.15.1: "true"
  name: global-overrides
  namespace: kcp-system
data:
  global.disableLegacyConnectivity: "true"
```

2. Configmap with global overrides for the `subaccount` ID and versions `1.16.0`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    overrides-subaccount-39ba9a66-2c1a-4fe4-a28e-4e8fb434253s: "true"
    overrides-version-1.16.0: "true"
  name: global-overrides
  namespace: kcp-system
data:
  global.disableLegacyConnectivity: "true"
```

### Use Kyma default overrides for specific plan and version

By default, the overrides lookup mechanism expects at least one ConfigMap present for each plan and version pair. Otherwise, it fails. To allow Kyma installation without providing any additional overrides, create an empty ConfigMap.

See the example of an empty ConfigMap for the `lite` plan and version `1.16.0`:


```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    overrides-plan-lite: "true"
    overrides-version-1.16.0: "true"
data:
```

## Secrets

In order to provide overrides using Secrets, label the Secret with the `runtime-override: "true"` label. Optionally, you can narrow the scope of the overrides to a specific component. Use the `component: "{COMPONENT_NAME}"` label to indicate the component.

See the examples:

- Secret with global overrides:

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      labels:
        runtime-override: "true"
      name: global-overrides
      namespace: kcp-system
    data:
      global.disableLegacyConnectivity: "true"
    ```  

- Secret with overrides for the `core` component:

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      labels:
        component: "core"
        runtime-override: "true"
      name: core-overrides
      namespace: kcp-system
    data:
      database.password: YWRtaW4xMjMK
    ```  
