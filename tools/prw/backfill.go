package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var prURL = regexp.MustCompile(`https://github\.com/([^/"\s\\]+)/([^/"\s\\]+)/pull/(\d+)`)

func Backfill(known map[string]Origin) ([]Origin, error) {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".claude", "projects")
	var out []Origin
	seen := map[string]bool{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		prs := map[string]bool{}
		cwd, sid := "", ""
		created := false
		armed := 0
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 8<<20)
		for sc.Scan() {
			line := sc.Bytes()
			if bytes.Contains(line, []byte("pr create")) {
				created = true
				armed = 3
			}
			if armed > 0 {
				for _, m := range prURL.FindAllSubmatch(line, -1) {
					prs[fmt.Sprintf("%s/%s#%s", m[1], m[2], m[3])] = true
				}
				armed--
			}
			var d struct {
				Cwd       string `json:"cwd"`
				SessionID string `json:"sessionId"`
			}
			if json.Unmarshal(line, &d) == nil {
				if d.Cwd != "" && (cwd == "" || len(d.Cwd) < len(cwd)) {
					cwd = d.Cwd
				}
				if d.SessionID != "" {
					sid = d.SessionID
				}
			}
		}
		if cwd == "" || !created {
			return nil
		}
		if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
			return nil
		}
		for pr := range prs {
			if seen[pr] {
				continue
			}
			if o, ok := known[pr]; ok && o.Source == "" {
				continue
			}
			seen[pr] = true
			out = append(out, Origin{PR: pr, Cwd: cwd, SessionID: sid, Source: "transcript"})
		}
		return nil
	})
	return out, err
}
