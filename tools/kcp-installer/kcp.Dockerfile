# The base version of the Kyma Operator that will be used to build Control Plane Installer
ARG INSTALLER_VERSION="12e41ab5"
ARG INSTALLER_DIR=eu.gcr.io/kyma-project
FROM $INSTALLER_DIR/kyma-operator:$INSTALLER_VERSION

LABEL source="git@github.com:kyma-project/kyma.git"

COPY /resources /kyma/injected/resources
