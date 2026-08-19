# Global Claude Permissions

## Automatically Allowed (no confirmation needed)

- Reading any files on the filesystem
- Search/grep commands (`grep`, `rg`, `find`, `locate`, etc.)
- Non-mutating queries (GET requests, read-only CLI commands, `describe`, `list`, `get`, `show`)
- Making code changes (editing, creating, or deleting source files)
- Running tests, linters, formatters, and build commands

## Code Style

- Keep comments and documentation MINIMAL. A comment must never restate what the code plainly does or anything easily gleaned from reading the code. Only comment the non-obvious: intent, invariants, gotchas, reasons.
- Never use an em-dash, in code, comments, docs, or prose. Use commas, parentheses, or separate sentences.

### Never Document the Iteration

The code is the artifact, not a changelog or a work log. When a change took several attempts, that history is invisible in the final code and it must stay that way. Err on the side of not documenting a change even if it took many iterations to get there.

NEVER write a comment or docstring that:

- Describes the change instead of the code: `# Now uses batching`, `# Switched to async`, `# Refactored to use X`, `# Updated to handle Y`.
- References a previous state of the code: `# Previously we did X`, `# Was O(n^2), now O(n)`, `# Old version used a dict`, `# No longer needs the lock`.
- Narrates the debugging journey: `# This was tricky`, `# Took a few tries`, `# Fixed the race condition`, `# Handles the edge case we hit`.
- Marks something as new, fixed, added, or removed: `# NEW:`, `# FIX:`, `# ADDED:`, `# Removed the old path`.
- Justifies the work to a reader: `# Simplified for clarity`, `# Cleaner approach`, `# More robust now`.
- Adds a docstring to a function that had none, purely because the function was touched.
- Expands an existing docstring to announce new behavior that the signature and body already show.

Rules:

- Default to NOT adding a comment or docstring. Add one only when a reader seeing the code for the first time, with zero knowledge of this session, would be confused without it.
- Write every comment as if the code had always looked this way. If a comment only makes sense to someone who saw the previous version, delete it.
- Do not explain why a bug fix works unless the underlying constraint is genuinely non-obvious and would invite someone to reintroduce the bug. Even then, state the constraint, not the fix: `# API rejects batches over 500`, never `# Fixed: was sending 1000 at a time`.
- Delete commented-out old code, never leave it behind.
- Do not remove or rewrite comments that were already there, unless the change made them wrong.
- Iteration history belongs in the commit message and the PR description, and nowhere else.

When in doubt, leave it out. Under-documenting is the correct failure mode here.

## Commit Behavior

- Once code changes are complete and verified, branch, commit, push, and open a draft PR without waiting to be asked. Do not leave finished work sitting uncommitted.
- Never commit to the default branch. If on it, create a branch first.
- Do not push or open a PR while the work is still mid-change or known broken. "Complete and verified" means tests, linters, and builds have been run, or it has been stated plainly which ones could not be run and why.
- After creating or amending a commit, output the GitHub PR link: `https://github.com/<org>/<repo>/compare/<branch>?expand=1`

## PR Behavior

- Create PRs as drafts (`--draft`) unless told otherwise. Draft is what makes opening one the safe default: it is somewhere to put the work, not a request for review.
- Check for a PR template (`.github/pull_request_template.md` or `.github/PULL_REQUEST_TEMPLATE/*.md`) and fill in all sections based on the changes.
- Write a real PR description. If the diff renders misleadingly (moved code showing up as unchanged context, for example), show the before and after in the body.
- Marking a PR ready for review, merging, and force-pushing to a shared branch still require asking.

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
