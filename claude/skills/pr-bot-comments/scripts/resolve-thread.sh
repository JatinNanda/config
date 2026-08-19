#!/usr/bin/env bash
# Marks a review thread resolved. Never posts a reply.
# Usage: resolve-thread.sh <threadId> [threadId...]
set -euo pipefail

for id in "$@"; do
  gh api graphql -F id="$id" -f query='
    mutation($id:ID!){
      resolveReviewThread(input:{threadId:$id}){ thread{ id isResolved } }
    }' | jq -r '.data.resolveReviewThread.thread | "\(.id) resolved=\(.isResolved)"'
done
