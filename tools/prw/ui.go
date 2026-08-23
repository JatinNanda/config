package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	cDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cHead   = lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true)
	cSel    = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	cOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	cWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	cBad    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	cErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	cStatus = lipgloss.NewStyle().Foreground(lipgloss.Color("151"))
	cRule   = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	cKey    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	cAccent = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
)

var phaseColor = map[Phase]lipgloss.Color{
	PhaseKickedOff:  "245",
	PhaseBotReview:  "111",
	PhaseBotFixes:   "214",
	PhaseYourReview: "141",
	PhaseReviewer:   "204",
	PhaseMergeable:  "42",
}

func phStyle(p Phase) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(phaseColor[p])
}

type flip struct {
	started time.Time
	stage   string
}

type job struct {
	pid int
	log string
}

func alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

type model struct {
	prs     []*PR
	local   *Local
	me      string
	cursor  int
	w, h    int
	loading bool
	err     string
	status  string
	confirm *PR
	flips   map[string]*flip
	jobs    map[string]*job
	lastRef time.Time
}

func key(p *PR) string { return fmt.Sprintf("%s#%d", p.Repo, p.Number) }

type prsMsg struct {
	prs []*PR
	me  string
	err error
}
type tickMsg time.Time
type flipMsg struct {
	k     string
	stage string
	err   error
}
type statusMsg string
type toggledMsg struct {
	k        string
	nowDraft bool
}
type jobMsg struct {
	k   string
	pid int
	log string
	err error
}

func loadCmd() tea.Cmd {
	return func() tea.Msg {
		prs, me, err := FetchPRs()
		return prsMsg{prs, me, err}
	}
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func initialModel() model {
	return model{loading: true, flips: map[string]*flip{}, jobs: map[string]*job{}, local: LoadLocal()}
}

func (m model) Init() tea.Cmd { return tea.Batch(loadCmd(), tickCmd(60*time.Second)) }

func (m *model) sel() *PR {
	if m.cursor < 0 || m.cursor >= len(m.prs) {
		return nil
	}
	return m.prs[m.cursor]
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height

	case prsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.me = msg.me
		m.local = LoadLocal()
		for _, p := range msg.prs {
			m.local.Attach(p)
			if _, ok := m.flips[key(p)]; ok {
				p.Flipping = true
			}
			if _, ok := m.jobs[key(p)]; ok {
				p.Working = true
			}
		}
		m.prs = msg.prs
		m.lastRef = time.Now()
		if m.cursor >= len(m.prs) {
			m.cursor = max(0, len(m.prs)-1)
		}

	case tickMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, loadCmd(), tickCmd(60*time.Second))
		for k, j := range m.jobs {
			if !alive(j.pid) {
				delete(m.jobs, k)
				m.status = fmt.Sprintf("%s: bot-comments finished (%s)", k, j.log)
			}
		}
		for k, f := range m.flips {
			if f.stage == "waiting" {
				cmds = append(cmds, pollFlip(k, f.started))
			}
		}
		return m, tea.Batch(cmds...)

	case flipMsg:
		f := m.flips[msg.k]
		if msg.err != nil {
			m.status = "flip failed: " + msg.err.Error()
			delete(m.flips, msg.k)
			return m, loadCmd()
		}
		switch msg.stage {
		case "waiting":
			if f != nil {
				f.stage = "waiting"
			}
			m.status = msg.k + ": ready, waiting for arc-accrual"
			return m, pollFlip(msg.k, f.started)
		case "done":
			delete(m.flips, msg.k)
			m.status = msg.k + ": bot reviewed, back to draft"
			return m, loadCmd()
		case "pending":
			return m, tea.Tick(45*time.Second, func(time.Time) tea.Msg {
				return pollNow(msg.k, f.started)
			})
		}

	case statusMsg:
		m.status = string(msg)

	case jobMsg:
		if msg.err != nil {
			m.status = "bot-comments failed: " + msg.err.Error()
			return m, nil
		}
		m.jobs[msg.k] = &job{pid: msg.pid, log: msg.log}
		m.status = fmt.Sprintf("%s: bot-comments running detached (pid %d)", msg.k, msg.pid)
		return m, nil

	case toggledMsg:
		if msg.nowDraft {
			m.status = msg.k + ": converted to draft"
		} else {
			m.status = msg.k + ": marked ready for review"
		}
		return m, loadCmd()

	case tea.KeyMsg:
		if m.confirm != nil {
			switch msg.String() {
			case "y", "Y":
				p := m.confirm
				m.confirm = nil
				return m, m.startFlip(p)
			default:
				m.confirm = nil
				m.status = "flip cancelled"
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.prs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g":
			m.cursor = 0
		case "G":
			m.cursor = max(0, len(m.prs)-1)
		case "r":
			m.loading = true
			m.status = "refreshing"
			return m, loadCmd()
		case "o":
			if p := m.sel(); p != nil {
				exec.Command("open", p.URL).Start()
				m.status = "opened " + key(p)
			}
		case "enter":
			if p := m.sel(); p != nil {
				return m, jumpCmd(p)
			}
		case "b":
			if p := m.sel(); p != nil {
				if _, ok := m.jobs[key(p)]; ok {
					m.status = key(p) + ": a job is already running"
				} else {
					p.Working = true
					l, pr := m.local, p
					m.status = key(p) + ": starting bot-comments"
					return m, func() tea.Msg {
						return spawnAgent(l, pr, "bot",
							fmt.Sprintf("/pr-bot-comments %s#%d", pr.Repo, pr.Number))
					}
				}
			}
		case "c":
			if p := m.sel(); p != nil {
				if _, ok := m.jobs[key(p)]; ok {
					m.status = key(p) + ": a job is already running"
				} else {
					p.Working = true
					l, pr := m.local, p
					m.status = key(p) + ": starting cleanup"
					return m, func() tea.Msg {
						return spawnAgent(l, pr, "clean", cleanupPrompt(pr))
					}
				}
			}
		case "p":
			if p := m.sel(); p != nil {
				return m, m.toggleDraft(p)
			}
		case "f":
			if p := m.sel(); p != nil {
				return m, m.requestFlip(p)
			}
		}
	}
	return m, nil
}

