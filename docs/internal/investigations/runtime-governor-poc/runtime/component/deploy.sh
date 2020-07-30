#!/usr/bin/env bash

set -o errexit

echo "Creating 'cp-poc' namespace..."
kubectl create ns cp-poc || true

echo "Installing Agent chart..."
helm upgrade -i cp-poc-agent -n cp-poc ./chart
