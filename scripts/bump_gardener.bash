#!/usr/bin/env bash

set -xeuo pipefail

DIR_SCRIPT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
DIR=$( realpath "$DIR_SCRIPT/.." )
GARDENER_VERSION=v1.42.0

tmp_dir=$(mktemp -d)
trap '{ rm -rf -- "$tmp_dir"; }' EXIT

cd "$tmp_dir"

git clone --branch=$GARDENER_VERSION --depth 1 https://github.com/gardener/gardener

cd gardener

PKG="components/kyma-environment-broker/internal/gardener"
DEST="$DIR/$PKG"

function c() {
    path=$1
    mkdir -p $DEST/$path
    echo "cp -r ./$path $DEST/$path"
    cp -r ./$path/* $DEST/$path
}

c "pkg/apis/authentication"
c "pkg/apis/authentication/v1alpha1"
c "pkg/apis/core"
c "pkg/apis/core/v1beta1"
c "pkg/client/core/clientset/versioned/scheme"
c "pkg/utils/timewindow"
c "pkg/utils/version"
c "pkg/client/core/clientset/versioned/fake"
c "pkg/client/core/clientset/versioned/typed/core"
c "pkg/client/core/clientset/versioned/typed/core/v1beta1"
c "pkg/client/core/clientset/versioned/typed/core/v1beta1/fake"

find $DIR/components/kyma-environment-broker -name '*.go' -type f -exec sed -i 's|"github.com/gardener/gardener|"github.com/kyma-project/control-plane/'"$PKG"'|' {} +
