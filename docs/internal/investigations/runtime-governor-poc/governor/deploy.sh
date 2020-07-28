#!/usr/bin/env bash

set -o errexit

LOCAL_ENV=${LOCAL_ENV:-true}
DOMAIN=${DOMAIN:-kyma.local}
ISTIO_GATEWAY_NAME=${ISTIO_GATEWAY_NAME:-compass-istio-gateway}
ISTIO_GATEWAY_NAMESPACE=${ISTIO_GATEWAY_NAMESPACE:-compass-system}

echo "=== CONFIGURATION ==="
echo "Local env: ${LOCAL_ENV}"
echo "Domain: ${DOMAIN}"
echo "Istio Gateway: ${ISTIO_GATEWAY_NAMESPACE}/${ISTIO_GATEWAY_NAME}"
echo "====================="

echo "Creating 'kcp-poc' namespace..."
kubectl create ns kcp-poc || true

echo "Installing Runtime Director chart..."
helm install kcp-poc -n kcp-poc ./chart \
  --set global.ingress.domainName="${DOMAIN}" \
  --set global.istio.gateway.name="${ISTIO_GATEWAY_NAME}" \
  --set global.istio.gateway.namespace="${ISTIO_GATEWAY_NAMESPACE}"

if [[ "$LOCAL_ENV" == "true" ]]; then
  echo "Adding entries to /etc/hosts..."
  sudo sh -c 'echo "\n$(minikube ip) runtime-governor.kyma.local" >> /etc/hosts'
fi


