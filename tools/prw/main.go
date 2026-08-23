package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	list := flag.Bool("list", false, "print the table and exit")
	flag.Parse()

	if *list {
		prs, _, err := FetchPRs()
		if err != nil {
			fmt.Fprintln(os.Stderr, "prw:", err)
			os.Exit(1)
		}
		l := LoadLocal()
		fmt.Printf("%-3s %-10s %-6s %-14s %-38s %-3s %-4s %-4s %-4s %s\n",
			"PH", "REPO", "PR", "WINDOW", "BRANCH", "CI", "BOT", "HUM", "MINE", "PHASE")
		for _, p := range prs {
			l.Attach(p)
			w := p.WindowName
			if w == "" {
				w = "—"
			}
			fmt.Printf("%-3s %-10s %-6d %-14s %-38s %-3s %-4d %-4d %-4d %s\n",
				p.Phase().Glyph(), trunc(p.RepoShort, 10), p.Number, w, trunc(p.Branch, 38), ciGlyph(p.CI),
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
