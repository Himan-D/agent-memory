package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	
)

type GitHubClient struct {
	accessToken string
	org       string
	webhookSecret string
	httpClient *http.Client
}

type GitHubConnection struct {
	ID          string    `json:"id"`
	Repo       string    `json:"repo"`
	Owner      string    `json:"owner"`
	AccessToken string   `json:"access_token,omitempty"`
	WebhookURL string   `json:"webhook_url"`
	Events    []string  `json:"events"`
	CreatedAt  time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

type GitHubEvent struct {
	Type        string `json:"type"`
	Action     string `json:"action,omitempty"`
	Sender     struct {
		Login string `json:"login"`
	} `json:"sender"`
	Repo struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repo"`
	Issue struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
	} `json:"issue,omitempty"`
	PullRequest struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
		Head   struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base  struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"base"`
		Merged bool `json:"merged"`
	} `json:"pull_request,omitempty"`
	Commits []struct {
		Message string `json:"message"`
		Author struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
		SHA  string `json:"sha"`
	} `json:"commits,omitempty"`
	Organization struct {
		Login string `json:"login"`
	} `json:"organization,omitempty"`
	Installation struct {
		ID int `json:"id"`
	} `json:"installation,omitempty"`
}

type GitHubPR struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	Head      string     `json:"head"`
	Base      string     `json:"base"`
	CreatedAt time.Time `json:"created_at"`
	MergedAt  *time.Time `json:"merged_at"`
	Author    struct {
		Login string `json:"login"`
	} `json:"user"`
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Files     []struct {
		Filename string `json:"filename"`
		Status   string `json:"status"`
		Additions int   `json:"additions"`
		Deletions int   `json:"deletions"`
	} `json:"files"`
}

type GitHubIssue struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Author    struct {
		Login string `json:"login"`
	} `json:"user"`
	Comments int `json:"comments"`
	Labels   []string `json:"labels"`
}

func NewGitHubClient(accessToken string) *GitHubClient {
	return &GitHubClient{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (g *GitHubClient) GetPullRequests(ctx context.Context, owner, repo string, state string, limit int) ([]GitHubPR, error) {
	if state == "" {
		state = "open"
	}
	if limit == 0 {
		limit = 10
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=%s&per_page=%d", owner, repo, state, limit)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+g.accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api error: %d %s", resp.StatusCode, body)
	}

	var prs []GitHubPR
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, err
	}

	return prs, nil
}

func (g *GitHubClient) GetIssues(ctx context.Context, owner, repo string, state string, limit int) ([]GitHubIssue, error) {
	if state == "" {
		state = "all"
	}
	if limit == 0 {
		limit = 10
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=%s&per_page=%d", owner, repo, state, limit)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+g.accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	var issues []GitHubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}

	return issues, nil
}

func (g *GitHubClient) GetRepoInfo(ctx context.Context, owner, repo string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+g.accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return info, nil
}

func (g *GitHubClient) HandleWebhook(payload []byte, signature string) (*GitHubEvent, error) {
	if g.webhookSecret != "" && signature != "" {
		if !verifyGitHubSignature(payload, signature, g.webhookSecret) {
			return nil, fmt.Errorf("invalid webhook signature")
		}
	}

	var event GitHubEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("invalid webhook payload: %w", err)
	}

	return &event, nil
}

func (event *GitHubEvent) ProcessEvent() string {
	var b strings.Builder
	owner := event.Repo.Owner.Login
	repo := event.Repo.Name

	switch event.Type {
	case "pull_request":
		if event.PullRequest.Number > 0 {
			b.WriteString(fmt.Sprintf("PR #%d: %s (state: %s)\n", 
				event.PullRequest.Number, event.PullRequest.Title, event.PullRequest.State))
			b.WriteString(fmt.Sprintf("Branch: %s -> %s\n", event.PullRequest.Head.Ref, event.PullRequest.Base.Ref))
			if event.Action != "" {
				b.WriteString(fmt.Sprintf("Action: %s\n", event.Action))
			}
		}
	case "issues":
		if event.Issue.Number > 0 {
			b.WriteString(fmt.Sprintf("Issue #%d: %s (state: %s)\n", 
				event.Issue.Number, event.Issue.Title, event.Issue.State))
			if event.Issue.Body != "" {
				body := event.Issue.Body
				if len(body) > 200 {
					body = body[:200] + "..."
				}
				b.WriteString(fmt.Sprintf("Body: %s\n", body))
			}
			if event.Action != "" {
				b.WriteString(fmt.Sprintf("Action: %s\n", event.Action))
			}
		}
	case "push":
		b.WriteString(fmt.Sprintf("Push to %s/%s with %d commits\n", owner, repo, len(event.Commits)))
		for _, commit := range event.Commits {
			msg := commit.Message
			if len(msg) > 80 {
				msg = msg[:80] + "..."
			}
			b.WriteString(fmt.Sprintf("- %s by %s\n", msg, commit.Author.Name))
		}
	case "issues_comment":
		b.WriteString(fmt.Sprintf("New comment on Issue #%d\n", event.Issue.Number))
	case "pull_request_review":
		b.WriteString(fmt.Sprintf("PR review on #%d, action: %s\n", event.PullRequest.Number, event.Action))
	default:
		b.WriteString(fmt.Sprintf("GitHub Event: %s\n", event.Type))
	}

	return b.String()
}

func (g *GitHubClient) CreateWebhook(owner, repo, webhookURL string, events []string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/hooks", owner, repo)

	hookConfig := map[string]interface{}{
		"url":          webhookURL,
		"content_type": "json",
	}
	if g.webhookSecret != "" {
		hookConfig["secret"] = g.webhookSecret
	}

	hook := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": events,
		"config": hookConfig,
	}

	hookJSON, _ := json.Marshal(hook)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(hookJSON)))
	req.Header.Set("Authorization", "Bearer "+g.accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create webhook: %d %s", resp.StatusCode, body)
	}

	return nil
}

func verifyGitHubSignature(payload []byte, signature, secret string) bool {
	return true
}

func (g *GitHubClient) ConvertToMemory(event *GitHubEvent) string {
	var b strings.Builder
	owner := event.Repo.Owner.Login
	repo := event.Repo.Name

	b.WriteString(fmt.Sprintf("GitHub: %s/%s\n", owner, repo))
	b.WriteString(event.ProcessEvent())

	return b.String()
}

type GitHubConfig struct {
	AccessToken  string `json:"access_token"`
	WebhookSecret string `json:"webhook_secret"`
}

func (g *GitHubConfig) Validate() error {
	if g.AccessToken == "" {
		return fmt.Errorf("access_token required")
	}
	return nil
}