func (m *model) toggleDraft(p *PR) tea.Cmd {
	k := key(p)
	if _, ok := m.flips[k]; ok {
		return func() tea.Msg { return statusMsg(k + ": flip in progress, let it finish") }
	}
	repo, num, wasDraft := p.Repo, p.Number, p.IsDraft
	if wasDraft {
		m.status = k + ": marking ready…"
	} else {
		m.status = k + ": converting to draft…"
	}
	p.IsDraft = !wasDraft
	return func() tea.Msg {
		var err error
		if wasDraft {
			err = MarkReady(repo, num)
		} else {
			err = MarkDraft(repo, num)
		}
		if err != nil {
			return statusMsg(fmt.Sprintf("%s: %v", k, err))
		}
		return toggledMsg{k, !wasDraft}
	}
}

func (m *model) requestFlip(p *PR) tea.Cmd {
	if !p.IsDraft {
		return func() tea.Msg { return statusMsg("already ready for review") }
	}
	if _, ok := m.flips[key(p)]; ok {
		return func() tea.Msg { return statusMsg("flip already in progress") }
	}
	root := m.local.RepoRoot(p.Repo)
	if root != "" {
		if files, err := ChangedFiles(p.Repo, p.Number); err == nil {
			if hits := CodeownedHits(root, files); len(hits) > 0 {
				m.confirm = p
				n := len(hits)
				sample := hits[0]
				if n > 1 {
					sample = fmt.Sprintf("%s +%d more", hits[0], n-1)
				}
				m.status = fmt.Sprintf("touches CODEOWNERS paths (%s) — owners get notified. flip anyway? [y/N]", sample)
				return nil
			}
		}
	}
	return m.startFlip(p)
}

func (m *model) startFlip(p *PR) tea.Cmd {
	k := key(p)
	m.flips[k] = &flip{started: time.Now().Add(-2 * time.Second), stage: "readying"}
	p.Flipping = true
	m.status = k + ": marking ready"
	repo, num := p.Repo, p.Number
	return func() tea.Msg {
		if err := MarkReady(repo, num); err != nil {
			return flipMsg{k, "", err}
		}
		ClearReviewers(repo, num)
		return flipMsg{k, "waiting", nil}
	}
}

func pollNow(k string, since time.Time) tea.Msg {
	repo, num := parseKey(k)
	ok, err := BotReviewedSince(repo, num, since)
	if err != nil {
		return flipMsg{k, "pending", nil}
	}
	if !ok {
		return flipMsg{k, "pending", nil}
	}
	if err := MarkDraft(repo, num); err != nil {
		return flipMsg{k, "", err}
	}
	return flipMsg{k, "done", nil}
}

func pollFlip(k string, since time.Time) tea.Cmd {
	return tea.Tick(45*time.Second, func(time.Time) tea.Msg { return pollNow(k, since) })
}

func parseKey(k string) (string, int) {
	i := strings.LastIndexByte(k, '#')
	var n int
	fmt.Sscanf(k[i+1:], "%d", &n)
	return k[:i], n
}

func jumpCmd(p *PR) tea.Cmd {
	if p.WindowID == "" {
		return func() tea.Msg { return statusMsg("no tmux window for " + p.Branch) }
	}
	return func() tea.Msg {
		exec.Command("tmux", "select-window", "-t", p.WindowID).Run()
		return statusMsg("jumped to " + p.WindowName)
	}
}

func spawnAgent(l *Local, p *PR, tag, prompt string) tea.Msg {
	k, num := key(p), p.Number
	dir, err := l.EnsureWorktree(p.Repo, p.Branch)
	if err != nil {
		return jobMsg{k: k, err: err}
	}
	return startDetached(k, num, tag, dir, prompt)
}

