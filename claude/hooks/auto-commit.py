#!/usr/bin/env python3
"""
PostToolUse hook: auto-stage, commit (or amend), push, and print a PR link.
- Edit/Write: stages the edited file and commits
- Bash: if the command contains 'git commit' or 'git push', outputs the PR link
"""
import json
import os
import re
import subprocess
import sys


def run(cmd, cwd=None):
    return subprocess.run(cmd, capture_output=True, text=True, cwd=cwd)


def get_repo_root():
    r = run(['git', 'rev-parse', '--show-toplevel'])
    return r.stdout.strip() if r.returncode == 0 else None


def get_current_branch(repo_root):
    return run(['git', 'branch', '--show-current'], cwd=repo_root).stdout.strip()


def get_pr_url(repo_root, branch):
    url = run(['git', 'remote', 'get-url', 'origin'], cwd=repo_root).stdout.strip()
    m = re.search(r'github\.com[:/](.+?)(?:\.git)?$', url)
    if not m:
        return None
    return f'https://github.com/{m.group(1)}/compare/{branch}?expand=1'


def print_pr_link(repo_root, branch):
    pr_url = get_pr_url(repo_root, branch)
    if pr_url:
        print(f'Let the user know the changes were committed and pushed. Include this PR link in your response: {pr_url}')


def handle_bash(data):
    # Only used to catch manual git push/commit in case Claude runs them directly.
    # Ideally this never fires — Edit/Write hook should handle everything.
    command = data.get('tool_input', {}).get('command', '')
    if not re.search(r'git (commit|push)', command):
        return

    repo_root = get_repo_root()
    if not repo_root:
        return

    branch = get_current_branch(repo_root)
    if not branch:
        return

    print_pr_link(repo_root, branch)


def handle_edit_write(data):
    file_path = data.get('tool_input', {}).get('file_path', '')
    if not file_path:
        return

    repo_root = get_repo_root()
    if not repo_root:
        return

    try:
        if os.path.relpath(file_path, repo_root).startswith('..'):
            return
    except ValueError:
        return

    current_branch = get_current_branch(repo_root)
    if not current_branch:
        return  # detached HEAD

    def commits_ahead_of_base():
        for base in ('dev', 'master', 'main', 'origin/dev', 'origin/master', 'origin/main'):
            r = run(['git', 'rev-list', '--count', f'{base}..HEAD'], cwd=repo_root)
            if r.returncode == 0 and r.stdout.strip().isdigit():
                return int(r.stdout.strip())
        return 0

    run(['git', 'add', file_path], cwd=repo_root)
    if run(['git', 'diff', '--cached', '--quiet'], cwd=repo_root).returncode == 0:
        return  # nothing staged

    if not current_branch.startswith('jatin/'):
        return  # don't interfere with non-jatin branches

    slug = current_branch.removeprefix('jatin/')
    commit_msg = 'update ' + re.sub(r'[_\-]+', '-', slug.lower())

    # Only amend if the latest commit was created by this hook (matching commit message).
    # Never amend pre-existing commits from other authors/tools.
    last_msg = run(['git', 'log', '-1', '--format=%s'], cwd=repo_root).stdout.strip()
    if last_msg == commit_msg:
        subprocess.run(['git', 'commit', '--amend', '--no-edit'], cwd=repo_root,
                       capture_output=True)
    else:
        subprocess.run(['git', 'commit', '-m', commit_msg], cwd=repo_root,
                       capture_output=True)

    subprocess.run(
        ['git', 'push', 'origin', f'HEAD:{current_branch}', '--force'],
        cwd=repo_root, capture_output=True
    )

    print_pr_link(repo_root, current_branch)


def main():
    try:
        data = json.load(sys.stdin)
    except Exception:
        return

    tool = data.get('tool_name')
    if tool == 'Bash':
        handle_bash(data)
    elif tool in ('Edit', 'Write'):
        handle_edit_write(data)


main()
