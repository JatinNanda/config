package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Origin struct {
	PR          string `json:"pr"`
	Cwd         string `json:"cwd"`
	WindowID    string `json:"window_id"`
	WindowName  string `json:"window_name"`
	TmuxSession string `json:"tmux_session"`
	SessionID   string `json:"session_id"`
	Source      string `json:"source"`
	TS          int64  `json:"ts"`
}

type Local struct {
	Worktrees map[string]string
	RepoRoots map[string]string
	Windows   map[string][2]string
	LiveIDs   map[string]string
	ByName    map[string]string
	Origins   map[string]Origin
}

func originsPath() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".cache", "prw", "origins.jsonl")
}

func LoadOrigins() map[string]Origin {
	res := map[string]Origin{}
	f, err := os.Open(originsPath())
	if err != nil {
		return res
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var o Origin
		if json.Unmarshal(sc.Bytes(), &o) != nil || o.PR == "" {
			continue
		}
		if prev, ok := res[o.PR]; ok && prev.Source == "" && o.Source != "" {
			continue
		}
		res[o.PR] = o
	}
	return res
}

func AppendOrigins(list []Origin) error {
	p := originsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, o := range list {
		if err := enc.Encode(o); err != nil {
			return err
		}
	}
	return nil
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

func tmuxIndex() (ids map[string]string, byName map[string]string) {
	ids, byName = map[string]string{}, map[string]string{}
	out, err := exec.Command("tmux", "list-windows", "-a", "-F",
		"#{window_id}\t#{window_name}").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 2 {
			continue
		}
		ids[f[0]] = f[1]
		if _, seen := byName[f[1]]; !seen {
			byName[f[1]] = f[0]
		}
	}
	return
}

func LoadLocal() *Local {
	ids, byName := tmuxIndex()
	l := &Local{
		Worktrees: map[string]string{},
		RepoRoots: discoverRepos(),
		Windows:   tmuxWindows(),
		LiveIDs:   ids,
		ByName:    byName,
		Origins:   LoadOrigins(),
	}
	for name, root := range l.RepoRoots {
		for br, path := range worktreeBranches(root) {
			l.Worktrees[name+"\x00"+br] = path
		}
	}
	return l
}

func (l *Local) Attach(p *PR) {
	if o, ok := l.Origins[keyOf(p)]; ok && l.resolveOrigin(p, o) {
		return
	}
	l.attachByBranch(p)
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func keyOf(p *PR) string {
	return p.Repo + "#" + itoa(p.Number)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func (l *Local) resolveOrigin(p *PR, o Origin) bool {
	if o.WindowID != "" {
		if name, ok := l.LiveIDs[o.WindowID]; ok {
			p.WindowID, p.WindowName = o.WindowID, name
			p.Worktree, p.OriginKind = o.Cwd, o.Source
			if p.OriginKind == "" {
				p.OriginKind = "recorded"
			}
			return true
		}
	}
	if o.WindowName != "" {
		if id, ok := l.ByName[o.WindowName]; ok {
			p.WindowID, p.WindowName = id, o.WindowName
			p.Worktree, p.OriginKind = o.Cwd, or(o.Source, "renamed")
			return true
		}
	}
	if o.Cwd != "" {
		if w, ok := l.Windows[o.Cwd]; ok {
			p.WindowID, p.WindowName = w[0], w[1]
			p.Worktree, p.OriginKind = o.Cwd, or(o.Source, "cwd")
			return true
		}
	}
	return false
}

func (l *Local) attachByBranch(p *PR) {
	name := p.Repo
	if i := strings.IndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	wt, ok := l.Worktrees[name+"\x00"+p.Branch]
	if !ok {
		return
	}
	p.Worktree = wt
	p.OriginKind = "branch"
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

func (l *Local) BranchWorktree(repo, branch string) string {
	name := repo
	if i := strings.IndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	return l.Worktrees[name+"\x00"+branch]
}

func (l *Local) EnsureWorktree(repo, branch string) (string, error) {
	if wt := l.BranchWorktree(repo, branch); wt != "" {
		return wt, nil
	}
	root := l.RepoRoot(repo)
	if root == "" {
		return "", fmt.Errorf("no local clone of %s", repo)
	}
	name := repo
	if i := strings.IndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	feature := branch
	if i := strings.LastIndexByte(feature, '/'); i >= 0 && strings.HasPrefix(feature, "jatin/") {
		feature = feature[i+1:]
	}
	feature = strings.ReplaceAll(feature, "/", "-")
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".worktrees", name, feature)

	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists but holds a different branch", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	exec.Command("git", "-C", root, "fetch", "origin", branch).Run()
	out, err := exec.Command("git", "-C", root, "worktree", "add", path, branch).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("worktree add: %s", strings.TrimSpace(string(out)))
	}
	return path, nil
}
