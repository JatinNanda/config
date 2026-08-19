#!/usr/bin/env bash
# Lists open PRs with unresolved review threads, split by bot vs human author.
# Usage: find-prs.sh [extra search qualifiers, e.g. "repo:owner/name" or "author:someone"]
set -euo pipefail

ME=$(gh api user -q .login)
QUALIFIERS="${*:-}"
QUERY="is:pr is:open author:@me ${QUALIFIERS}"

gh api graphql -F q="$QUERY" -f query='
query($q:String!){
  search(query:$q, type:ISSUE, first:50){
    nodes{
      ... on PullRequest{
        number title url isDraft
        repository{nameWithOwner}
        reviewThreads(first:100){
          nodes{
            isResolved isOutdated
            comments(first:10){nodes{author{login __typename}}}
          }
        }
      }
    }
  }
}' | jq -r --arg me "$ME" '
  def is_bot: (.__typename? // "") == "Bot" or ((.login? // "") | test("\\[bot\\]$|^(coderabbitai|greptile|sonarcloud|codecov|copilot|sentry|dependabot)"; "i"));
  def reviewers: [ .comments.nodes[].author | select(is_bot | not) | select((.login // "") != $me) ] | length;
  [ .data.search.nodes[] | select(.number)
    | . as $pr
    | ($pr.reviewThreads.nodes | map(select(.isResolved == false))) as $open
    | {
        repo: $pr.repository.nameWithOwner,
        number: $pr.number,
        title: $pr.title,
        url: $pr.url,
        draft: $pr.isDraft,
        bot: ($open | map(select(.comments.nodes[0].author | is_bot)) | length),
        human: ($open | map(select((.comments.nodes[0].author | is_bot) | not) | select(reviewers > 0)) | length)
      }
  ]
  | map(select(.bot + .human > 0))
  | sort_by(-.bot)
  | .[]
  | [.repo, .number, .bot, .human, (if .draft then "draft" else "ready" end), .title, .url]
  | @tsv
' | (printf "REPO\tPR\tBOT\tHUMAN\tSTATE\tTITLE\tURL\n"; cat) | column -t -s $'\t'
