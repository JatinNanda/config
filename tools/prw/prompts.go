package main

import "fmt"

func cleanupPrompt(p *PR) string {
	base := "origin/main"
	ci := "CI is currently green, so do not go looking for failures."
	if p.CI == "FAILURE" || p.CI == "ERROR" {
		ci = "CI is FAILING on this PR. Diagnose and fix it."
	} else if p.CI == "PENDING" || p.CI == "EXPECTED" {
		ci = "CI is still running. Wait for it to settle with `gh pr checks --watch`, then fix it if it fails."
	}

	return fmt.Sprintf(`Clean up %s#%d (branch %s) in this worktree. Two jobs, in order.

1. Strip the comments this PR added.
   Run: strip-comments --base %s --changed
   That removes only comments added on top of the merge-base and keeps
   pre-existing ones plus tooling directives. Do not hand-edit comments it
   leaves behind, and do not touch comments that were already on %s.

2. %s
   Use `+"`gh pr checks %d`"+` to see the failing checks and
   `+"`gh run view <run-id> --log-failed`"+` for the actual output. Fix the
   underlying cause. Never weaken or delete a test to make it pass, and never
   retry a job hoping it flakes green. If a failure is pre-existing on %s or
   is infrastructure rather than this PR, say so and change nothing.

When done: if anything changed, run the repo's formatter and linter, commit
with a message describing what changed and why, and push to %s. If nothing
needed changing, push nothing and say so.

Do not mark the PR ready for review, do not merge it, and do not reply to any
review comments.`,
		p.Repo, p.Number, p.Branch, base, base, ci, p.Number, base, p.Branch)
}
