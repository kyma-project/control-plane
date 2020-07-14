#!/usr/bin/env bash

set -o errexit

DOMAIN=${DOMAIN:-kyma.local}
ISTIO_GATEWAY_NAME=${ISTIO_GATEWAY_NAME:-compass-istio-gateway}
ISTIO_GATEWAY_NAMESPACE=${ISTIO_GATEWAY_NAMESPACE:-compass-system}

echo "Creating 'kcp-poc' namespace..."
kubectl create ns kcp-poc || true

echo "Installing Runtime Director chart..."
helm install kcp-poc -n kcp-poc ./chart \
  --set global.ingress.domainName="${DOMAIN}" \
  --set global.istio.gateway.name="${ISTIO_GATEWAY_NAME}" \
  --set global.istio.gateway.namespace="${ISTIO_GATEWAY_NAMESPACE}"

echo "Adding entries to /etc/hosts..."
sudo sh -c 'echo "\n$(minikube ip) runtime-director.kyma.local" >> /etc/hosts'