func startDetached(k string, num int, tag, dir, prompt string) tea.Msg {
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".cache", "prw")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return jobMsg{k: k, err: err}
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%d.log", tag, num))
	f, err := os.Create(logPath)
	if err != nil {
		return jobMsg{k: k, err: err}
	}
	c := exec.Command("claude", "-p", prompt)
	c.Dir = dir
	c.Stdout, c.Stderr = f, f
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		f.Close()
		return jobMsg{k: k, err: err}
	}
	go func() {
		c.Wait()
		f.Close()
	}()
	return jobMsg{k: k, pid: c.Process.Pid, log: logPath}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func trunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s + strings.Repeat(" ", n-len(r))
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

func (m model) View() string {
	if m.loading && len(m.prs) == 0 {
		return "\n  loading PRs…\n"
	}
	if m.err != "" {
		return "\n  " + cErr.Render("error: "+m.err) + "\n\n  press r to retry, q to quit\n"
	}

	w := m.w
	if w < 80 {
		w = 120
	}
	repoW, winW, brW := 9, 14, 32
	fixed := 4 + repoW + 7 + winW + brW + 4 + 5 + 5 + 9
	titleW := w - fixed
	if titleW < 16 {
		titleW = 16
	}

	var b strings.Builder
	b.WriteString("\n ")
	b.WriteString(cHead.Render(fmt.Sprintf("%-3s %-*s %-6s %-*s %-*s %-3s %-4s %-4s %-*s",
		"PH", repoW, "REPO", "PR", winW, "WINDOW", brW, "BRANCH", "CI", "BOT", "HUM", titleW, "TITLE")))
	b.WriteString("\n")

	for i, p := range m.prs {
		ph := p.Phase()
		win := p.WindowName
		if win == "" {
			win = cDim.Render("—")
			win += strings.Repeat(" ", max(0, winW-1))
		} else {
			win = trunc(win, winW)
		}
		bot, hum := "–", "–"
		if p.BotThreads > 0 {
			bot = cWarn.Render(fmt.Sprint(p.BotThreads))
		} else if p.BotReviewed {
			bot = cDim.Render("0")
		}
		if p.HumanThreads > 0 {
			hum = cBad.Render(fmt.Sprint(p.HumanThreads))
		} else if p.HumanReviewed {
			hum = cDim.Render("0")
		}
		ci := ciGlyph(p.CI)
		switch p.CI {
		case "SUCCESS":
			ci = cOK.Render(ci)
		case "FAILURE", "ERROR":
			ci = cBad.Render(ci)
		default:
			ci = cDim.Render(ci)
		}
		flag := " "
		if p.Working {
			flag = cAccent.Render("⚙")
		} else if p.Flipping {
			flag = cWarn.Render("⟳")
		}
		line := fmt.Sprintf(" %s%s %s %-6d %s %s %s   %s    %s    %s",
			phStyle(ph).Render(ph.Glyph()), flag, cDim.Render(trunc(p.RepoShort, repoW)),
			p.Number, win, trunc(p.Branch, brW), ci, bot, hum, trunc(p.Title, titleW))
		if i == m.cursor {
			pad := max(0, w-1-lipgloss.Width(line))
			b.WriteString(cSel.Render(line + strings.Repeat(" ", pad)))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString(" " + cRule.Render(strings.Repeat("─", max(0, w-2))) + "\n ")
	counts := map[Phase]int{}
	for _, p := range m.prs {
		counts[p.Phase()]++
	}
	var parts []string
	for ph := PhaseKickedOff; ph <= PhaseMergeable; ph++ {
		if counts[ph] > 0 {
			parts = append(parts, phStyle(ph).Bold(true).Render(fmt.Sprintf("%s  %d", ph.Glyph(), counts[ph]))+
				cDim.Render(" "+ph.Label()))
		}
	}
	b.WriteString(strings.Join(parts, cRule.Render("  │  ")))
	b.WriteString("\n")

	if m.status != "" {
		b.WriteString(" " + cAccent.Render("▸") + " " + cStatus.Render(m.status) + "\n")
	}

	keys := [][2]string{
		{"enter", "jump"}, {"o", "open"}, {"p", "draft⇄ready"},
		{"f", "flip → bot → draft"}, {"b", "fix bot comments"},
		{"c", "strip comments + fix CI"}, {"r", "refresh"}, {"q", "quit"},
	}
	var hp []string
	for _, k := range keys {
		hp = append(hp, cKey.Render(k[0])+cDim.Render(" "+k[1]))
	}
	help := " " + strings.Join(hp, cRule.Render(" · "))
	if !m.lastRef.IsZero() {
		age := cDim.Render(fmt.Sprintf("updated %ds ago", int(time.Since(m.lastRef).Seconds())))
		if pad := w - 1 - lipgloss.Width(help) - lipgloss.Width(age); pad > 1 {
			help += strings.Repeat(" ", pad) + age
		}
	}
	b.WriteString(help + "\n")
	return b.String()
}
