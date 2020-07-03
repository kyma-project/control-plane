# Set overrides for Kyma Runtime

You can set overrides to customize your Kyma Runtime. To provision a cluster with custom overrides, add a Secret or a ConfigMap with a specific label. Kyma Environment Broker uses this Secret and/or ConfigMap to prepare a request to the Runtime Provisioner.

Overrides can be either global or specified for a given component. In the second case, use the `component: {"COMPONENT_NAME"}` label to indicate the component. Create all overrides in the `kcp-system` Namespace.

See the examples:

- ConfigMap with global overrides:
    ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      labels:
        provisioning-runtime-override: "true"
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
        provisioning-runtime-override: "true"
      name: core-overrides
      namespace: kcp-system
    data:
      database.password: YWRtaW4xMjMK
    ```  

### Disable overrides for specific plans

Config Maps and Secrets overrides for customization Kyma Runtime works for all plans but for some kinds of special plans like for example "AzureLite"
overrides can be disabled. To disable a specific override for a special lite plan use label `default-for-lite: "true"`.

See the examples:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    provisioning-runtime-override: "true"
    default-for-lite: "true"
  name: global-overrides
  namespace: compass-system
data:
  global.disableLegacyConnectivity: "true"
```  
    
Above ConfigMap activates global override for all plans except SKR provisioned with special plan marked as `lite`.
