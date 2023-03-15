#!/usr/bin/env bash

set -xeuo pipefail

prs=( $(gh pr list --json number,author,title --jq '.[] | select(.author.login == "app/dependabot") | select(.title | endswith("/components/kyma-environment-broker")) | .number') )
body="/lgtm
/approve"

for pr in "${prs[@]}"; do
    is_draft=$(gh pr view ""${pr} --json isDraft --jq '.isDraft')
    if [[ "$is_draft" == true ]]; then
        continue
    fi
    status=$(gh pr view "${pr}" --json reviewDecision | jq --raw-output '.reviewDecision')
    if [[ "$status" == 'REVIEW_REQUIRED' ]]; then
        gh pr review "${pr}" --approve --body "${body}"
    fi
done
