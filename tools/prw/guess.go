package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func translateHome(p string) string {
	if p == "" {
		return ""
	}
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(p, home) {
		return p
	}
	for _, pre := range []string{"/home/", "/Users/", "/mnt/c/Users/"} {
		if !strings.HasPrefix(p, pre) {
			continue
		}
		rest := p[len(pre):]
		i := strings.IndexByte(rest, '/')
		if i < 0 {
			return home
		}
		return filepath.Join(home, rest[i+1:])
	}
	return p
}

func usableDir(p string) (string, bool) {
	t := translateHome(p)
	if t == "" {
		return "", false
	}
	if st, err := os.Stat(t); err == nil && st.IsDir() {
		return t, true
	}
	return "", false
}

type evidence struct {
	byBranch  map[string]map[string]int
	byMention map[string]map[string]int
}

func scanTranscripts(branches, prKeys map[string]bool) *evidence {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".claude", "projects")
	ev := &evidence{
		byBranch:  map[string]map[string]int{},
		byMention: map[string]map[string]int{},
	}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 8<<20)

		rootCwd := ""
		brHits := map[string]int{}
		mentions := map[string]bool{}
		for sc.Scan() {
			line := sc.Bytes()
			var d struct {
				Cwd       string `json:"cwd"`
				GitBranch string `json:"gitBranch"`
			}
			if json.Unmarshal(line, &d) == nil {
				if d.Cwd != "" && (rootCwd == "" || len(d.Cwd) < len(rootCwd)) {
					rootCwd = d.Cwd
				}
				if d.GitBranch != "" && branches[d.GitBranch] {
					brHits[d.GitBranch]++
				}
			}
			for _, m := range prURL.FindAllSubmatch(line, -1) {
				k := string(m[1]) + "/" + string(m[2]) + "#" + string(m[3])
				if prKeys[k] {
					mentions[k] = true
				}
			}
		}
		dir, ok := usableDir(rootCwd)
		if !ok {
			return nil
		}
		for b, n := range brHits {
			if ev.byBranch[b] == nil {
				ev.byBranch[b] = map[string]int{}
			}
			ev.byBranch[b][dir] += n
		}
		for k := range mentions {
			if ev.byMention[k] == nil {
				ev.byMention[k] = map[string]int{}
			}
			ev.byMention[k][dir]++
		}
		return nil
	})
	return ev
}

func bestOf(m map[string]int) (string, int) {
	best, n := "", 0
	for k, v := range m {
		if v > n || (v == n && k < best) {
			best, n = k, v
		}
	}
	return best, n
}

func Guess(prs []*PR, l *Local) []Origin {
	branches := map[string]bool{}
	prKeys := map[string]bool{}
	var todo []*PR
	for _, p := range prs {
		if p.OriginKind != "" && p.OriginKind != "branch" {
			continue
		}
		todo = append(todo, p)
		branches[p.Branch] = true
		prKeys[keyOf(p)] = true
	}
	if len(todo) == 0 {
		return nil
	}
	ev := scanTranscripts(branches, prKeys)

	var out []Origin
	for _, p := range todo {
		key := keyOf(p)
		if dir, n := bestOf(ev.byBranch[p.Branch]); dir != "" && n > 0 {
			out = append(out, Origin{PR: key, Cwd: dir, Source: "gitbranch"})
			continue
		}
		if dir, n := bestOf(ev.byMention[key]); dir != "" && n > 0 {
			out = append(out, Origin{PR: key, Cwd: dir, Source: "mention"})
			continue
		}
		if root := l.RepoRoot(p.Repo); root != "" {
			out = append(out, Origin{PR: key, Cwd: root, Source: "repo"})
		}
	}
	return out
}
