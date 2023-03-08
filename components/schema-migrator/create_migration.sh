#!/usr/bin/env bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

COMPONENT=$1
NAME=$2

for var in COMPONENT NAME; do
    if [ -z "${!var}" ] ; then
        echo "One or more arguments not provided. Usage: ./create_migration [COMPONENT] [NAME]"
        exit 1
    fi
done

DATE="$(date +%Y%m%d%H%M)"
MIGRATIONS_DIR="../../resources/kcp/charts/${COMPONENT}/migrations"
TRANSACTION_STR=$'BEGIN;\nCOMMIT;'
if [ $COMPONENT == "kyma-environment-broker" ] || [ $COMPONENT == "provisioner" ] ; then
    mkdir -p ${MIGRATIONS_DIR}
fi

echo "$TRANSACTION_STR" > "${MIGRATIONS_DIR}/${DATE}_${NAME}.up.sql"
echo "$TRANSACTION_STR" > "${MIGRATIONS_DIR}/${DATE}_${NAME}.down.sql"
