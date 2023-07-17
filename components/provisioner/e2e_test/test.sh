#!/usr/bin/env bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
echo "Entering dir: $SCRIPT_DIR"
cd $SCRIPT_DIR

service docker start

sleep 20s

docker run --name some-postgres -e POSTGRES_PASSWORD=somepass -p 5432:5432 --rm -d postgres

sleep 10s

go run ./pgsetup.go

mkdir migrations
cp ../schema-migrator/migrations/provisioner/* ./migrations/
cp ../../resources/kcp/charts/provisioner/migrations/* ./migrations/

export APP_DIRECTOR_URL=https://compass-gateway-auth-oauth.mps.dev.kyma.cloud.sap/director/graphql

export APP_GARDENER_KUBECONFIG_PATH=$GARDENER_KYMA_PROW_KUBECONFIG
export APP_GARDENER_PROJECT=$GARDENER_KYMA_PROW_PROJECT_NAME

export APP_DATABASE_HOST=localhost
export APP_DATABASE_NAME=provisioner
export APP_DATABASE_PASSWORD=somepass
export APP_DATABASE_PORT=5432
export APP_DATABASE_SECRET_KEY=f16854073f716495dc933f3cc16de9ee
export APP_DATABASE_USER=postgres

docker run -v $PWD/migrations:/migrate/migrations/provisioner:ro \
    -e DB_HOST="host.docker.internal" \
    -e DB_NAME=$APP_DATABASE_NAME \
    -e DB_PORT=$APP_DATABASE_PORT \
    -e DB_PASSWORD=$APP_DATABASE_PASSWORD \
    -e MIGRATION_PATH=provisioner \
    -e DIRECTION=up \
    -e DB_SSL=disable \
    europe-docker.pkg.dev/kyma-project/dev/incubator/compass-schema-migrator

go run ../cmd/ &
export PROVISIONER_PID=$!

go test ./

kill -INT $PROVISIONER_PID
wait $PROVISIONER_PID
export CODE=$!
docker kill some-postgres
exit $CODE
