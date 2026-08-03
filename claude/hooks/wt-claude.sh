#!/usr/bin/env bash
# launch claude in a worktree pane: name a fresh session on first run,
# resume the most recent one when tmux-resurrect re-runs us after a reboot
name="$1"

# claude encodes cwd by replacing / and . with -
project_dir="$HOME/.claude/projects/$(pwd | sed 's|[/.]|-|g')"

if compgen -G "$project_dir/*.jsonl" > /dev/null; then
  exec claude --continue
else
  exec claude -n "$name"
fi
