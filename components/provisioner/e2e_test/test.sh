#!/usr/bin/env bash

LOG_DIR=${ARTIFACTS:-"/var/log"}
set -e

export POSTGRES_CONTAINER="provisioner-psql"
export POSTGRES_NETWORK="provisioner-psql-net"

function ensure_exists {
    if [[ -z ${!1} ]]; then
        echo "$1 is undefined"
        exit 1
    fi
}

function cleanup {
    if docker ps | grep $POSTGRES_CONTAINER; then
        docker kill $POSTGRES_CONTAINER
        docker network rm $POSTGRES_NETWORK
    fi
    if ! [[ -z $PROVISIONER_PID ]]; then
        kill $PROVISIONER_PID || true
        wait $PROVISIONER_PID || true
        export PROVISIONER_CODE=$? || true
        unset PROVISIONER_PID
    fi
}

trap cleanup EXIT

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
echo "Entering dir: $SCRIPT_DIR"
cd $SCRIPT_DIR

if which service; then
    service docker start
fi

printf '\n########## SETTING UP PROVISIONER ##########\n\n'

sleep 20

docker network create $POSTGRES_NETWORK
docker run --name $POSTGRES_CONTAINER --network $POSTGRES_NETWORK -e POSTGRES_PASSWORD=somepass -p 5432:5432 --rm -d postgres

sleep 10


export APP_DIRECTOR_URL=https://compass-gateway-auth-oauth.mps.dev.kyma.cloud.sap/director/graphql
if [[ -z "$APP_DIRECTOR_OAUTH_PATH" ]]; then
    export APP_DIRECTOR_OAUTH_PATH=/compass-director-secret/secret.yaml
fi

if ! [[ -f "$APP_DIRECTOR_OAUTH_PATH" ]]; then
    echo "APP_DIRECTOR_OAUTH_PATH is not set or file doesn't exist $APP_DIRECTOR_OAUTH_PATH"
    exit 1
fi

export APP_GARDENER_KUBECONFIG_PATH=${APP_GARDENER_KUBECONFIG_PATH:-$GARDENER_KYMA_PROW_KUBECONFIG}
export APP_GARDENER_PROJECT=${GARDENER_KYMA_PROW_PROJECT_NAME:-$APP_GARDENER_PROJECT}

ensure_exists APP_GARDENER_KUBECONFIG_PATH
ensure_exists APP_GARDENER_PROJECT

export GARDENER_PROVIDER=${GARDENER_PROVIDER:-gcp}
export GARDENER_SECRET_NAME=${GARDENER_KYMA_PROW_PROVIDER_SECRET_NAME:-$GARDENER_SECRET_NAME}

ensure_exists GARDENER_PROVIDER
ensure_exists GARDENER_SECRET_NAME

export APP_DATABASE_HOST=localhost
export APP_DATABASE_NAME=provisioner
export APP_DATABASE_PASSWORD=somepass
export APP_DATABASE_PORT=5432
# 👇 this "key leak" is fine, the database is only accesible on localhost and has no persistence
export APP_DATABASE_SECRET_KEY=f16854073f716495dc933f3cc16de9ee
export APP_DATABASE_USER=postgres

export APP_PROVISIONING_TIMEOUT_INSTALLATION=90m
export APP_PROVISIONING_TIMEOUT_UPGRADE=90m
export APP_PROVISIONING_TIMEOUT_AGENT_CONFIGURATION=90m
export APP_PROVISIONING_NO_INSTALL_TIMEOUT_AGENT_CONFIGURATION=90m
export APP_PROVISIONING_TIMEOUT_AGENT_CONNECTION=90m
export APP_PROVISIONING_TIMEOUT_CLUSTER_CREATION=90m
export APP_PROVISIONING_NO_INSTALL_TIMEOUT_CLUSTER_CREATION=90m
export APP_PROVISIONING_TIMEOUT_UPGRADE_TRIGGERING=90m

printf '\n########## SETTING UP THE DB ##########\n\n'
go run ./pgsetup.go

printf '\n########## BUILDING SCHEMA-MIGRATOR ##########\n\n'
docker build -t schema-migrator ../../schema-migrator

mkdir -p migrations
cp ../../schema-migrator/migrations/provisioner/* ./migrations/
cp ../../../resources/kcp/charts/provisioner/migrations/* ./migrations/

printf '\n########## MIGRATING THE DB ##########\n\n'
docker run -v $PWD/migrations:/migrate/migrations/provisioner:ro \
    --network $POSTGRES_NETWORK \
    -e DB_HOST=$POSTGRES_CONTAINER \
    -e DB_NAME=$APP_DATABASE_NAME \
    -e DB_PORT=$APP_DATABASE_PORT \
    -e DB_USER=$APP_DATABASE_USER \
    -e DB_PASSWORD=$APP_DATABASE_PASSWORD \
    -e MIGRATION_PATH=provisioner \
    -e DIRECTION=up \
    -e DB_SSL=disable \
    schema-migrator

printf '\n########## SETTING UP PROVISIONER ##########\n\n'
go mod download
go run ../cmd/ | tee "${LOG_DIR}/provisioner.log" &
export PROVISIONER_PID=$!

sleep 60

printf '\n########## RUNNING TESTS ##########\n\n'
go test -timeout 100m -v ./ | tee "${LOG_DIR}/test.log"
export TEST_CODE=$?

cleanup

echo exiting with $TEST_CODE
exit $TEST_CODE
