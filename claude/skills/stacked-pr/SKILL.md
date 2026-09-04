---
name: stacked-pr
description: Split separable work into a stack of chained draft PRs, each based on the one below it, and keep the stack rebased as review lands. Use when work divides along distinct concerns (migration vs code, refactor vs feature, per-layer), when the user says "stack this", "open a stack", "split this into PRs", or when a change is large enough that one PR would bury the review.
---

# Stacked PRs

One concern per PR, each based on the one below, all drafts. The point is that a reviewer can read PR 2 without re-reading PR 1's diff.

## 1. Decide whether it's a stack at all

Split along **distinct concerns**, not line count. Real seams:

- schema migration, then the code that uses it
- refactor that changes no behavior, then the behavior change on top
- data model, then ingest, then the thing that reads it
- one PR per layer when a feature crosses api / task / web

Not a seam: "this file and that file", or an arbitrary size cut through one idea. A genuinely single-concern change stays one PR no matter how big, and a 40-line change with two concerns can still be worth two.

**Say the split before opening anything.** List the intended PRs in chat, one line each, and let the user redirect. Opening five PRs the user did not expect is worse than asking.

## 2. Branch and worktree per PR

Branches are `stack/<n>-<slug>`, numbered from 1 at the bottom:

```
stack/1-migration-ddl-timeouts     <- base: main
stack/2-unsafe-migration-linter    <- base: stack/1-migration-ddl-timeouts
stack/3-verify-schema-at-boot      <- base: stack/2-unsafe-migration-linter
```

Never check a stack branch out in `~/code/<repo>`. Each gets a worktree:

```bash
git worktree add ~/.worktrees/<repo>/<slug> -b stack/<n>-<slug> <parent-branch>
```

The older `jatin/<name>` chained convention exists in the repo history. Match it when adding to a stack that already uses it, otherwise use the numbered form.

## 3. Each PR stands on its own

Before a PR joins the stack it must build and pass its own tests **on its own base**. A PR that only works once its child lands is a sign the seam is in the wrong place. Do not push a stack that is mid-change or known broken.

## 4. Open them bottom-up

```bash
gh pr create --draft --base <parent-branch> --head stack/<n>-<slug> \
  --title "..." --body-file <body>
```

Bottom-up, because `--base` must already exist on the remote. Draft always, per CLAUDE.md.

## 5. Body format

Use the `Summary` / `Validation` / `Notes` shape (see the pr-description-format memory). The stack pointer goes first in `## Notes`, as full URLs, never bare `#123`:

```markdown
## Notes

- Stack 2 of 4. Based on https://github.com/accrual-dev/epsilon/pull/10563, which is based on `main`.
```

Once every PR exists, go back and add a "Stack" list to the bottom PR so there is one place showing the whole chain. Mark the current one.

## 6. Keeping it rebased

When review changes land on a parent, every descendant needs rebasing onto it, bottom-up:

```bash
cd ~/.worktrees/<repo>/<child-slug>
git rebase --onto <parent-branch> <old-parent-sha>
```

This rewrites the child branch, so pushing it needs `--force-with-lease`, never plain `--force`. These are personal stack branches, so that is fine; ask first if anyone else has pushed to one.

When a parent merges, GitHub retargets its immediate child to the parent's base automatically. Verify with `gh pr view <child> --json baseRefName` rather than trusting it, and rebase the child so its diff does not still carry the parent's commits.

## 7. Report

Per stack, give the user: the ordered list of PRs as full URLs with their titles, what each one contains in a line, which tests ran per PR, and the merge order. If you split differently than proposed in step 1, say so and why.
