#!/usr/bin/env sh

# Description: Returns Kyma reconciliation status

## ---------------------------------------------------------------------------------------
## Execution steps
## ---------------------------------------------------------------------------------------
export RECONCILE_STATUS_URL=$(cat status_url.txt)
status=$(curl -sL "$RECONCILE_STATUS_URL" | jq -r .status)
echo "$status"