package main

import "fmt"

func cleanupPrompt(p *PR) string {
	base := "origin/main"
	ci := "CI is green right now, so do not go hunting for failures. Re-check after the rebase and fix anything the rebase broke."
	if p.CI == "FAILURE" || p.CI == "ERROR" {
		ci = "CI is FAILING on this PR. Diagnose it and fix the cause."
	} else if p.CI == "PENDING" || p.CI == "EXPECTED" {
		ci = "CI is still running. Let it settle with `gh pr checks --watch`, then fix it if it goes red."
	}

	return fmt.Sprintf(`Clean up %s#%d (branch %s) in this worktree. Three jobs, in order.

1. Rebase onto %s.
   Fetch first. If the rebase hits a conflict you cannot resolve with real
   confidence, run `+"`git rebase --abort`"+`, leave the branch exactly as it
   was, and report FAILED. Do not guess at a resolution, do not take one side
   wholesale to make it apply, and never force-push anything you did not
   rebase cleanly yourself.

2. Strip the comments this PR added.
   Run: strip-comments --base %s --changed
   That removes only comments added on top of the merge-base and keeps
   pre-existing ones plus tooling directives. Do not hand-edit whatever it
   leaves behind, and do not touch comments that were already on %s.

3. %s
   Use `+"`gh pr checks %d`"+` for the failing checks and
   `+"`gh run view <run-id> --log-failed`"+` for the actual output. Fix the
   underlying cause. Never weaken, skip, or delete a test to get green, and
   never re-run a job hoping it flakes. If the failure is pre-existing on %s
   or is infrastructure rather than this PR, change nothing and say so.

When the work is done: run the repo's formatter and linter, commit with a
message describing what changed and why, and push. A rebase you performed
yourself needs --force-with-lease; never a bare --force. If nothing needed
changing, push nothing.

Do not mark the PR ready for review, do not merge it, and do not reply to any
review comment.

Finally, and this is required: write a single line to the file named by the
$PRW_RESULT environment variable. Start it with OK: if every job above
succeeded, or FAILED: if any of them did not, followed by a short summary. If
you rebased but could not fix CI, that is FAILED. Write this line even if you
changed nothing, and write it last, after everything else is finished.`,
		p.Repo, p.Number, p.Branch, base, base, base, ci, p.Number, base)
}
