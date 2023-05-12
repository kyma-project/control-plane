#!/bin/bash
cd "$(dirname "$0")"
LOCAL_BIN=$(pwd)/bin
mkdir -p $LOCAL_BIN
GOBIN=$LOCAL_BIN go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
OUTPUT=$($LOCAL_BIN/setup-envtest use --bin-dir $LOCAL_BIN -p path latest)
echo $OUTPUT