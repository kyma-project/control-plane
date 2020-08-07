#!/usr/bin/env bash
NETWORK="cpnet"

k3d cluster create runtime2 --network $NETWORK --api-port 6445
sleep 10s

kubectl --context=k3d-runtime2 create ns dapr-system
helm --kube-context=k3d-runtime2 repo add dapr https://daprio.azurecr.io/helm/v1/repo
helm --kube-context=k3d-runtime2 upgrade -i dapr dapr/dapr -n dapr-system

kubectl --context=k3d-runtime2 create ns cp-poc
kubectl --context=k3d-runtime2 -n cp-poc apply -f ./runtime/node.yaml
