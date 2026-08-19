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

# ---- Homebrew (macOS) / apt (Linux, incl. WSL) + packages -------------------
if [ "$(uname)" = "Darwin" ]; then
  if ! command -v brew >/dev/null 2>&1; then
    log "Installing Homebrew"
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  fi
  eval "$(/opt/homebrew/bin/brew shellenv 2>/dev/null || /usr/local/bin/brew shellenv)"
  log "Installing Brewfile packages"
  brew bundle --file "$REPO/Brewfile"
elif [ "$(uname)" = "Linux" ] && command -v apt-get >/dev/null 2>&1; then
  log "Installing packages via apt"
  sudo apt-get update
  # node/yarn/pnpm come from nvm+corepack below, not apt, to match Brewfile's intent
  for pkg in git gh zsh tmux neovim fzf silversearcher-ag jq golang-go; do
    sudo apt-get install -y "$pkg" || warn "apt install $pkg failed (non-fatal)"
  done
else
  warn "Unsupported OS — install git/zsh/tmux/neovim/fzf/ag/jq/go/node yourself."
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
mkdir -p "$HOME/.claude/skills"
for s in "$REPO"/claude/skills/*; do
  link "$s" "$HOME/.claude/skills/$(basename "$s")"
done

link "$REPO/bin/wt-render" "$HOME/.local/bin/wt-render"
link "$REPO/bin/wt-kill"   "$HOME/.local/bin/wt-kill"
link "$REPO/bin/winterm-colors" "$HOME/.local/bin/winterm-colors"
link "$REPO/bin/wsl-browser" "$HOME/.local/bin/wsl-browser"
link "$REPO/bin/onboard-teammate" "$HOME/.local/bin/onboard-teammate"
link "$REPO/bin/tmux-resurrect-guard" "$HOME/.local/bin/tmux-resurrect-guard"
link "$REPO/bin/claude-context-sync" "$HOME/.local/bin/claude-context-sync"
link "$REPO/zsh/jatin.zsh-theme" "$OMZ/themes/jatin.zsh-theme"

chmod +x "$HOME"/.claude/hooks/* "$REPO"/claude/skills/*/scripts/* \
         "$HOME/.local/bin/wt-render" "$HOME/.local/bin/wt-kill" \
         "$HOME/.local/bin/winterm-colors" "$HOME/.local/bin/wsl-browser" \
         "$HOME/.local/bin/onboard-teammate" \
         "$HOME/.local/bin/tmux-resurrect-guard" \
         "$HOME/.local/bin/claude-context-sync" 2>/dev/null

# ---- toolchains -------------------------------------------------------------
if ! command -v cargo >/dev/null 2>&1 && [ ! -f "$HOME/.cargo/env" ]; then
  log "Installing Rust (rustup)"
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
fi

log "Installing nvm + Node LTS"
clone_if_missing https://github.com/nvm-sh/nvm "$HOME/.nvm"
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
command -v node >/dev/null 2>&1 || nvm install --lts
command -v yarn >/dev/null 2>&1 || corepack enable 2>/dev/null || npm install -g yarn
command -v pnpm >/dev/null 2>&1 || corepack enable 2>/dev/null || npm install -g pnpm

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
if [ "$(uname)" = "Darwin" ]; then
  FONT_DIR="$HOME/Library/Fonts"
else
  FONT_DIR="$HOME/.local/share/fonts"
fi
mkdir -p "$FONT_DIR"
find "$REPO/fonts" -type f \( -name '*.ttf' -o -name '*.otf' \) -exec cp {} "$FONT_DIR/" \;
command -v fc-cache >/dev/null 2>&1 && fc-cache -f "$FONT_DIR" >/dev/null

# ---- terminal colors --------------------------------------------------------
# On macOS the palette is imported into iTerm2 by hand. On WSL nothing was
# applying it, so Windows Terminal kept its dark default (Campbell) and every
# color the theme picks came out muddy.
if grep -qi microsoft /proc/version 2>/dev/null || [ -n "${WSL_DISTRO_NAME:-}" ]; then
  if command -v python3 >/dev/null 2>&1; then
    log "Applying color palette to Windows Terminal"
    "$REPO/bin/winterm-colors" || warn "winterm-colors failed (non-fatal)"
  else
    warn "python3 missing, skipping Windows Terminal palette"
  fi
fi

# ---- default shell ----------------------------------------------------------
if [[ "$SHELL" != *zsh* ]] && command -v zsh >/dev/null 2>&1; then
  log "Setting default shell to zsh"
  chsh -s "$(command -v zsh)" || warn "chsh failed — run it manually"
fi

if [ "$(uname)" = "Darwin" ]; then
  TERM_STEP="iTerm2: import iterm/jatin.itermcolors and select the \"jatin\" profile."
else
  TERM_STEP="Colors were applied to Windows Terminal automatically. Fonts were installed on the Linux side only: on WSL, also copy fonts/Inconsolata/*.ttf into a Windows font dir (or double-click + Install) if you want them in Windows Terminal, then pick Inconsolata there."
fi

cat <<DONE

Setup complete. Manual finishing touches:
  1. $TERM_STEP
  2. tmux: start tmux, then press prefix (C-a) + I to install tmux plugins.
  3. Secrets/per-job env: put them in ~/.zshrc.local (untracked, auto-sourced).
  4. Restart your terminal.
DONE
