---
name: pr-bot-comments
description: Find open PRs carrying unresolved review comments, let the user pick which ones to work, then fix and resolve the bot threads. Use when the user says things like "clear the bot comments", "any PRs with review comments?", "resolve the coderabbit/copilot/greptile feedback", "go through my open PR comments", or names a PR and asks to handle its review comments.
---

# Resolving PR review comments

Two audiences, two rules. They are not negotiable and they come from CLAUDE.md:

- **Bot comments**: never reply. Fix the issue if it is legitimate; leave the code alone if it is not. Resolve the thread either way.
- **Human comments**: fix the issue if it makes sense. Never reply, never resolve. The user answers their own reviewers.
- **Your own comments** (`kind: "mine"`): ignore them. Jatin's comments on his own PRs are context written for the reviewer, not feedback to act on. No fix, no reply, no resolve, and they never count toward the human total. A thread he started counts as human only once a real reviewer replies in it.

Anything ambiguous is human.

## 1. Find candidates

```bash
~/.claude/skills/pr-bot-comments/scripts/find-prs.sh
```

Prints one row per open PR with unresolved threads: repo, number, bot count, human count, draft/ready, title, URL. Sorted by bot count.

Pass extra GitHub search qualifiers to narrow it: `find-prs.sh repo:accrual-dev/epsilon`, `find-prs.sh is:draft`. The base query is always `is:pr is:open author:@me`.

Skip this step if the user already named a PR.

## 2. Let the user select

Show the table, then use AskUserQuestion with `multiSelect: true` to pick PRs. One option per PR, labelled `repo#number`, with the bot/human counts and title in the description. Cap it at 4 options, most bot comments first; if there are more candidates, say so in the surrounding text and offer to go again after.

Do not start fixing before the user picks. Selection is the whole point of the skill.

## 3. Read the threads

```bash
~/.claude/skills/pr-bot-comments/scripts/list-threads.sh <owner/repo> <pr-number>
```

Returns unresolved threads as JSON, each with `threadId`, `kind` (`bot` or `human`), `author`, `path`, `line`, `outdated`, `diffHunk`, and the full comment chain.

Check out the PR branch before editing (`gh pr checkout <number>` in the right repo). Read the real file at the real line; a `diffHunk` is a snapshot and the branch may have moved past it. Threads marked `outdated: true` usually point at code that no longer exists, so verify before acting and resolve without a code change when the concern is already gone.

## 4. Work the bot threads

For each `kind: "bot"` thread, judge the comment on its merits:

- **Legitimate**: fix it. Small and mechanical is the common case (a typo, a missing guard, a wrong default). If a fix is large, risky, or touches something outside the PR's scope, stop and ask the user rather than doing it silently.
- **Wrong, or a style opinion you disagree with, or already handled elsewhere**: change nothing.

Resolve every bot thread you processed, fixed or not:

```bash
~/.claude/skills/pr-bot-comments/scripts/resolve-thread.sh <threadId> [<threadId>...]
```

Never post a reply comment on a bot thread. Not an acknowledgement, not an explanation of the fix, not a "good catch". The resolve is the entire response.

Top-level PR comments from bots (CI summaries, coverage reports) are not review threads and cannot be resolved. Leave them alone.

## 5. Work the human threads

Fix what makes sense, exactly as above. Then stop: leave the thread open and unanswered. If you decided against a human's suggestion, do not argue it on the PR, tell the user in chat so they can reply themselves.

## 6. Get the tests green

Every fix you made has to survive the PR's own test suite. Do not push red.

Work the smallest suite that covers the change and widen only if it stays green. In `accrual-dev/epsilon` that means:

```bash
pnpm --filter <package> exec vitest run <path/to/file.test.ts>   # one file, while iterating
pnpm test:api-srv:workflow-tests                                  # Temporal workflow projects
pnpm test:api-srv                                                 # full api-srv, slowest
```

Then loop: run, read the first real failure, fix it, run again. Keep going while each pass is making progress.

Stop and report instead of looping when:

- The same failure survives three attempts. You are guessing; say so and hand it back.
- A test fails on `main` too. Pre-existing breakage is not yours to fix inside a comment-resolution pass, so confirm against the base branch before assuming your change caused it.
- The suite cannot run at all: dependencies will not install, a database or other service is missing, credentials are absent. Say plainly which suite could not run and why, and do not claim the fix is verified.
- Getting green would mean editing the test to match the code rather than the other way round. Never loosen an assertion to force a pass; that is a finding to report, not a fix to apply.

"Within reason" is the governing phrase: a handful of focused iterations, not an open-ended grind. If deps alone take many minutes, run the narrowest suite that proves the change and say what you skipped.

## 7. Commit and report

Commit the fixes to the PR branch and push. Follow the usual commit rules from CLAUDE.md.

Then report per PR:

- bot threads fixed, with a one-line description of each fix
- bot threads resolved without a change, and why
- human threads you fixed, left open, and what the user may want to say back
- which test suites you ran and their result, or plainly which ones you could not run and why
- anything you skipped and the reason
- the PR URL

Keep it a summary. The user is deciding what to reply to, so give them what they need for that and nothing more.
