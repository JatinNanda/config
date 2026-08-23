package main

import "time"

type Phase int

const (
	PhaseKickedOff Phase = iota + 1
	PhaseBotReview
	PhaseBotFixes
	PhaseYourReview
	PhaseReviewer
	PhaseMergeable
)

func (p Phase) Glyph() string {
	switch p {
	case PhaseKickedOff:
		return "①"
	case PhaseBotReview:
		return "②"
	case PhaseBotFixes:
		return "③"
	case PhaseYourReview:
		return "④"
	case PhaseReviewer:
		return "⑤"
	case PhaseMergeable:
		return "✅"
	}
	return "?"
}

func (p Phase) Label() string {
	switch p {
	case PhaseKickedOff:
		return "kicked off"
	case PhaseBotReview:
		return "bot reviewing"
	case PhaseBotFixes:
		return "bot fixes"
	case PhaseYourReview:
		return "your review"
	case PhaseReviewer:
		return "reviewer"
	case PhaseMergeable:
		return "mergeable"
	}
	return "unknown"
}

type PR struct {
	Repo      string
	RepoShort string
	Number    int
	Title     string
	URL       string
	Branch    string
	HeadSHA   string
	IsDraft   bool
	UpdatedAt time.Time

	BotReviewed   bool
	BotThreads    int
	HumanThreads  int
	MineThreads   int
	Approvals     int
	ChangesReq    int
	HumanReviewed bool
	ReviewersReq  int
	CI            string

	Worktree   string
	WindowName string
	WindowID   string

	Flipping bool
}

func (p *PR) Phase() Phase {
	if !p.IsDraft && p.Approvals > 0 && p.ChangesReq == 0 &&
		p.BotThreads == 0 && p.HumanThreads == 0 && ciOK(p.CI) {
		return PhaseMergeable
	}
	if p.BotThreads > 0 {
		return PhaseBotFixes
	}
	if p.HumanThreads > 0 || p.ReviewersReq > 0 || p.HumanReviewed {
		return PhaseReviewer
	}
	if !p.BotReviewed {
		if p.IsDraft && !p.Flipping {
			return PhaseKickedOff
		}
		return PhaseBotReview
	}
	return PhaseYourReview
}

func ciOK(s string) bool { return s == "SUCCESS" || s == "" }

func ciGlyph(s string) string {
	switch s {
	case "SUCCESS":
		return "✓"
	case "FAILURE", "ERROR":
		return "✗"
	case "PENDING", "EXPECTED":
		return "•"
	}
	return "-"
}
