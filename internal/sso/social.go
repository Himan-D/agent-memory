package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SocialProvider struct {
	providerType string
	config       *Config
	httpClient   *http.Client
}

func NewGoogleProvider(cfg *Config) (*SocialProvider, error) {
	return &SocialProvider{
		providerType: "google",
		config:       cfg,
		httpClient:   &http.Client{Timeout: 10 * httpSecond},
	}, nil
}

func NewGitHubProvider(cfg *Config) (*SocialProvider, error) {
	return &SocialProvider{
		providerType: "github",
		config:       cfg,
		httpClient:   &http.Client{Timeout: 10 * httpSecond},
	}, nil
}

const httpSecond = 1e9

func (p *SocialProvider) Name() string {
	if p.providerType == "google" {
		return "Google"
	}
	return "GitHub"
}

func (p *SocialProvider) Type() ProviderType {
	return ProviderTypeOAuth
}

func (p *SocialProvider) GetAuthorizationURL(state string) string {
	switch p.providerType {
	case "google":
		return fmt.Sprintf(
			"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+email+profile&state=%s",
			url.QueryEscape(p.config.ClientID),
			url.QueryEscape(p.config.CallbackURL),
			url.QueryEscape(state),
		)
	case "github":
		return fmt.Sprintf(
			"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email&state=%s",
			url.QueryEscape(p.config.ClientID),
			url.QueryEscape(p.config.CallbackURL),
			url.QueryEscape(state),
		)
	}
	return ""
}

func (p *SocialProvider) Authenticate(ctx context.Context, code string) (*User, error) {
	token, err := p.exchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("social oauth token exchange: %w", err)
	}
	return p.fetchUserInfo(ctx, token)
}

func (p *SocialProvider) exchangeCode(ctx context.Context, code string) (string, error) {
	var tokenURL string
	var data url.Values

	switch p.providerType {
	case "google":
		tokenURL = "https://oauth2.googleapis.com/token"
		data = url.Values{
			"code":          {code},
			"client_id":     {p.config.ClientID},
			"client_secret": {p.config.ClientSecret},
			"redirect_uri":  {p.config.CallbackURL},
			"grant_type":    {"authorization_code"},
		}
	case "github":
		tokenURL = "https://github.com/login/oauth/access_token"
		data = url.Values{
			"code":          {code},
			"client_id":     {p.config.ClientID},
			"client_secret": {p.config.ClientSecret},
			"redirect_uri":  {p.config.CallbackURL},
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if p.providerType == "github" {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	accessToken, _ := result["access_token"].(string)
	if accessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}
	return accessToken, nil
}

type socialUserInfo struct {
	ID    string
	Email string
	Name  string
}

func (p *SocialProvider) fetchUserInfo(ctx context.Context, accessToken string) (*User, error) {
	var infoURL string
	headers := map[string]string{}

	switch p.providerType {
	case "google":
		infoURL = "https://openidconnect.googleapis.com/v1/userinfo"
		headers["Authorization"] = "Bearer " + accessToken
	case "github":
		infoURL = "https://api.github.com/user"
		headers["Authorization"] = "Bearer " + accessToken
		headers["Accept"] = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	info := &socialUserInfo{}
	if p.providerType == "google" {
		info.ID, _ = raw["sub"].(string)
		info.Email, _ = raw["email"].(string)
		info.Name, _ = raw["name"].(string)
	} else {
		info.ID = fmt.Sprintf("%.0f", raw["id"].(float64))
		info.Email, _ = raw["email"].(string)
		info.Name, _ = raw["name"].(string)
		if info.Email == "" {
			info.Email = fmt.Sprintf("%s@github", info.ID)
		}
	}

	return &User{
		ID:       info.ID,
		Email:    info.Email,
		Name:     info.Name,
		TenantID: p.config.TenantID,
	}, nil
}

func (p *SocialProvider) GetLogoutURL(redirectURL string) (string, error) {
	if p.providerType == "google" {
		return fmt.Sprintf("https://accounts.google.com/Logout?continue=%s", url.QueryEscape(redirectURL)), nil
	}
	return redirectURL, nil
}

func (p *SocialProvider) ValidateSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("social oauth: session validation not supported")
}

func (p *SocialProvider) RefreshSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("social oauth: refresh not supported")
}
