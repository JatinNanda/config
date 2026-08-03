# Modification of agnoster theme

CURRENT_BG='NONE'
if [[ -z "$PRIMARY_FG" ]]; then
  PRIMARY_FG=black
fi

SEGMENT_SEPARATOR="\ue0b0"
PLUSMINUS="\u00b1"
BRANCH="\ue0a0"
DETACHED="\u27a6"

prompt_segment() {
  local bg fg
  [[ -n $1 ]] && bg="%K{$1}" || bg="%k"
  [[ -n $2 ]] && fg="%F{$2}" || fg="%f"
  if [[ $CURRENT_BG != 'NONE' && $1 != $CURRENT_BG ]]; then
    print -n "%{$bg%F{$CURRENT_BG}%}$SEGMENT_SEPARATOR%{$fg%}"
  else
    print -n "%{$bg%}%{$fg%}"
  fi
  CURRENT_BG=$1
  [[ -n $3 ]] && print -n $3
}

prompt_end() {
  if [[ -n $CURRENT_BG ]]; then
    print -n "%{%k%F{$CURRENT_BG}%}$SEGMENT_SEPARATOR"
  else
    print -n "%{%k%}"
  fi
  print -n "%{%f%}"
  CURRENT_BG=''
}

# Current git project (unused for now)
prompt_context() {
    ref="$vcs_info_msg_0_"
    if [[ -n "$ref" ]]; then
        git_project=$(git rev-parse --show-toplevel)
        if [[ -n "$git_project" ]]; then
            project=$(basename $git_project)
            prompt_segment 98 260 " ${project} "
        else
            prompt_segment 98 260 " ??? "
        fi
    fi
}

# Reduce git branch names
shorten_git() {
    branch_ref=$1
      if [ $branch_ref != 'master' ]; then
        name=$(echo $branch_ref | cut -d'_' -f-3)
        echo ${name}
      else
        echo $branch_ref
      fi
}


# Git prompt
prompt_git() {
    local color ref status ret
    is_dirty() {
        if [[ -n "$(git rev-parse --show-toplevel)" ]]; then
            ret="$(git status --porcelain --ignore-submodules)"
        else
            ret="aaa"
        fi
        test -n "$ret"
    }
    ref="$vcs_info_msg_0_"
    if [[ -n "$ref" ]]; then
        if is_dirty; then
            color=yellow
        else
            color=green
        fi
        if [[ "${ref/.../}" == "$ref" ]]; then
            shortref=$(shorten_git $ref)
            ref="$BRANCH $shortref"
        else
            shortref=$(shorten_git $ref)
            ref="$DETACHED ${shortref/.../}"
        fi
        prompt_segment $color 235
        print -n " $ref "
    fi
}

# Current dir, in red if last command returned non 0
prompt_dir() {
    if [[ $RETVAL -ne 0 ]]; then
        prompt_segment red black " $(basename $(pwd)) "
    else
        prompt_segment blue black " $(basename $(pwd)) "
    fi

}

# Virtual env
prompt_virtualenv() {
  if [[ -n $VIRTUAL_ENV ]]; then
    color=cyan
    prompt_segment 96 260
    print -Pn " $(basename $VIRTUAL_ENV) "
  fi
}


prompt_rb_main() {
  RETVAL=$?
  CURRENT_BG='NONE'
  prompt_virtualenv
  prompt_dir
  prompt_git
  prompt_end
}


prompt_rb_precmd() {
  vcs_info
  PROMPT='%{%f%b%k%}$(prompt_rb_main) '
}

prompt_rb_setup() {
  autoload -Uz add-zsh-hook
  autoload -Uz vcs_info

  prompt_opts=(cr subst percent)

  add-zsh-hook precmd prompt_rb_precmd

  zstyle ':vcs_info:*' enable git
  zstyle ':vcs_info:*' check-for-changes false
  zstyle ':vcs_info:git*' formats '%b'
  zstyle ':vcs_info:git*' actionformats '%b (%a)'
}

prompt_rb_setup "$@"
