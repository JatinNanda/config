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
	all     []*PR
	prs     []*PR
	hidden  map[string]bool
	filter  string
	search  bool
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

const (
	refreshEvery  = 5 * time.Minute
	localEvery    = 10 * time.Second
	flipPollEvery = 2 * time.Minute
)

type tickMsg time.Time
type refreshMsg time.Time
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

func refreshCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return refreshMsg(t) })
}

func (m *model) applyJobStates() {
	states := LoadJobs()
	for _, p := range m.prs {
		st, ok := states[key(p)]
		if !ok {
			continue
		}
		if st.Running && !alive(st.Pid) {
			st.Running = false
			st.Ended = time.Now().Unix()
			SaveJob(st, st.Tag, p.Number)
		}
		p.LogPath = st.Log
		p.Alert, p.AlertMsg = st.Alert()
	}
}

func initialModel() model {
	return model{
		loading: true,
		flips:   map[string]*flip{},
		jobs:    map[string]*job{},
		hidden:  LoadHidden(),
		local:   LoadLocal(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadCmd(), tickCmd(localEvery), refreshCmd(refreshEvery))
}

func haystack(p *PR) string {
	return strings.ToLower(strings.Join([]string{
		p.RepoShort, p.Repo, itoa(p.Number), "#" + itoa(p.Number),
		p.Branch, p.Title, p.WindowName, p.Phase().Label(), p.OriginKind,
	}, " "))
}

func (m *model) refilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter))
	var keep []*PR
	for _, p := range m.all {
		if m.hidden[key(p)] {
			continue
		}
		if q != "" && !strings.Contains(haystack(p), q) {
			continue
		}
		keep = append(keep, p)
	}
	m.prs = keep
	if m.cursor >= len(m.prs) {
		m.cursor = max(0, len(m.prs)-1)
	}
}

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
		states := LoadJobs()
		for _, p := range msg.prs {
			m.local.Attach(p)
			if st, ok := states[key(p)]; ok {
				if st.Running && !alive(st.Pid) {
					st.Running = false
					st.Ended = time.Now().Unix()
					SaveJob(st, st.Tag, p.Number)
				}
				p.LogPath = st.Log
				a, msgTxt := st.Alert()
				p.Alert, p.AlertMsg = a, msgTxt
			}
			if _, ok := m.flips[key(p)]; ok {
				p.Flipping = true
			}
			if _, ok := m.jobs[key(p)]; ok {
				p.Working = true
			}
		}
		m.all = msg.prs
		m.refilter()
		m.lastRef = time.Now()

	case tickMsg:
		for k, j := range m.jobs {
			if !alive(j.pid) {
				delete(m.jobs, k)
				m.status = fmt.Sprintf("%s: agent finished (%s)", k, j.log)
			}
		}
		m.applyJobStates()
		return m, tickCmd(localEvery)

	case refreshMsg:
		cmds := []tea.Cmd{loadCmd(), refreshCmd(refreshEvery)}
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
			return m, tea.Tick(flipPollEvery, func(time.Time) tea.Msg {
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
		if m.search {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.search, m.filter = false, ""
				m.refilter()
			case "enter":
				m.search = false
			case "backspace":
				if r := []rune(m.filter); len(r) > 0 {
					m.filter = string(r[:len(r)-1])
					m.refilter()
				}
			case " ", "space":
				m.filter += " "
				m.refilter()
			default:
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					m.filter += string(msg.Runes)
					m.refilter()
				}
			}
			return m, nil
		}
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
		case "/":
			m.search = true
			m.status = ""
		default:
			if msg.Type == tea.KeyRunes && len(msg.Runes) > 1 && msg.Runes[0] == '/' {
				m.search = true
				m.filter += string(msg.Runes[1:])
				m.refilter()
			}
		case "esc":
			if m.filter != "" {
				m.filter = ""
				m.refilter()
				m.status = "filter cleared"
			}
		case "ctrl+h":
			if p := m.sel(); p != nil {
				m.hidden[key(p)] = true
				if err := SaveHidden(m.hidden); err != nil {
					m.status = "hide failed: " + err.Error()
				} else {
					m.status = fmt.Sprintf("hid %s (prw -unhide %s to undo)", key(p), key(p))
				}
				m.refilter()
			}
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
		case "L":
			if p := m.sel(); p != nil {
				return m, openLogCmd(p)
			}
		case "x":
			if p := m.sel(); p != nil && p.Alert != "" && p.Alert != "running" {
				ClearJob("clean", p.Number)
				ClearJob("bot", p.Number)
				p.Alert, p.AlertMsg = "", ""
				m.status = key(p) + ": alert cleared"
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
	return tea.Tick(flipPollEvery, func(time.Time) tea.Msg { return pollNow(k, since) })
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
	logDir := cacheDir()
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return jobMsg{k: k, err: err}
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%d.log", tag, num))
	resPath := filepath.Join(logDir, fmt.Sprintf("%s-%d.result", tag, num))
	os.Remove(resPath)
	f, err := os.Create(logPath)
	if err != nil {
		return jobMsg{k: k, err: err}
	}
	c := exec.Command("claude", "-p", prompt)
	c.Dir = dir
	c.Env = append(os.Environ(), "PRW_RESULT="+resPath)
	c.Stdout, c.Stderr = f, f
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		f.Close()
		return jobMsg{k: k, err: err}
	}
	st := JobState{
		PR: k, Tag: tag, Pid: c.Process.Pid, Log: logPath, Result: resPath,
		Started: time.Now().Unix(), Running: true,
	}
	SaveJob(st, tag, num)
	go func() {
		c.Wait()
		f.Close()
		st.Running = false
		st.Ended = time.Now().Unix()
		SaveJob(st, tag, num)
	}()
	return jobMsg{k: k, pid: c.Process.Pid, log: logPath}
}

