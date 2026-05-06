package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ProviderType string

const (
	ProviderTypeSAML  ProviderType = "saml"
	ProviderTypeOAuth ProviderType = "oauth"
	ProviderTypeOIDC  ProviderType = "oidc"
	ProviderTypeLDAP  ProviderType = "ldap"
)

type Config struct {
	ProviderType ProviderType
	ClientID     string
	ClientSecret string
	IssuerURL    string
	CallbackURL  string
	TenantID     string
}

type User struct {
	ID        string
	Email     string
	Name      string
	Roles     []string
	TenantID  string
	Groups    []string
	ExpiresAt *string
}

type Session struct {
	ID        string
	UserID    string
	TenantID  string
	Token     string
	ExpiresAt string
}

type Provider interface {
	Name() string
	Type() ProviderType
	Authenticate(ctx context.Context, code string) (*User, error)
	GetLogoutURL(redirectURL string) (string, error)
	ValidateSession(ctx context.Context, token string) (*Session, error)
	RefreshSession(ctx context.Context, token string) (*Session, error)
}

type Manager struct {
	providers map[string]Provider
	configs   map[string]*Config
}

func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]Provider),
		configs:   make(map[string]*Config),
	}
}

func (m *Manager) RegisterProvider(tenantID string, cfg *Config) error {
	if tenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}

	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	var provider Provider
	var err error

	switch cfg.ProviderType {
	case ProviderTypeSAML:
		provider, err = NewSAMLProvider(cfg)
	case ProviderTypeOAuth:
		provider, err = NewOAuthProvider(cfg)
	case ProviderTypeOIDC:
		provider, err = NewOIDCProvider(cfg)
	case ProviderTypeLDAP:
		provider, err = NewLDAPProvider(cfg)
	default:
		return fmt.Errorf("unsupported provider type: %s", cfg.ProviderType)
	}

	if err != nil {
		return err
	}

	m.providers[tenantID] = provider
	m.configs[tenantID] = cfg

	return nil
}

func (m *Manager) GetProvider(tenantID string) (Provider, error) {
	provider, ok := m.providers[tenantID]
	if !ok {
		return nil, fmt.Errorf("no SSO provider registered for tenant: %s", tenantID)
	}
	return provider, nil
}

func (m *Manager) GetConfig(tenantID string) (*Config, error) {
	cfg, ok := m.configs[tenantID]
	if !ok {
		return nil, fmt.Errorf("no SSO config registered for tenant: %s", tenantID)
	}
	return cfg, nil
}

func (m *Manager) ListProviders() []string {
	providers := make([]string, 0, len(m.providers))
	for tenantID := range m.providers {
		providers = append(providers, tenantID)
	}
	return providers
}

func (m *Manager) UnregisterProvider(tenantID string) error {
	if _, ok := m.providers[tenantID]; !ok {
		return fmt.Errorf("no SSO provider registered for tenant: %s", tenantID)
	}
	delete(m.providers, tenantID)
	delete(m.configs, tenantID)
	return nil
}

type OAuthProvider struct {
	config     *Config
	httpClient *http.Client
}

func NewOAuthProvider(cfg *Config) (*OAuthProvider, error) {
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("OAuth client ID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("OAuth client secret is required")
	}
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("OAuth issuer URL is required")
	}

	return &OAuthProvider{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (p *OAuthProvider) Name() string {
	return "OAuth2"
}

func (p *OAuthProvider) Type() ProviderType {
	return ProviderTypeOAuth
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken     string `json:"id_token"`
}

type oauthUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Sub   string `json:"sub"`
}

func (p *OAuthProvider) Authenticate(ctx context.Context, code string) (*User, error) {
	tokenURL := fmt.Sprintf("%s/oauth/token", p.config.IssuerURL)

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"client_id":    {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
		"redirect_uri": {p.config.CallbackURL},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	userInfo, err := p.fetchUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("fetching user info: %w", err)
	}

	userID := userInfo.ID
	if userID == "" {
		userID = userInfo.Sub
	}

	expiresAt := ""
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339)
	}

	return &User{
		ID:        userID,
		Email:     userInfo.Email,
		Name:      userInfo.Name,
		Roles:     []string{"user"},
		TenantID:  p.config.TenantID,
		ExpiresAt: &expiresAt,
	}, nil
}

func (p *OAuthProvider) fetchUserInfo(ctx context.Context, accessToken string) (*oauthUserInfo, error) {
	userInfoURL := fmt.Sprintf("%s/oauth/userinfo", p.config.IssuerURL)

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating user info request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user info request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info returned status %d", resp.StatusCode)
	}

	var info oauthUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decoding user info: %w", err)
	}

	return &info, nil
}

func (p *OAuthProvider) GetLogoutURL(redirectURL string) (string, error) {
	return fmt.Sprintf("%s/oauth/logout?redirect=%s", p.config.IssuerURL, url.QueryEscape(redirectURL)), nil
}

func (p *OAuthProvider) ValidateSession(ctx context.Context, token string) (*Session, error) {
	userInfo, err := p.fetchUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("validating OAuth session: %w", err)
	}

	userID := userInfo.ID
	if userID == "" {
		userID = userInfo.Sub
	}

	return &Session{
		ID:       fmt.Sprintf("oauth-%s", userID),
		UserID:   userID,
		TenantID: p.config.TenantID,
		Token:    token,
		ExpiresAt: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}, nil
}

func (p *OAuthProvider) RefreshSession(ctx context.Context, refreshToken string) (*Session, error) {
	tokenURL := fmt.Sprintf("%s/oauth/token", p.config.IssuerURL)

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh returned %d", resp.StatusCode)
	}

	var tokenResp oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decoding refresh response: %w", err)
	}

	return p.ValidateSession(ctx, tokenResp.AccessToken)
}

type Middleware struct {
	manager *Manager
}

func NewMiddleware(m *Manager) *Middleware {
	return &Middleware{manager: m}
}

func (m *Middleware) Authenticate(ctx context.Context, tenantID string) (*User, error) {
	_, err := m.manager.GetProvider(tenantID)
	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("authentication requires OAuth/SAML code - use provider.Authenticate()")
}

func (m *Middleware) RequireAuth(ctx context.Context, tenantID string) error {
	provider, err := m.manager.GetProvider(tenantID)
	if err != nil {
		return err
	}

	if provider == nil {
		return fmt.Errorf("authentication required but no SSO provider configured for tenant: %s", tenantID)
	}

	return nil
}

func (m *Middleware) RequireRole(ctx context.Context, user *User, role string) error {
	if user == nil {
		return fmt.Errorf("user is required")
	}

	for _, r := range user.Roles {
		if r == role {
			return nil
		}
	}

	return fmt.Errorf("user does not have required role: %s", role)
}
