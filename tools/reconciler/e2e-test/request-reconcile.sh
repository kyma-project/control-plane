#!/usr/bin/env sh

# Description: Initiates Kyma reconciliation requests to reconciler

## ---------------------------------------------------------------------------------------
## Configurations and Variables
## ---------------------------------------------------------------------------------------
set -e

readonly RECONCILER_HOST="http://reconciler-mothership-reconciler.reconciler"

readonly RECONCILE_API="${RECONCILER_HOST}/v1/clusters"
readonly RECONCILE_PAYLOAD_FILE="/tmp/body.json"

## ---------------------------------------------------------------------------------------
## Functions
## ---------------------------------------------------------------------------------------
# Sends HTTP POST request to mothership-reconciler to trigger reconciliation of Kyma
function send_reconciliation_request() {
  echo "Request body:"
  jq '.kubeconfig = "" | .metadata = ""' ${RECONCILE_PAYLOAD_FILE} > temp_body.json
  cat temp_body.json
  echo "sending reconciliation request to mothership-reconciler at: ${RECONCILE_API}"
  response=$(curl --request POST -sL \
                    --url "${RECONCILE_API}"\
                    --data @"${RECONCILE_PAYLOAD_FILE}" )
  echo "Response: ${response}"
  statusURL=$(echo "${response}" | jq -r .statusURL)

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

# Saving status URL in a file
echo $RECONCILE_STATUS_URL > status_url.txt

exit 0