package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

func hiddenPath() string { return filepath.Join(cacheDir(), "hidden.json") }

func LoadHidden() map[string]bool {
	res := map[string]bool{}
	b, err := os.ReadFile(hiddenPath())
	if err != nil {
		return res
	}
	var list []string
	if json.Unmarshal(b, &list) != nil {
		return res
	}
	for _, k := range list {
		res[k] = true
	}
	return res
}

func SaveHidden(set map[string]bool) error {
	if err := os.MkdirAll(cacheDir(), 0o755); err != nil {
		return err
	}
	list := make([]string, 0, len(set))
	for k, v := range set {
		if v {
			list = append(list, k)
		}
	}
	sort.Strings(list)
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(hiddenPath(), append(b, '\n'), 0o644)
}
