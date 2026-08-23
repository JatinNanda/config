package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const prQuery = `
query($q:String!){
  viewer{ login }
  search(query:$q, type:ISSUE, first:60){
    nodes{
      ... on PullRequest{
        number title url isDraft updatedAt headRefName
        repository{ nameWithOwner }
        commits(last:1){ nodes{ commit{ oid statusCheckRollup{ state } } } }
        reviews(first:60){ nodes{ author{ login __typename } state } }
        reviewRequests(first:20){ nodes{ requestedReviewer{ __typename } } }
        reviewThreads(first:100){
          nodes{
            isResolved
            comments(first:10){ nodes{ author{ login __typename } } }
          }
        }
      }
    }
  }
}`

type ghAuthor struct {
	Login    string `json:"login"`
	Typename string `json:"__typename"`
}

type ghResp struct {
	Data struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
		Search struct {
			Nodes []struct {
				Number     int    `json:"number"`
				Title      string `json:"title"`
				URL        string `json:"url"`
				IsDraft    bool   `json:"isDraft"`
				UpdatedAt  string `json:"updatedAt"`
				HeadRef    string `json:"headRefName"`
				Repository struct {
					NameWithOwner string `json:"nameWithOwner"`
				} `json:"repository"`
				Commits struct {
					Nodes []struct {
						Commit struct {
							Oid               string `json:"oid"`
							StatusCheckRollup *struct {
								State string `json:"state"`
							} `json:"statusCheckRollup"`
						} `json:"commit"`
					} `json:"nodes"`
				} `json:"commits"`
				Reviews struct {
					Nodes []struct {
						Author ghAuthor `json:"author"`
						State  string   `json:"state"`
					} `json:"nodes"`
				} `json:"reviews"`
				ReviewRequests struct {
					Nodes []struct {
						RequestedReviewer struct {
							Typename string `json:"__typename"`
						} `json:"requestedReviewer"`
					} `json:"nodes"`
				} `json:"reviewRequests"`
				ReviewThreads struct {
					Nodes []struct {
						IsResolved bool `json:"isResolved"`
						Comments   struct {
							Nodes []struct {
								Author ghAuthor `json:"author"`
							} `json:"nodes"`
						} `json:"comments"`
					} `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"nodes"`
		} `json:"search"`
	} `json:"data"`
}

func isBot(a ghAuthor) bool {
	if a.Typename == "Bot" {
		return true
	}
	l := strings.ToLower(a.Login)
	if strings.HasSuffix(l, "[bot]") {
		return true
	}
	for _, p := range []string{"coderabbitai", "greptile", "sonarcloud", "codecov", "copilot", "sentry", "dependabot", "linear-code", "spacelift-"} {
		if strings.HasPrefix(l, p) {
			return true
		}
	}
	return false
}

func FetchPRs() ([]*PR, string, error) {
	cmd := exec.Command("gh", "api", "graphql",
		"-F", "q=is:pr is:open author:@me archived:false",
		"-f", "query="+prQuery)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, "", fmt.Errorf("gh: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, "", err
	}
	var r ghResp
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, "", err
	}
	me := r.Data.Viewer.Login

	var prs []*PR
	for _, n := range r.Data.Search.Nodes {
		if n.Number == 0 {
			continue
		}
		short := n.Repository.NameWithOwner
		if i := strings.IndexByte(short, '/'); i >= 0 {
			short = short[i+1:]
		}
		p := &PR{
			Repo:      n.Repository.NameWithOwner,
			RepoShort: short,
			Number:    n.Number,
			Title:     n.Title,
			URL:       n.URL,
			Branch:    n.HeadRef,
			IsDraft:   n.IsDraft,
		}
		p.UpdatedAt, _ = time.Parse(time.RFC3339, n.UpdatedAt)
		if len(n.Commits.Nodes) > 0 {
			c := n.Commits.Nodes[0].Commit
			p.HeadSHA = c.Oid
			if c.StatusCheckRollup != nil {
				p.CI = c.StatusCheckRollup.State
			}
		}
		for _, rv := range n.Reviews.Nodes {
			switch {
			case isBot(rv.Author):
				p.BotReviewed = true
			case rv.Author.Login == me:
			default:
				p.HumanReviewed = true
			}
			if rv.Author.Login != me && !isBot(rv.Author) {
				switch rv.State {
				case "APPROVED":
					p.Approvals++
				case "CHANGES_REQUESTED":
					p.ChangesReq++
				}
			}
		}
		p.ReviewersReq = len(n.ReviewRequests.Nodes)
		for _, t := range n.ReviewThreads.Nodes {
			if t.IsResolved || len(t.Comments.Nodes) == 0 {
				continue
			}
			first := t.Comments.Nodes[0].Author
			if isBot(first) {
				p.BotThreads++
				continue
			}
			others := 0
			for _, c := range t.Comments.Nodes {
				if !isBot(c.Author) && c.Author.Login != me {
					others++
				}
			}
			if others > 0 {
				p.HumanThreads++
			} else {
				p.MineThreads++
			}
		}
		prs = append(prs, p)
	}
	sort.Slice(prs, func(i, j int) bool {
		if prs[i].Phase() != prs[j].Phase() {
			return prs[i].Phase() < prs[j].Phase()
		}
		return prs[i].UpdatedAt.After(prs[j].UpdatedAt)
	})
	return prs, me, nil
}

func BotReviewedSince(repo string, number int, since time.Time) (bool, error) {
	q := fmt.Sprintf(`query{ repository(owner:"%s", name:"%s"){ pullRequest(number:%d){
	  reviews(first:60){ nodes{ author{ login __typename } submittedAt } } } } }`,
		strings.Split(repo, "/")[0], strings.Split(repo, "/")[1], number)
	out, err := exec.Command("gh", "api", "graphql", "-f", "query="+q).Output()
	if err != nil {
		return false, err
	}
	var r struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					Reviews struct {
						Nodes []struct {
							Author      ghAuthor `json:"author"`
							SubmittedAt string   `json:"submittedAt"`
						} `json:"nodes"`
					} `json:"reviews"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return false, err
	}
	for _, rv := range r.Data.Repository.PullRequest.Reviews.Nodes {
		if !isBot(rv.Author) {
			continue
		}
		t, err := time.Parse(time.RFC3339, rv.SubmittedAt)
		if err == nil && t.After(since) {
			return true, nil
		}
	}
	return false, nil
}

func ChangedFiles(repo string, number int) ([]string, error) {
	out, err := exec.Command("gh", "pr", "view", fmt.Sprint(number), "--repo", repo,
		"--json", "files", "--jq", ".files[].path").Output()
	if err != nil {
		return nil, err
	}
	var fs []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l != "" {
			fs = append(fs, l)
		}
	}
	return fs, nil
}

func MarkReady(repo string, number int) error {
	return exec.Command("gh", "pr", "ready", fmt.Sprint(number), "--repo", repo).Run()
}

func MarkDraft(repo string, number int) error {
	return exec.Command("gh", "pr", "ready", fmt.Sprint(number), "--repo", repo, "--undo").Run()
}

func ClearReviewers(repo string, number int) {
	out, err := exec.Command("gh", "pr", "view", fmt.Sprint(number), "--repo", repo,
		"--json", "reviewRequests", "--jq", "[.reviewRequests[].login] | join(\",\")").Output()
	if err != nil {
		return
	}
	logins := strings.TrimSpace(string(out))
	if logins == "" {
		return
	}
	exec.Command("gh", "pr", "edit", fmt.Sprint(number), "--repo", repo,
		"--remove-reviewer", logins).Run()
}
