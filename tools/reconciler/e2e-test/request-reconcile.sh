#!/usr/bin/env sh

# Description: Initiates Kyma reconciliation requests to reconciler

## ---------------------------------------------------------------------------------------
## Configurations and Variables
## ---------------------------------------------------------------------------------------
set -e

readonly RECONCILER_HOST="http://reconciler-mothership-reconciler.reconciler"

readonly RECONCILE_API="${RECONCILER_HOST}/v1/clusters"
readonly RECONCILE_PAYLOAD_TEMPLATE="/tmp/body.json"
readonly RECONCILE_PAYLOAD_FILE="/tmp/body.json.tmp"

## ---------------------------------------------------------------------------------------
## Functions
## ---------------------------------------------------------------------------------------
# Sends HTTP POST request to mothership-reconciler to trigger reconciliation of Kyma
function send_reconciliation_request() {
  echo "sending reconciliation request to mothership-reconciler at: ${RECONCILE_API}"
  statusURL=$(curl --request POST -sL \
       --url "${RECONCILE_API}"\
       --data @"${RECONCILE_PAYLOAD_FILE}" | jq -r .statusURL)

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

function check_kyma_upgrade_version() {
  if [ -z "${KYMA_UPGRADE_VERSION}" ] ; then
    echo "ERROR: KYMA_UPGRADE_VERSION is not set"
    exit 1
  fi
  echo "KYMA_UPGRADE_VERSION is set to: ${KYMA_UPGRADE_VERSION}"
}

function render_template() {
  echo "Render body.json template"
  envsubst < ${RECONCILE_PAYLOAD_TEMPLATE} > ${RECONCILE_PAYLOAD_FILE}
  echo "Rendered template:"
  cat ${RECONCILE_PAYLOAD_FILE}
}

## ---------------------------------------------------------------------------------------
## Execution steps
## ---------------------------------------------------------------------------------------
# Install curl and jq
echo "Installing curl and jq to the environment"
apk --no-cache add curl jq

# Check if upgrade version has been set
check_kyma_upgrade_version

# Renders body template with environment variables
render_template

# Send reconciliation http request to mothership-reconciler
send_reconciliation_request

# Check if reconcile status url is valid
check_reconcile_status_url

# Saving status URL in a file
echo $RECONCILE_STATUS_URL > status_url.txt

exit 0