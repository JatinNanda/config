package main

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) string {
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadResult(t *testing.T) {
	d := t.TempDir()
	cases := []struct{ body, want, sum string }{
		{"OK: rebased and stripped 4 comments", "ok", "rebased and stripped 4 comments"},
		{"FAILED: rebase conflict in workflow.ts", "failed", "rebase conflict in workflow.ts"},
		{"FAIL: could not fix CI", "failed", "could not fix CI"},
		{"something else entirely", "unknown", "something else entirely"},
		{"", "", ""},
	}
	for i, c := range cases {
		p := write(t, d, "r"+string(rune('a'+i)), c.body)
		st, sum := ReadResult(p)
		if st != c.want || sum != c.sum {
			t.Errorf("%q -> (%q,%q), want (%q,%q)", c.body, st, sum, c.want, c.sum)
		}
	}
	if st, _ := ReadResult(filepath.Join(d, "missing")); st != "" {
		t.Errorf("missing file -> %q, want empty", st)
	}
}

func TestAlert(t *testing.T) {
	d := t.TempDir()
	if a, _ := (JobState{Running: true}).Alert(); a != "running" {
		t.Errorf("running -> %q", a)
	}
	if a, _ := (JobState{Result: filepath.Join(d, "nope")}).Alert(); a != "noreport" {
		t.Errorf("no result file -> %q, want noreport", a)
	}
	p := write(t, d, "ok", "OK: done")
	if a, _ := (JobState{Result: p}).Alert(); a != "ok" {
		t.Errorf("ok -> %q", a)
	}
	p = write(t, d, "bad", "FAILED: nope")
	if a, s := (JobState{Result: p}).Alert(); a != "failed" || s != "nope" {
		t.Errorf("failed -> (%q,%q)", a, s)
	}
}
