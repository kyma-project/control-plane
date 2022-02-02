#!/usr/bin/env sh

# Description: Initiates Kyma reconciliation requests to reconciler and waits until Kyma is installed

## ---------------------------------------------------------------------------------------
## Configurations and Variables
## ---------------------------------------------------------------------------------------
set -e

readonly RECONCILER_HOST="http://reconciler-mothership-reconciler.reconciler"
readonly RECONCILER_DELAY=15 # in secs
readonly RECONCILER_STATUS_RETRY=80
readonly RECONCILE_API="${RECONCILER_HOST}/v1/clusters"
readonly RECONCILE_PAYLOAD_FILE="/tmp/body.json"

## ---------------------------------------------------------------------------------------
## Functions
## ---------------------------------------------------------------------------------------

# Waits until Kyma reconciliation is in ready state
function wait_until_kyma_installed() {
  iterationsLeft=${RECONCILER_STATUS_RETRY}
  while : ; do
    reconcileStatusResponse=$(curl -sL "${RECONCILE_STATUS_URL}")
    status=$(echo "${reconcileStatusResponse}" | jq -r .status)
    echo "status: ${status}"

    if [ "${status}" = "ready" ]; then
      echo "Kyma is reconciled"
      exit 0
    fi

    if [ "${status}" = "error" ]; then
      echo "Failed to reconcile Kyma. Exiting"
      exit 1
    fi

    if [ "$iterationsLeft" -le 0 ]; then
      echo "reconcileStatusResponse: ${reconcileStatusResponse}"
      echo "Timeout reached on Kyma reconciliation. Exiting"
      exit 1
    fi

    sleep $RECONCILER_DELAY
    iterationsLeft=$(( iterationsLeft-1 ))
    echo "Waiting for reconciliation to finish, current status: ${status} .... Iterations left:  ${iterationsLeft}"
  done
}

# Sends HTTP POST request to mothership-reconciler to trigger reconciliation of Kyma
function send_reconciliation_request() {
  echo "sending reconciliation request to mothership-reconciler at: ${RECONCILE_API}"

  reconciliationResponse=$(curl --request POST -sL \
       --url "${RECONCILE_API}"\
       --data @"${RECONCILE_PAYLOAD_FILE}")
  echo "Request body:"
  jq '.kubeconfig = "" | .metadata = ""' ${RECONCILE_PAYLOAD_FILE} > temp_body.json
  cat temp_body.json
  statusURL=$(echo "${reconciliationResponse}" | jq -r .statusURL)
  echo "reconciliationResponse: ${reconciliationResponse}"

  export RECONCILE_STATUS_URL="${statusURL}"
}

# Checks if the reconciler returned status url is valid or not
function check_reconcile_status_url() {
  echo "RECONCILE_STATUS_URL: ${RECONCILE_STATUS_URL}"
  if [[ ! $RECONCILE_STATUS_URL ]] || [[ "$RECONCILE_STATUS_URL" == "null" ]]; then
    echo "reconciliation request failed: RECONCILE_STATUS_URL is invalid"
    exit 1
  fi
}

## ---------------------------------------------------------------------------------------
## Execution steps
## ---------------------------------------------------------------------------------------

# Install curl and jq
echo "Installing curl and jq to the environment"
apk --no-cache add curl jq

# Send reconciliation http request to mothership-reconciler
send_reconciliation_request

# Check if reconcile status url is valid
check_reconcile_status_url

# Wait until Kyma is installed
wait_until_kyma_installed

echo "reconcile-kyma completed"