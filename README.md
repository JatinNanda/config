# config

Portable macOS dotfiles: zsh, tmux, vim/neovim, git, and Claude Code. One command sets up a fresh machine.

## Install

```sh
git clone https://github.com/JatinNanda/config ~/code/config
cd ~/code/config
./install.sh
```

`install.sh` is idempotent (safe to re-run). It installs tools via Homebrew, sets up oh-my-zsh / tpm / Vundle / rust / nvm, then symlinks every config back to this repo. Existing real files are backed up to `*.bak.<timestamp>` before linking.

## Layout

| Path              | Symlinks to |
|-------------------|-------------|
| `home/`           | `~/.zshrc`, `~/.bash_profile`, `~/.profile`, `~/.zshenv`, `~/.zprofile`, `~/.gitconfig`, `~/.tmux.conf`, `~/.vimrc` |
| `nvim/`           | `~/.config/nvim/` |
| `claude/`         | `~/.claude/` (CLAUDE.md, settings.json, keybindings.json, hooks/) |
| `bin/`            | `~/.local/bin/` (worktree helpers) |
| `zsh/`            | oh-my-zsh theme |
| `iterm/`          | iTerm2 color preset (imported manually) |
| `fonts/`          | copied to `~/Library/Fonts` |

Because everything is symlinked, editing a config on the machine edits the repo directly. Commit and push to sync; no export step.

## Secrets and per-machine settings

Nothing secret lives in this repo. `~/.zshrc` sources `~/.zshrc.local` if present, so put API tokens, per-job env vars, and machine-specific overrides there. It is gitignored (`*.local`).

## Notable shell helpers

- `wt <feature>` — create an isolated git worktree in a new tmux window (Claude left, shell right)
- `cdw` / `wtl` — fzf navigator / overview across worktrees
- `clone [org]` — fzf-pick and clone a repo (defaults to `$GH_ORG`, else your own repos)
- `cdf` / `cdr` — jump to a `~/code` project / to the current repo root
- `co`, `gnewb`, `gbasemaster`, `gresetmaster` — git branch helpers
