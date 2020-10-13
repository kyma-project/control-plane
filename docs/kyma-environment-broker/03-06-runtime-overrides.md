---
title: Set overrides for Kyma Runtime
type: Details
---

You can set overrides to customize your Kyma Runtime. To provision a cluster with custom overrides, add a Secret or a ConfigMap with a specific label. Kyma Environment Broker uses this Secret and/or ConfigMap to prepare a request to the Runtime Provisioner.

> **NOTE:** Create all overrides in the `kcp-system` Namespace.

## ConfigMap

The overrides mechanism selects ConfigMaps by filtering the resources using labels. You can prepare overrides for a given plan and Kyma version, using the `overrides-plan-{"PLAN_NAME"}: "true"` and `overrides-version-{"KYMA_VERSION"}: "true"` labels.

> **NOTE:** Each ConfigMap that defines overrides must have both labels assigned.

Optionally you can narrow the scope for the overrides to a specific component. Use the `component: {"COMPONENT_NAME"}` label to indicate the component. 

The mechanism for the overrides lookup requires at least ConfigMap to be present otherwise it fails.

See the examples:

- ConfigMap with global overrides for plan `trial` and versions `1.15.1`, and `1.16.0`:

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

### Use Kyma default overrides for specific plan and version

By default, the overrides lookup mechanism expects at least one ConfigMap present for each plan and version pair. It will fail otherwise. To allow the Kyma installation without providing any additional overrides create empty ConfigMap.

Example:

- Empty ConfigMap for plan `lite` and version `1.16.0`

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

In order to use Secret to provide overrides, you must label it using `runtime-override: "true"`. Optionally you can narrow the scope for the overrides to a specific component. Use the `component: {"COMPONENT_NAME"}` label to indicate the component. 

See the examples:

- Secret with global overrides:

    ```yaml
    apiVersion: v1
    kind: ConfigMap
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