func openLogCmd(p *PR) tea.Cmd {
	if p.LogPath == "" {
		return func() tea.Msg { return statusMsg("no agent log for this PR") }
	}
	path := p.LogPath
	num := p.Number
	return func() tea.Msg {
		if _, err := os.Stat(path); err != nil {
			return statusMsg("log is gone: " + path)
		}
		exec.Command("tmux", "new-window", "-n", fmt.Sprintf("log-%d", num),
			fmt.Sprintf("less +G %q", path)).Run()
		return statusMsg("opened " + path)
	}
}

func (m model) hiddenCount() int {
	n := 0
	for _, p := range m.all {
		if m.hidden[key(p)] {
			n++
		}
	}
	return n
}

func shortDur(d time.Duration) string {
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	if s < 3600 {
		return fmt.Sprintf("%dm%02ds", s/60, s%60)
	}
	return fmt.Sprintf("%dh%02dm", s/3600, (s%3600)/60)
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
		switch {
		case p.Working || p.Alert == "running":
			flag = cAccent.Render("⚙")
		case p.Alert == "failed":
			flag = cBad.Render("✗")
		case p.Alert == "noreport" || p.Alert == "unknown":
			flag = cWarn.Render("!")
		case p.Flipping:
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
	summary := strings.Join(parts, cRule.Render("  │  "))
	if !m.lastRef.IsZero() {
		since := time.Since(m.lastRef)
		next := refreshEvery - since
		if next < 0 {
			next = 0
		}
		stamp := cDim.Render(fmt.Sprintf("updated %s · next %s", shortDur(since), shortDur(next)))
		if pad := w - 2 - lipgloss.Width(summary) - lipgloss.Width(stamp); pad > 1 {
			summary += strings.Repeat(" ", pad) + stamp
		}
	}
	b.WriteString(summary)
	b.WriteString("\n")

	if sel := m.prs; len(sel) > 0 && m.cursor < len(sel) {
		if p := sel[m.cursor]; p.AlertMsg != "" {
			mark, style := "!", cWarn
			if p.Alert == "failed" {
				mark, style = "✗", cBad
			}
			b.WriteString(" " + style.Render(mark) + " " + style.Render(p.AlertMsg) +
				cDim.Render("   L log · x clear") + "\n")
		}
	}
	if m.search || m.filter != "" {
		cur := m.filter
		if m.search {
			cur += "▏"
		}
		hid := ""
		if n := len(m.all) - len(m.prs) - m.hiddenCount(); n >= 0 {
			hid = cDim.Render(fmt.Sprintf("   %d/%d shown", len(m.prs), len(m.all)))
		}
		b.WriteString(" " + cAccent.Render("/") + " " + cStatus.Render(cur) + hid + "\n")
	}
	if m.status != "" {
		b.WriteString(" " + cAccent.Render("▸") + " " + cStatus.Render(m.status) + "\n")
	}

	keys := [][2]string{
		{"enter", "jump"}, {"o", "open"}, {"/", "search"}, {"p", "draft⇄ready"},
		{"f", "bot-flip"}, {"b", "bot-fix"}, {"c", "cleanup"},
		{"L", "log"}, {"x", "clear"}, {"^h", "hide"},
		{"r", "refresh"}, {"q", "quit"},
	}
	var hp []string
	for _, k := range keys {
		hp = append(hp, cKey.Render(k[0])+cDim.Render(" "+k[1]))
	}
	b.WriteString(" " + strings.Join(hp, cRule.Render(" · ")) + "\n")
	return b.String()
}
