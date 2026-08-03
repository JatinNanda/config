#!/usr/bin/env bash
# One-click setup: installs tools and symlinks every config into place.
# Idempotent — safe to re-run. Existing real files are backed up to *.bak.<ts>.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OMZ="$HOME/.oh-my-zsh"
ZSH_CUSTOM="${ZSH_CUSTOM:-$OMZ/custom}"

log()  { printf "\033[1;33m==> %s\033[0m\n" "$*"; }
warn() { printf "\033[1;31m!!  %s\033[0m\n" "$*"; }

# symlink src -> dst, backing up any pre-existing real file
link() {
  local src="$1" dst="$2"
  mkdir -p "$(dirname "$dst")"
  [ -L "$dst" ] && rm "$dst"
  [ -e "$dst" ] && mv "$dst" "$dst.bak.$(date +%s)"
  ln -s "$src" "$dst"
}

clone_if_missing() {
  local url="$1" dst="$2"
  [ -d "$dst" ] || git clone --depth 1 "$url" "$dst"
}

# ---- Homebrew + packages ----------------------------------------------------
if [ "$(uname)" = "Darwin" ]; then
  if ! command -v brew >/dev/null 2>&1; then
    log "Installing Homebrew"
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  fi
  eval "$(/opt/homebrew/bin/brew shellenv 2>/dev/null || /usr/local/bin/brew shellenv)"
  log "Installing Brewfile packages"
  brew bundle --file "$REPO/Brewfile"
else
  warn "Not macOS — skipping Homebrew. Install git/zsh/tmux/neovim/fzf/ag/jq/go/node yourself."
fi

# ---- oh-my-zsh + plugins ----------------------------------------------------
if [ ! -d "$OMZ" ]; then
  log "Installing oh-my-zsh"
  RUNZSH=no CHSH=no KEEP_ZSHRC=yes \
    sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
fi
log "Installing zsh plugins"
clone_if_missing https://github.com/zsh-users/zsh-autosuggestions      "$ZSH_CUSTOM/plugins/zsh-autosuggestions"
clone_if_missing https://github.com/zsh-users/zsh-syntax-highlighting  "$ZSH_CUSTOM/plugins/zsh-syntax-highlighting"
clone_if_missing https://github.com/joshskidmore/zsh-fzf-history-search "$ZSH_CUSTOM/plugins/zsh-fzf-history-search"

# ---- tmux + vim plugin managers ---------------------------------------------
log "Installing tpm and Vundle"
clone_if_missing https://github.com/tmux-plugins/tpm       "$HOME/.tmux/plugins/tpm"
clone_if_missing https://github.com/VundleVim/Vundle.vim   "$HOME/.vim/bundle/Vundle.vim"

# ---- symlinks ---------------------------------------------------------------
log "Linking dotfiles"
link "$REPO/home/zshrc"        "$HOME/.zshrc"
link "$REPO/home/bash_profile" "$HOME/.bash_profile"
link "$REPO/home/profile"      "$HOME/.profile"
link "$REPO/home/zshenv"       "$HOME/.zshenv"
link "$REPO/home/zprofile"     "$HOME/.zprofile"
link "$REPO/home/gitconfig"    "$HOME/.gitconfig"
link "$REPO/home/tmux.conf"    "$HOME/.tmux.conf"
link "$REPO/home/vimrc"        "$HOME/.vimrc"

link "$REPO/nvim/init.vim"          "$HOME/.config/nvim/init.vim"
link "$REPO/nvim/coc-settings.json" "$HOME/.config/nvim/coc-settings.json"

link "$REPO/claude/CLAUDE.md"        "$HOME/.claude/CLAUDE.md"
link "$REPO/claude/settings.json"    "$HOME/.claude/settings.json"
link "$REPO/claude/keybindings.json" "$HOME/.claude/keybindings.json"
for h in "$REPO"/claude/hooks/*; do
  link "$h" "$HOME/.claude/hooks/$(basename "$h")"
done

link "$REPO/bin/wt-render" "$HOME/.local/bin/wt-render"
link "$REPO/bin/wt-kill"   "$HOME/.local/bin/wt-kill"
link "$REPO/zsh/jatin.zsh-theme" "$OMZ/themes/jatin.zsh-theme"

chmod +x "$HOME"/.claude/hooks/* "$HOME/.local/bin/wt-render" "$HOME/.local/bin/wt-kill" 2>/dev/null

# ---- toolchains -------------------------------------------------------------
if ! command -v cargo >/dev/null 2>&1 && [ ! -f "$HOME/.cargo/env" ]; then
  log "Installing Rust (rustup)"
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
fi

if [ ! -d "$HOME/.nvm" ]; then
  log "Installing nvm + Node LTS"
  clone_if_missing https://github.com/nvm-sh/nvm "$HOME/.nvm"
  export NVM_DIR="$HOME/.nvm"
  . "$NVM_DIR/nvm.sh" && nvm install --lts
fi

if command -v go >/dev/null 2>&1; then
  log "Installing recon (tmux claude orchestrator)"
  go install github.com/gavraz/recon@latest || warn "recon install failed (non-fatal)"
fi

# ---- vim plugins ------------------------------------------------------------
if command -v vim >/dev/null 2>&1; then
  log "Installing vim plugins"
  vim +PluginInstall +qall || true
  [ -d "$HOME/.vim/bundle/coc.nvim" ] && ( cd "$HOME/.vim/bundle/coc.nvim" && yarn install ) || true
fi

# ---- fonts ------------------------------------------------------------------
log "Installing fonts"
mkdir -p "$HOME/Library/Fonts"
find "$REPO/fonts" -type f \( -name '*.ttf' -o -name '*.otf' \) -exec cp {} "$HOME/Library/Fonts/" \;

# ---- default shell ----------------------------------------------------------
if [[ "$SHELL" != *zsh* ]] && command -v zsh >/dev/null 2>&1; then
  log "Setting default shell to zsh"
  chsh -s "$(command -v zsh)" || warn "chsh failed — run it manually"
fi

cat <<'DONE'

Setup complete. Manual finishing touches:
  1. iTerm2: import iterm/jatin.itermcolors and select the "jatin" profile.
  2. tmux: start tmux, then press prefix (C-a) + I to install tmux plugins.
  3. Secrets/per-job env: put them in ~/.zshrc.local (untracked, auto-sourced).
  4. Restart your terminal.
DONE
