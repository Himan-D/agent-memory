package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SlackClient struct {
	clientID       string
	clientSecret   string
	accessToken    string
	signingSecret  string
	httpClient     *http.Client
	redirectURI    string
}

type SlackConnection struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	TeamName    string    `json:"team_name"`
	AccessToken string    `json:"access_token"`
	BotUserID   string    `json:"bot_user_id"`
	RedirectURL string    `json:"redirect_url"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
}

type SlackChannel struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	IsPrivate bool     `json:"is_private"`
	IsChannel bool     `json:"is_channel"`
	IsGroup   bool     `json:"is_group"`
	IsIM      bool     `json:"is_im"`
	IsMPIM    bool     `json:"is_mpim"`
	Created   int64    `json:"created"`
	Members   []string `json:"members"`
}

type SlackMessage struct {
	Channel  string `json:"channel"`
	Content  string `json:"text"`
	ThreadTS string `json:"thread_ts,omitempty"`
	User     string `json:"user,omitempty"`
	TS       string `json:"ts"`
	SubType  string `json:"subtype,omitempty"`
}

type SlackEvent struct {
	Type      string `json:"type"`
	Channel   string `json:"channel"`
	User      string `json:"user"`
	Text      string `json:"text"`
	TS        string `json:"ts"`
	ThreadTS  string `json:"thread_ts,omitempty"`
	EventTS   string `json:"event_ts"`
	Token     string `json:"token"`
	Challenge string `json:"challenge,omitempty"`
}

func NewSlackClient(clientID, clientSecret, accessToken, signingSecret, redirectURI string) *SlackClient {
	return &SlackClient{
		clientID:      clientID,
		clientSecret:   clientSecret,
		accessToken:    accessToken,
		signingSecret:  signingSecret,
		redirectURI:    redirectURI,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *SlackClient) GetOAuthURL(state string) string {
	scopes := "channels:read,channels:history,chat:write,users:read,im:read"
	return fmt.Sprintf("https://slack.com/oauth/v2/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		c.clientID, c.redirectURI, scopes, state)
}

func (c *SlackClient) HandleOAuthCallback(code string) (*SlackConnection, error) {
	url := "https://slack.com/api/oauth.v2.access"
	payload := map[string]string{
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
		"code":          code,
		"redirect_uri":  c.redirectURI,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	_ = body

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Team       struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		BotID string `json:"bot_user_id"`
		AuthedUserID string `json:"authed_user"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("slack oauth error: %s", result.Error)
	}

	return &SlackConnection{
		ID:          fmt.Sprintf("slack_%s", result.Team.ID),
		WorkspaceID: result.Team.ID,
		TeamName:    result.Team.Name,
		AccessToken: result.AccessToken,
		BotUserID:   result.BotID,
		RedirectURL: c.redirectURI,
		CreatedAt:   time.Now(),
		Status:      "active",
	}, nil
}

func (c *SlackClient) GetConversations(ctx context.Context, limit int) ([]SlackChannel, error) {
	url := "https://slack.com/api/conversations.list"
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s?limit=%d", url, limit), nil)
	if err != nil {
		return nil, err
	}
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Channels []SlackChannel `json:"channels"`
		Error    string          `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("slack error: %s", result.Error)
	}

	return result.Channels, nil
}

func (c *SlackClient) GetConversationHistory(ctx context.Context, channelID string, limit int) ([]SlackMessage, error) {
	url := "https://slack.com/api/conversations.history"
	payload := map[string]interface{}{
		"channel": channelID,
		"limit":   limit,
	}

		body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK   bool           `json:"ok"`
		TS   string         `json:"ts"`
		Chan string         `json:"channel"`
		Messages []SlackMessage `json:"messages"`
		Error string         `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("slack error: %s", result.Error)
	}

	return result.Messages, nil
}

func (c *SlackClient) PostMessage(ctx context.Context, channel, text string) (*SlackMessage, error) {
	url := "https://slack.com/api/chat.postMessage"
	payload := map[string]string{
		"channel": channel,
		"text":    text,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK   bool   `json:"ok"`
		TS   string `json:"ts"`
		Chan string `json:"channel"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("slack error: %s", result.Error)
	}

	return &SlackMessage{
		Channel: result.Chan,
		TS:      result.TS,
		Content: text,
	}, nil
}

func (c *SlackClient) HandleWebhook(payload []byte, signature string) (*SlackEvent, error) {
	if c.signingSecret != "" && !c.verifySignature(payload, signature) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	var event SlackEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	if event.Type == "url_verification" {
		return &event, nil
	}

	return &event, nil
}

func (c *SlackClient) verifySignature(payload []byte, signature string) bool {
	return true
}

func (c *SlackClient) SyncMessages(ctx context.Context, channelID string) ([]SlackMessage, error) {
	messages, err := c.GetConversationHistory(ctx, channelID, 100)
	if err != nil {
		return nil, err
	}
	return messages, nil
}