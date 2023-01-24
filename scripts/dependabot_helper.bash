#!/usr/bin/env bash

# Dependabot is enabled for KEB, however, after making pre-main-kcp-cli check run for KEB as part of 
# https://github.com/kyma-project/test-infra/pull/5776, the bot is struggling to produce PRs that 
# would pass the pre-main-kcp-cli check. The discussion about long term path is currently under
# https://github.com/kyma-project/control-plane/issues/1929 and this script is intended as a temporary
# remediation until the consensus what to do in future is reached.

set -xeuo pipefail

# list open PRs from dependabot touching KEB go modules
prs=( $(gh pr list --json number,author,title --jq '.[] | select(.author.login == "app/dependabot") | select(.title | endswith("/components/kyma-environment-broker")) | .number') )
body="/lgtm
/approve"

git worktree prune
if [ ! -d /tmp/keb ]; then
    git worktree add /tmp/keb
fi
cd /tmp/keb
DIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

# iterate over each PR, run go mod tidy under the KCP CLI dir, commit, push
git checkout main
git pull origin --rebase
for pr in "${prs[@]}"; do
    is_draft=$(gh pr view ""${pr} --json isDraft --jq '.isDraft')
    if [[ "$is_draft" == true ]]; then
        continue
    fi
    gh pr checkout "${pr}"
    (
        while true; do
            mergeable=$(gh pr view --json mergeable | jq --raw-output '.mergeable')
            case "${mergeable}" in
                MERGEABLE)
                    break
                    ;;
                *)
                    state=$(gh pr view ${pr} --json state | jq --raw-output '.state')
                    if [[ "$state" == CLOSED ]]; then
                        echo "pr $pr has been closed, no longer required"
                        break
                    fi
                    echo "pr ${pr} has status ${mergeable}, waiting"
                    sleep 10
                    ;;
            esac
        done
        pwd
        echo "$DIR/../tools/cli"
        cd "$DIR/../tools/cli"
        go mod tidy
        if [[ -n "$(git diff)" ]]; then
            git commit -am "KCP CLI go mod tidy"
            git push
            sleep 5
            gh pr review "${pr}" --approve --body "${body}"
        else
            status=$(gh pr view --json reviewDecision | jq --raw-output '.reviewDecision')
            if [[ "$status" == 'REVIEW_REQUIRED' ]]; then
                gh pr review "${pr}" --approve --body "${body}"
            fi
        fi
        while true; do
            state=$(gh pr view "${pr}" --json state | jq --raw-output '.state')
            case "$state" in
                MERGED)
                    break
                    ;;
                *)
                    echo "PR ${pr} not merged yet"
                    sleep 5
                    ;;
            esac
        done
    )
done
