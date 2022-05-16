export ZSH="$HOME/.oh-my-zsh"
export KEYTIMEOUT=20
alias fzf="fzf --color fg:242,bg:233,hl:65,fg+:15,bg+:234,hl+:108 --color info:108,prompt:109,spinner:108,pointer:168,marker:168"
alias vim="nvim"
alias rb="review-branch"
alias ag="ag --ignore-dir playground --ignore-dir virtualenv_run --ignore-dir virtualenv_run_py27 --ignore-dir log --ignore-dir node_modules --ignore-dir coverage --ignore-dir logs --ignore-dir venv  --ignore-dir virtualenv_py3  --ignore-dir docker-venv-py3"

# re-source dots
alias sz="source ~/.zshrc"
alias st="tmux source-file ~/.tmux.conf"
alias z="vim ~/.zshrc"
alias t="vim ~/.tmux.conf"
alias v="vim ~/.vimrc"

alias ipython="ipython --TerminalInteractiveShell.editing_mode=vi"


ZSH_THEME="jatin"

plugins=(
  vi-mode
  git
  zsh-syntax-highlighting
  zsh-autosuggestions
  extract
)

source $ZSH/oh-my-zsh.sh

if [ "$TMUX" ]; then
    PROMPT_COMMAND='eval "$(/nail/scripts/tmux-env)"; '"$PROMPT_COMMAND"
fi

# cd does ls
chpwd() {
  ls
}

# opens file from name in vim regardless of path
 vf() {
     result=$(fzf)
    if [ ! -z "$result" ]; then
        vim $result
    fi
 }

# runs pytests from name regardless of path
 ptf() {
     find . -name "$1" -not -path "./node_modules/*" | xargs pytest
 }

# runs python from name regardless of path
 pyf() {
     find . -name "$1" -not -path "./node_modules/*" | xargs python
 }

# find files from name
 f() {
     find . -name "$1" -not -path "./node_modules/*"
 }

# unix timestamp conversion
dt() {
    if [ -z $1 ]; then
        date '+%s'
    else
        date -d "@$1"
    fi
}

# cd to a directory 2 levels deep from root
cdf() {
    result=$(print -l $HOME/*(/) | fzf)
    if [ ! -z "$result" ]; then
        cd $result
    fi
}

# search and replace
sr() {
    grep -rl "$1" ./ | xargs sed -i "s/$1/$2/g"
}


# clone a repo
 clone() {
     result=$(git list-repos | fzf)
     if [ ! -z "$result" ]; then
        git clone $result
     fi
 }

# checkout branch
co() {
    if [ ! -z "$1" ]; then
        git checkout "$1"
    else
        branch=$(git branch | fzf | rev | cut -d ' ' -f -1 | rev)
        if [ ! -z "$branch" ]; then
            git checkout $branch
        fi
    fi
}

# pull feature branch onto this one
gpullfeat() {
    if [ ! -z "$1" ]; then
        git pull . "$1"
    else
        branch=$(git branch | fzf | rev | cut -d ' ' -f -1 | rev)
        if [ ! -z "$branch" ]; then
            git pull . $branch
        fi
    fi
}

# merge feature to master
gfeatmaster() {
    branch=$(git branch | grep '^*' | sed 's/* //'  )
    if [ "$branch" != "master" ]; then
        git fetch origin master:master
        git rebase origin master
        git checkout master
        git reset --hard origin/master
        git pull . $branch --no-ff
    fi
}

# rebase on master
gbasemaster() {
    branch=$(git branch | grep '^*' | sed 's/* //'  )
    if [ "$branch" != "master" ]; then
        git fetch origin master:master
        git rebase -i origin/master
    fi
}

# reset to master
gresetmaster() {
    git fetch
    git reset --hard origin/master
}

# cd to git project root
cdr() {
    cd "$(git rev-parse --show-toplevel)"
}

# look at files in last n commit's diff (defaults to last diff)
vd() {
    file=$(git diff HEAD~${1:-1} --name-only | fzf)
    if [ ! -z "$file" ]; then
        vim $file
    fi
}

# git diff last n commits (defaults to last commit)
gdiffhead() {
    git diff HEAD~${1:-1}
}

export FZF_DEFAULT_COMMAND='ag --hidden --ignore .git -g ""'

# plugin settings
bindkey '^ ' autosuggest-accept
bindkey '^T' forward-word
bindkey jk vi-cmd-mode

# Fix issues with vi-mode plugin and reverse searching T_T
if [[ "${terminfo[kcuu1]}" != "" ]]; then
  autoload -U up-line-or-beginning-search
  zle -N up-line-or-beginning-search
  bindkey "${terminfo[kcuu1]}" up-line-or-beginning-search
fi
# start typing + [Down-Arrow] - fuzzy find history backward
if [[ "${terminfo[kcud1]}" != "" ]]; then
  autoload -U down-line-or-beginning-search
  zle -N down-line-or-beginning-search
  bindkey "${terminfo[kcud1]}" down-line-or-beginning-search
fi
