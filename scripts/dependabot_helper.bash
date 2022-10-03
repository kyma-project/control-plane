#!/usr/bin/env bash

# Dependabot is enabled for KEB, however, after making pre-main-kcp-cli check run for KEB as part of 
# https://github.com/kyma-project/test-infra/pull/5776, the bot is struggling to produce PRs that 
# would pass the pre-main-kcp-cli check. The discussion about long term path is currently under
# https://github.com/kyma-project/control-plane/issues/1929 and this script is intended as a temporary
# remediation until the consensus what to do in future is reached.

set -xeuo pipefail

DIR=$(dirname "${BASH_SOURCE[0]}")/..

# list open PRs from dependabot touching KEB go modules
prs=( $(gh pr list --json number,author,title --jq '.[] | select(.author.login == "dependabot") | select(.title | startswith("gomod(deps)")) | select(.title | endswith("/components/kyma-environment-broker")) | .number') )
body="/lgtm
/approve"

# iterate over each PR, run go mod tidy under the KCP CLI dir, commit, push
for pr in "${prs[@]}"; do
    gh pr checkout "${pr}"
    (
        cd "$DIR/tools/cli"
        go mod tidy
        if [[ -n "$(git diff)" ]]; then
            git commit -am "KCP CLI go mod tidy"
            git push
            sleep 5
            gh pr review "${pr}" --approve --body "${body}"
        fi
    )
done
