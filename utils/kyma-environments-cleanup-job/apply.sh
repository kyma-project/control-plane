#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

deploymentName=kcp-kyma-environment-broker
namespace=kcp-system
kebContainerName=kyma-environment-broker
cloudsqlProxyContainerName=cloudsql-proxy
brokerUrlEnv=APP_BROKER_URL
host=kyma-env-broker

SCRIPT_BROKER_URL=$(kubectl get deployment $deploymentName -n $namespace -o jsonpath=\
"{.spec.template.spec.containers[?(@.name==\"$kebContainerName\")].env[?(@.name==\"$brokerUrlEnv\")]}" \
| jq -r '.value')
SCRIPT_DOMAIN=${SCRIPT_BROKER_URL#"$host."}

SCRIPT_CLOUDSQL_PROXY_COMMAND=$(kubectl get deployment $deploymentName -n $namespace -o jsonpath=\
"{.spec.template.spec.containers[?(@.name==\"$cloudsqlProxyContainerName\")].command}")

export SCRIPT_BROKER_URL
export SCRIPT_DOMAIN
export SCRIPT_CLOUDSQL_PROXY_COMMAND

envsubst < kyma-environments-cleanup-job.yaml | kubectl apply -f -
