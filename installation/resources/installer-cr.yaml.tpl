apiVersion: "installer.kyma-project.io/v1alpha1"
kind: Installation
metadata:
  name: kcp-installation
  namespace: default
  labels:
    action: install
    kyma-project.io/installation: ""
  finalizers:
    - finalizer.installer.kyma-project.io
spec:
  version: "__VERSION__"
  url: "__URL__"
  components:
    - name: "istio-gateway"
      namespace: "istio-system"
    - name: "provisioner"
      namespace: "kcp-system"
    - name: "kyma-environment-broker"
      namespace: "kcp-system"
    - name: "oidc-kubeconfig-service"
      namespace: "kyma-system"
