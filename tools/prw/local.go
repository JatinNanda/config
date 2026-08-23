package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Local struct {
	Worktrees map[string]string
	RepoRoots map[string]string
	Windows   map[string][2]string
}

func codeDir() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "code")
}

func discoverRepos() map[string]string {
	roots := map[string]string{}
	entries, err := os.ReadDir(codeDir())
	if err != nil {
		return roots
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(codeDir(), e.Name())
		if _, err := os.Stat(filepath.Join(p, ".git")); err == nil {
			roots[e.Name()] = p
		}
	}
	return roots
}

func worktreeBranches(repoRoot string) map[string]string {
	out, err := exec.Command("git", "-C", repoRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil
	}
	res := map[string]string{}
	var path string
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			br := strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
			if path != "" && br != "" {
				res[br] = path
			}
		}
	}
	return res
}

func tmuxWindows() map[string][2]string {
	out, err := exec.Command("tmux", "list-windows", "-a", "-F",
		"#{window_id}\t#{window_name}\t#{pane_current_path}").Output()
	if err != nil {
		return nil
	}
	res := map[string][2]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			continue
		}
		res[f[2]] = [2]string{f[0], f[1]}
	}
	return res
}

func LoadLocal() *Local {
	l := &Local{
		Worktrees: map[string]string{},
		RepoRoots: discoverRepos(),
		Windows:   tmuxWindows(),
	}
	for name, root := range l.RepoRoots {
		for br, path := range worktreeBranches(root) {
			l.Worktrees[name+"\x00"+br] = path
		}
	}
	return l
}

func (l *Local) Attach(p *PR) {
	name := p.Repo
	if i := strings.IndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	wt, ok := l.Worktrees[name+"\x00"+p.Branch]
	if !ok {
		return
	}
	p.Worktree = wt
	best := ""
	for path, w := range l.Windows {
		if path == wt || strings.HasPrefix(path, wt+string(os.PathSeparator)) {
			if best == "" || len(path) < len(best) {
				best = path
				p.WindowID = w[0]
				p.WindowName = w[1]
			}
		}
	}
}

func (l *Local) RepoRoot(repo string) string {
	name := repo
	if i := strings.IndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	return l.RepoRoots[name]
}

func codeownerPatterns(repoRoot string) []string {
	var pats []string
	for _, rel := range []string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS"} {
		b, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			f := strings.Fields(line)
			if len(f) >= 2 {
				pats = append(pats, f[0])
			}
		}
	}
	return pats
}

func CodeownedHits(repoRoot string, files []string) []string {
	pats := codeownerPatterns(repoRoot)
	var hits []string
	for _, f := range files {
		for _, pat := range pats {
			p := strings.TrimPrefix(pat, "/")
			if p == "" {
				continue
			}
			if strings.HasSuffix(p, "/") {
				if strings.HasPrefix(f, p) {
					hits = append(hits, f)
					break
				}
				continue
			}
			if f == p || strings.HasPrefix(f, p+"/") {
				hits = append(hits, f)
				break
			}
			if ok, _ := filepath.Match(p, f); ok {
				hits = append(hits, f)
				break
			}
			if ok, _ := filepath.Match(p, filepath.Base(f)); ok && !strings.Contains(p, "/") {
				hits = append(hits, f)
				break
			}
		}
	}
	return hits
}
