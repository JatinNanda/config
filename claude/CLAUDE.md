# Global Claude Permissions

## Automatically Allowed (no confirmation needed)

- Reading any files on the filesystem
- Search/grep commands (`grep`, `rg`, `find`, `locate`, etc.)
- Non-mutating queries (GET requests, read-only CLI commands, `describe`, `list`, `get`, `show`)
- Making code changes (editing, creating, or deleting source files)
- Running tests, linters, formatters, and build commands

## Code Style

- NO COMMENTS. Do not write comments or docstrings. Not for intent, not for invariants, not for gotchas, not for a non-obvious why. There is no "but this one is genuinely useful" exception. That judgment call is exactly what produces the comments this rule exists to stop, so treat the ban as absolute.
- If code needs explaining, explain it in chat, the commit message, or the PR description. Never in the source.
- The only comments allowed are ones that are load-bearing for tooling, because they change behavior rather than describe it: `// @ts-expect-error`, `// eslint-disable-next-line`, `//go:build`, `/// <reference>`, `# noqa`, `# type: ignore`, `# shellcheck disable`, shebangs, and licence or SPDX headers.
- Do not remove or rewrite comments that were already there, unless the change made them wrong.
- Delete commented-out code, never leave it behind.
- Never use an em-dash, in code, comments, docs, or prose. Use commas, parentheses, or separate sentences.

Iteration history belongs in the commit message and the PR description, and nowhere else. The code is the artifact, not a changelog or a work log.

This is enforced mechanically as well as by instruction: a `PostToolUse` hook runs `strip-comments` on every file written on this machine, removing comments added on top of git HEAD while leaving pre-existing ones and the tooling directives above intact.

## Commit Behavior

- Once code changes are complete and verified, branch, commit, push, and open a draft PR without waiting to be asked. Do not leave finished work sitting uncommitted.
- Never commit to the default branch. If on it, create a branch first.
- Do not push or open a PR while the work is still mid-change or known broken. "Complete and verified" means tests, linters, and builds have been run, or it has been stated plainly which ones could not be run and why.
- After creating or amending a commit, output the GitHub PR link: `https://github.com/<org>/<repo>/compare/<branch>?expand=1`

## PR Behavior

- Whenever a PR is mentioned, write it as a full clickable URL (`https://github.com/<org>/<repo>/pull/<number>`), never a bare `#123`, `PR 123`, or plain number. This applies everywhere: chat replies, summaries, commit messages, and PR bodies. The same goes for a PR that was just created, and for any PR referred to later in the conversation.
- Create PRs as drafts (`--draft`) unless told otherwise. Draft is what makes opening one the safe default: it is somewhere to put the work, not a request for review.
- Check for a PR template (`.github/pull_request_template.md` or `.github/PULL_REQUEST_TEMPLATE/*.md`) and fill in all sections based on the changes.
- Write a real PR description. If the diff renders misleadingly (moved code showing up as unchanged context, for example), show the before and after in the body.
- Marking a PR ready for review, merging, and force-pushing to a shared branch still require asking.
- When work splits along distinct concerns (migration vs code, refactor vs behavior change, per-layer), default to a stack of chained draft PRs rather than one large PR, and use the `stacked-pr` skill. Propose the split in chat before opening anything. A genuinely single-concern change stays one PR however big it is.

## Responding to PR Review Comments

- **Bot comments** (Copilot, CodeRabbit, Greptile, Codecov, linters, any non-human reviewer): never post a reply. Fix the issue if it is legitimate, leave the code alone if it is not, and resolve the thread either way. No acknowledgement, no "good catch", no comment explaining the fix. Resolving is the entire response.
- **Human comments**: fix the issue if it makes sense, then leave the thread open and unanswered so I can respond myself. Never reply, never resolve. If you disagree with a human reviewer, say so in chat rather than on the PR.
- When it is unclear whether a commenter is a bot, treat it as human and leave the thread open. The GraphQL `author.__typename == "Bot"` is the reliable signal, a `[bot]` login suffix is not always present.
- The `pr-bot-comments` skill does this end to end: finds open PRs with unresolved threads, lets me select which to work, then fixes and resolves.

## Require Explicit User Confirmation

- Connecting to any database (read or write)
- Force-pushing to, or resetting, a shared branch
- Mutating cloud CLI commands (create/update/delete/put/apply against real infrastructure)
- Running IaC apply/destroy (`terraform apply`, `terragrunt apply`, `pulumi up`, etc.)
- Any operation that modifies shared or production infrastructure
