#!/usr/bin/env bash

# script populating license type into KEB db
# the input is taken from STDIN in format of instance_id:license_type
# the script processes each entry in order serially and calls KEB service instance update endpoint
#
# usage:
# ./license_type_sync.bash < ers_instances > log

set -euo pipefail

trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT

KEB=localhost:8080
KUBECTL_PORT_FORWARD_KEB=true
READ_ONLY=false

for i in "$@"; do
    case $i in
        -k=*|-keb=*|--keb=*)
            KEB="${i#*=}"
            shift
            ;;
        -k|-keb|--keb)
            KEB="$2"
            shift 2
            ;;
        -p=*|-port-forward=*|--port-forward=*)
            KUBECTL_PORT_FORWARD_KEB="${i#*=}"
            shift
            ;;
        -p|-port-forward|--port-forward)
            KUBECTL_PORT_FORWARD_KEB="$2"
            shift 2
            ;;
        -r=*|-readonly=*|--readonly=*)
            READ_ONLY="${i#*=}"
            shift
            ;;
        -r|-readonly|--readonly)
            READ_ONLY="$2"
            shift 2
            ;;
    esac
done

echo "KEB address: $KEB"
echo "kubectl port-forward KEB: $KUBECTL_PORT_FORWARD_KEB"
echo ""

function patch_license() {
    local instance=$1
    local license_type=${2//$'\n'/}
    echo "patching instance '$instance' to license type '$license_type'"
    if [[ $READ_ONLY == true ]]; then
        echo "readonly"
    else
        curl -XPATCH "$KEB/oauth/v2/service_instances/$instance?accepts_incomplete=true" -i \
            -H "X-Broker-API-Version: 2.14" \
            -H "Content-Type: application/json" \
            --data-binary @- << EOF
{
   "service_id":"47c9dcbf-ff30-448e-ab36-d3bad66ba281",
   "context":{
       "license_type": "$license_type"
   }
}
EOF
    fi
}

if [[ "$KUBECTL_PORT_FORWARD_KEB" == true ]]; then
    echo "Starting kubectl port-forward for KEB and sleeping for a little bit"
    kubectl port-forward -nkcp-system deployment/kcp-kyma-environment-broker 8080:8080 &
    sleep 5
    echo "continue"
    echo ""
fi

while read line; do
    echo reading $line
    readarray -d , -t split <<< ${line}
    patch_license "${split[0]}" "${split[6]}"
done
