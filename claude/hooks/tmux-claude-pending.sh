#!/bin/bash
# Set or clear the @claude_pending flag on the current tmux window
# Usage: tmux-claude-pending.sh [0|1]
[ -z "$TMUX_PANE" ] && exit 0
tmux set-window-option -t "$TMUX_PANE" @claude_pending "${1:-1}"
