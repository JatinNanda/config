#!/usr/bin/env bash
# wrapper that runs an interactive shell in a worktree and cleans up on exit
WORKTREE_PATH="$1"
REPO_ROOT="$2"

_cleaned=0
cleanup() {
  [[ $_cleaned -eq 1 ]] && return
  _cleaned=1
  git -C "$REPO_ROOT" worktree remove --force "$WORKTREE_PATH" 2>/dev/null
  git -C "$REPO_ROOT" worktree prune 2>/dev/null
}

trap cleanup EXIT SIGHUP SIGTERM

cd "$WORKTREE_PATH" || exit 1
"${SHELL:-zsh}"
