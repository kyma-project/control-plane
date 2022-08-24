#!/usr/bin/env bash

# Dependabot is enabled for KEB, however, after making pre-main-kcp-cli check run for KEB as part of 
# https://github.com/kyma-project/test-infra/pull/5776, the bot is struggling to produce PRs that 
# would pass the pre-main-kcp-cli check. The discussion about long term path is currently under
# https://github.com/kyma-project/control-plane/issues/1929 and this script is intended as a temporary
# remediation until the consensus what to do in future is reached.

set -xeuo pipefail

DIR=$(dirname "${BASH_SOURCE[0]}")/..

# list open PRs from dependabot touching KEB go modules
prs=( $(gh pr list | awk '/gomod\(deps\).*kyma-environment-broker/{print($1)}') )

# iterate over each PR, run go mod tidy under the KCP CLI dir, commit, push
for pr in "${prs[@]}"; do
    gh pr checkout "${pr}"
    (
        cd "$DIR/tools/cli"
        go mod tidy
        if [[ -n "$(git diff)" ]]; then
            git commit -am "KCP CLI go mod tidy"
            git push
            gh pr review "${pr}" --approve --body "/lgtm\n/approve"
        fi
    )
done
