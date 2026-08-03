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

## Commit Behavior

- Commit or push only when asked. If on the default branch, create a branch first.
- After creating or amending a commit, output the GitHub PR link: `https://github.com/<org>/<repo>/compare/<branch>?expand=1`

## PR Behavior

- Create PRs as drafts (`--draft`) unless told otherwise.
- Check for a PR template (`.github/pull_request_template.md` or `.github/PULL_REQUEST_TEMPLATE/*.md`) and fill in all sections based on the changes.

## Require Explicit User Confirmation

- Connecting to any database (read or write)
- Force-pushing to, or resetting, a shared branch
- Mutating cloud CLI commands (create/update/delete/put/apply against real infrastructure)
- Running IaC apply/destroy (`terraform apply`, `terragrunt apply`, `pulumi up`, etc.)
- Any operation that modifies shared or production infrastructure
