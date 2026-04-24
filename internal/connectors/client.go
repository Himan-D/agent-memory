package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type NotionClient struct {
	clientID     string
	clientSecret string
	accessToken string
	version    string
	httpClient *http.Client
}

type NotionPage struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title,omitempty"`
	Content    string                 `json:"content,omitempty"`
	URL        string                 `json:"url,omitempty"`
	LastEdited time.Time             `json:"last_edited_time,omitempty"`
	ParentID   string                 `json:"parent_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type NotionDatabase struct {
	ID     string                 `json:"id"`
	Title  string                 `json:"title"`
	Config map[string]interface{} `json:"config,omitempty"`
}

type NotionConnection struct {
	ID            string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id"`
	WorkspaceName string   `json:"workspace_name"`
	AccessToken string   `json:"access_token"`
	RedirectURL  string   `json:"redirect_url"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string   `json:"status"`
}

func NewNotionClient(clientID, clientSecret, accessToken string) *NotionClient {
	return &NotionClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		accessToken: accessToken,
		version:     "2022-06-28",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type NotionOAuthStart struct {
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
}

func (n *NotionClient) GetOAuthURL(req NotionOAuthStart) (string, error) {
	authURL := fmt.Sprintf(
		"https://api.notion.com/v1/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&owner=user&scope=blocks:read,blocks:write,content:read,content:write,metadata:read,metadata:write",
		n.clientID,
		req.RedirectURI,
	)
	return authURL, nil
}

type NotionOAuthCallback struct {
	Code         string `json:"code"`
	GrantType    string `json:"grant_type"`
	RedirectURI  string `json:"redirect_uri"`
}

type NotionOAuthResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	BotID        string `json:"bot_id"`
	WorkspaceName string `json:"workspace_name"`
	WorkspaceID  string `json:"workspace_id"`
}

func (n *NotionClient) HandleOAuthCallback(callback NotionOAuthCallback) (*NotionOAuthResponse, error) {
	if callback.Code == "" {
		return nil, fmt.Errorf("authorization code required")
	}

	reqBody := map[string]string{
		"grant_type":    "authorization_code",
		"code":         callback.Code,
		"redirect_uri": callback.RedirectURI,
		"client_id":    n.clientID,
		"client_secret": n.clientSecret,
	}

	reqJSON, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.notion.com/v1/oauth/token", strings.NewReader(string(reqJSON)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oauth failed: %s", body)
	}

	var result NotionOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	n.accessToken = result.AccessToken
	return &result, nil
}

func (n *NotionClient) SyncPages(ctx context.Context, limit int) ([]NotionPage, error) {
	if n.accessToken == "" {
		return nil, fmt.Errorf("notion not authenticated")
	}

	pages, err := n.listPages(ctx, limit)
	if err != nil {
		return nil, err
	}

	for i := range pages {
		content, err := n.getPageContent(pages[i].ID)
		if err == nil {
			pages[i].Content = content
		}
	}

	return pages, nil
}

func (n *NotionClient) listPages(ctx context.Context, limit int) ([]NotionPage, error) {
	if limit == 0 {
		limit = 100
	}

	reqBody := map[string]interface{}{
		"filter": map[string]interface{}{
			"property": "object",
			"value":    "page",
		},
		"page_size": limit,
	}

	reqJSON, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.notion.com/v1/search", strings.NewReader(string(reqJSON)))
	req.Header.Set("Authorization", "Bearer "+n.accessToken)
	req.Header.Set("Notion-Version", n.version)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("notion api error: %d", resp.StatusCode)
	}

	var searchResp struct {
		Results []struct {
			ID       string `json:"id"`
			CreatedTime string `json:"created_time"`
			LastEditedTime string `json:"last_edited_time"`
			Properties map[string]interface{} `json:"properties"`
			URL      string `json:"url"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	var pages []NotionPage
	for _, r := range searchResp.Results {
		title := extractNotionTitle(r.Properties)

		page := NotionPage{
			ID:         r.ID,
			Title:      title,
			URL:        r.URL,
			LastEdited:  parseNotionDate(r.LastEditedTime),
		}

		pages = append(pages, page)
	}

	return pages, nil
}

