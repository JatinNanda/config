package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	list := flag.Bool("list", false, "print the table and exit")
	backfill := flag.Bool("backfill", false, "recover PR origins from Claude session transcripts")
	guess := flag.Bool("guess", false, "infer origins for open PRs that still have none")
	hidden := flag.Bool("hidden", false, "list PRs hidden from the table")
	unhide := flag.String("unhide", "", "unhide a PR (owner/repo#123, or all)")
	apply := flag.Bool("apply", false, "with -guess, write the inferred origins")
	flag.Parse()

	if *hidden {
		h := LoadHidden()
		if len(h) == 0 {
			fmt.Println("nothing hidden")
			return
		}
		keys := make([]string, 0, len(h))
		for k := range h {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Println(k)
		}
		return
	}

	if *unhide != "" {
		h := LoadHidden()
		if *unhide == "all" {
			h = map[string]bool{}
		} else if !h[*unhide] {
			fmt.Fprintf(os.Stderr, "prw: %s is not hidden\n", *unhide)
			os.Exit(1)
		} else {
			delete(h, *unhide)
		}
		if err := SaveHidden(h); err != nil {
			fmt.Fprintln(os.Stderr, "prw:", err)
			os.Exit(1)
		}
		fmt.Println("unhidden:", *unhide)
		return
	}

	if *guess {
		prs, _, err := FetchPRs()
		if err != nil {
			fmt.Fprintln(os.Stderr, "prw:", err)
			os.Exit(1)
		}
		l := LoadLocal()
		for _, p := range prs {
			l.Attach(p)
		}
		found := Guess(prs, l)
		if len(found) == 0 {
			fmt.Println("nothing left to guess")
			return
		}
		byKey := map[string]*PR{}
		for _, p := range prs {
			byKey[keyOf(p)] = p
		}
		for _, o := range found {
			br := ""
			if p := byKey[o.PR]; p != nil {
				br = p.Branch
			}
			fmt.Printf("%-9s %-28s %-38s <- %s\n", o.Source, o.PR, trunc(br, 38), o.Cwd)
		}
		if !*apply {
			fmt.Printf("\n%d guesses, nothing written. re-run with -apply to record them.\n", len(found))
			return
		}
		if err := AppendOrigins(found); err != nil {
			fmt.Fprintln(os.Stderr, "prw:", err)
			os.Exit(1)
		}
		fmt.Printf("\nrecorded %d guessed origins\n", len(found))
		return
	}

	if *backfill {
		known := LoadOrigins()
		found, err := Backfill(known)
		if err != nil {
			fmt.Fprintln(os.Stderr, "prw:", err)
		}
		if len(found) == 0 {
			fmt.Println("no new origins found")
			return
		}
		if err := AppendOrigins(found); err != nil {
			fmt.Fprintln(os.Stderr, "prw:", err)
			os.Exit(1)
		}
		for _, o := range found {
			fmt.Printf("%-28s <- %s\n", o.PR, o.Cwd)
		}
		fmt.Printf("recorded %d origins\n", len(found))
		return
	}

	if *list {
		prs, _, err := FetchPRs()
		if err != nil {
			fmt.Fprintln(os.Stderr, "prw:", err)
			os.Exit(1)
		}
		l := LoadLocal()
		fmt.Printf("%-3s %-10s %-6s %-14s %-11s %-38s %-3s %-4s %-4s %-4s %s\n",
			"PH", "REPO", "PR", "WINDOW", "VIA", "BRANCH", "CI", "BOT", "HUM", "MINE", "PHASE")
		for _, p := range prs {
			l.Attach(p)
			w := p.WindowName
			if w == "" {
				w = "—"
			}
			via := p.OriginKind
			if via == "" {
				via = "-"
			}
			fmt.Printf("%-3s %-10s %-6d %-14s %-11s %-38s %-3s %-4d %-4d %-4d %s\n",
				p.Phase().Glyph(), trunc(p.RepoShort, 10), p.Number, w, via, trunc(p.Branch, 38), ciGlyph(p.CI),
				p.BotThreads, p.HumanThreads, p.MineThreads, p.Phase().Label())
		}
		return
	}

	prog := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "prw:", err)
		os.Exit(1)
	}
}
