#!/usr/bin/env bash
#find . -type f -exec sed -i '' -e 's/github.com\/kyma-project\/control-plane\/components\/kyma-environment-broker/github.com\/kyma-project\/kyma-environment-broker/g' {} +
find . -type f -print0 | xargs -0 perl -pi -e 's#github.comkyma-projectkyma-environment-broker#github.com\kyma-project\kyma-environment-broker#g'
