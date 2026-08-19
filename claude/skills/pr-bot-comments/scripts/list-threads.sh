#!/usr/bin/env bash
# Dumps unresolved review threads for one PR as JSON, each tagged bot/human.
# Usage: list-threads.sh <owner/repo> <pr-number>
set -euo pipefail

OWNER="${1%%/*}"
REPO="${1##*/}"
PR="$2"

gh api graphql -F owner="$OWNER" -F repo="$REPO" -F pr="$PR" -f query='
query($owner:String!,$repo:String!,$pr:Int!){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$pr){
      reviewThreads(first:100){
        nodes{
          id isResolved isOutdated path line
          comments(first:20){
            nodes{ author{login __typename} body url createdAt diffHunk }
          }
        }
      }
    }
  }
}' | jq '
  def is_bot: (.__typename? // "") == "Bot" or ((.login? // "") | test("\\[bot\\]$|^(coderabbitai|greptile|sonarcloud|codecov|copilot|sentry|dependabot)"; "i"));
  [ .data.repository.pullRequest.reviewThreads.nodes[]
    | select(.isResolved == false)
    | {
        threadId: .id,
        author: (.comments.nodes[0].author.login // "unknown"),
        kind: (if (.comments.nodes[0].author | is_bot) then "bot" else "human" end),
        path: .path,
        line: .line,
        outdated: .isOutdated,
        url: .comments.nodes[0].url,
        diffHunk: .comments.nodes[0].diffHunk,
        comments: [.comments.nodes[] | {author: (.author.login // "unknown"), body: .body}]
      }
  ]
'
