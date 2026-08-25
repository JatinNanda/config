package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type JobState struct {
	PR      string `json:"pr"`
	Tag     string `json:"tag"`
	Pid     int    `json:"pid"`
	Log     string `json:"log"`
	Result  string `json:"result"`
	Started int64  `json:"started"`
	Ended   int64  `json:"ended"`
	Running bool   `json:"running"`
}

func cacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "prw")
}

func jobsDir() string { return filepath.Join(cacheDir(), "jobs") }

func jobFile(tag string, num int) string {
	return filepath.Join(jobsDir(), tag+"-"+itoa(num)+".json")
}

func SaveJob(j JobState, tag string, num int) {
	if err := os.MkdirAll(jobsDir(), 0o755); err != nil {
		return
	}
	b, err := json.Marshal(j)
	if err != nil {
		return
	}
	os.WriteFile(jobFile(tag, num), b, 0o644)
}

func LoadJobs() map[string]JobState {
	res := map[string]JobState{}
	entries, err := os.ReadDir(jobsDir())
	if err != nil {
		return res
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(jobsDir(), e.Name()))
		if err != nil {
			continue
		}
		var j JobState
		if json.Unmarshal(b, &j) != nil || j.PR == "" {
			continue
		}
		if prev, ok := res[j.PR]; ok && prev.Started > j.Started {
			continue
		}
		res[j.PR] = j
	}
	return res
}

func ClearJob(tag string, num int) {
	os.Remove(jobFile(tag, num))
}

func ReadResult(path string) (status, summary string) {
	if path == "" {
		return "", ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	txt := strings.TrimSpace(string(b))
	if txt == "" {
		return "", ""
	}
	line := txt
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	up := strings.ToUpper(line)
	switch {
	case strings.HasPrefix(up, "OK"):
		return "ok", strings.TrimSpace(strings.TrimPrefix(line[2:], ":"))
	case strings.HasPrefix(up, "FAILED"), strings.HasPrefix(up, "FAIL"):
		i := strings.IndexByte(line, ':')
		if i >= 0 {
			return "failed", strings.TrimSpace(line[i+1:])
		}
		return "failed", ""
	}
	return "unknown", line
}

func (j JobState) Alert() (string, string) {
	if j.Running {
		return "running", ""
	}
	st, sum := ReadResult(j.Result)
	switch st {
	case "ok":
		return "ok", sum
	case "failed":
		return "failed", sum
	case "":
		return "noreport", ""
	}
	return "unknown", sum
}

func ClearAlert(num int) {
	for _, tag := range []string{"clean", "bot"} {
		ClearJob(tag, num)
		base := filepath.Join(cacheDir(), tag+"-"+itoa(num))
		os.Remove(base + ".result")
	}
}
