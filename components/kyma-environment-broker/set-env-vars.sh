#!/bin/bash

LOCALBIN=`pwd`/bin
ENVTEST=${LOCALBIN}/setup-envtest
export KUBEBUILDER_ASSETS="$(${ENVTEST} use --bin-dir ${LOCALBIN} -p path)"