func (n *NotionClient) getPageContent(pageID string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.notion.com/v1/blocks/"+pageID+"/children", nil)
	req.Header.Set("Authorization", "Bearer "+n.accessToken)
	req.Header.Set("Notion-Version", n.version)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("notion api error: %d", resp.StatusCode)
	}

	var blocksResp struct {
		Results []struct {
			Type string `json:"type"`
			Paragraph struct {
				RichText []struct {
					Text struct {
						Content string `json:"content"`
					} `json:"text"`
				} `json:"rich_text"`
			} `json:"paragraph"`
			Heading1 struct {
				RichText []struct {
					Text struct {
						Content string `json:"content"`
					} `json:"text"`
				} `json:"rich_text"`
			} `json:"heading_1"`
			Heading2 struct {
				RichText []struct {
					Text struct {
						Content string `json:"content"`
					} `json:"text"`
				} `json:"rich_text"`
			} `json:"heading_2"`
			Heading3 struct {
				RichText []struct {
					Text struct {
						Content string `json:"content"`
					} `json:"text"`
				} `json:"rich_text"`
			} `json:"heading_3"`
			BulletedListItem struct {
				RichText []struct {
					Text struct {
						Content string `json:"content"`
					} `json:"text"`
				} `json:"rich_text"`
			} `json:"bulleted_list_item"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&blocksResp); err != nil {
		return "", err
	}

	var content strings.Builder
	for _, block := range blocksResp.Results {
		switch block.Type {
		case "paragraph":
			for _, rt := range block.Paragraph.RichText {
				content.WriteString(rt.Text.Content)
			}
			content.WriteString("\n\n")
		case "heading_1":
			content.WriteString("# ")
			for _, rt := range block.Heading1.RichText {
				content.WriteString(rt.Text.Content)
			}
			content.WriteString("\n\n")
		case "heading_2":
			content.WriteString("## ")
			for _, rt := range block.Heading2.RichText {
				content.WriteString(rt.Text.Content)
			}
			content.WriteString("\n\n")
		case "heading_3":
			content.WriteString("### ")
			for _, rt := range block.Heading3.RichText {
				content.WriteString(rt.Text.Content)
			}
			content.WriteString("\n\n")
		case "bulleted_list_item":
			content.WriteString("- ")
			for _, rt := range block.BulletedListItem.RichText {
				content.WriteString(rt.Text.Content)
			}
			content.WriteString("\n")
		}
	}

	return content.String(), nil
}

type NotionWebhook struct {
	Source     string `json:"source"`
	PageID    string `json:"page_id"`
	Timestamp string `json:"timestamp"`
}

func (n *NotionClient) HandleWebhook(payload NotionWebhook) (*NotionPage, error) {
	if payload.Source != "api" && payload.Source != "user" {
		return nil, fmt.Errorf("invalid webhook source")
	}

	page, err := n.getPage(payload.PageID)
	if err != nil {
		return nil, err
	}

	content, err := n.getPageContent(page.ID)
	if err == nil {
		page.Content = content
	}

	return page, nil
}

func (n *NotionClient) getPage(pageID string) (*NotionPage, error) {
	req, _ := http.NewRequest("GET", "https://api.notion.com/v1/pages/"+pageID, nil)
	req.Header.Set("Authorization", "Bearer "+n.accessToken)
	req.Header.Set("Notion-Version", n.version)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("notion api error: %d", resp.StatusCode)
	}

	var pageResp struct {
		ID       string `json:"id"`
		Properties map[string]interface{} `json:"properties"`
		URL      string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pageResp); err != nil {
		return nil, err
	}

	title := extractNotionTitle(pageResp.Properties)

	return &NotionPage{
		ID:    pageResp.ID,
		Title: title,
		URL:  pageResp.URL,
	}, nil
}

func extractNotionTitle(props map[string]interface{}) string {
	if props == nil {
		return ""
	}

	titleProp, ok := props["title"]
	if !ok {
		return ""
	}

	titleMap, ok := titleProp.(map[string]interface{})
	if !ok {
		return ""
	}

	richText, ok := titleMap["rich_text"]
	if !ok {
		return ""
	}

	rtArray, ok := richText.([]any)
	if !ok || len(rtArray) == 0 {
		return ""
	}

	first, ok := rtArray[0].(map[string]interface{})
	if !ok {
		return ""
	}

	text, ok := first["plain_text"].(string)
	if !ok {
		return ""
	}

	return text
}

func parseNotionDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, dateStr)
	return t
}

type NotionConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI string `json:"redirect_uri"`
}

func (c *NotionConfig) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("client_id required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("client_secret required")
	}
	return nil
}

func (n *NotionClient) CreateConnection(workspaceName, redirectURL string) *NotionConnection {
	return &NotionConnection{
		ID:            uuid.New().String(),
		WorkspaceName: workspaceName,
		RedirectURL:   redirectURL,
		CreatedAt:    time.Now(),
		Status:       "pending",
	}
}

func (n *NotionClient) ConvertToMemory(page *NotionPage) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Notion Page: %s\n", page.Title))
	b.WriteString(fmt.Sprintf("URL: %s\n", page.URL))
	if page.LastEdited.Unix() > 0 {
		b.WriteString(fmt.Sprintf("Last edited: %s\n", page.LastEdited.Format("2006-01-02")))
	}
	b.WriteString("\n")
	if page.Content != "" {
		b.WriteString("Content:\n")
		b.WriteString(page.Content)
	}
	return b.String()
}