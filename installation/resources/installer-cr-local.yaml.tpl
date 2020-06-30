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
    - name: "postgresql"
      namespace: "kcp-system"
    - name: "oidc-kubeconfig-service"
      namespace: "kyma-system"
    - name: "metris"
      namespace: "kcp-system"
