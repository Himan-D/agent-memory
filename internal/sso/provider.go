package sso

import (
	"context"
	"fmt"
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
	config *Config
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

	return &OAuthProvider{config: cfg}, nil
}

func (p *OAuthProvider) Name() string {
	return "OAuth"
}

func (p *OAuthProvider) Type() ProviderType {
	return ProviderTypeOAuth
}

func (p *OAuthProvider) Authenticate(ctx context.Context, code string) (*User, error) {
	return nil, fmt.Errorf("OAuth authentication not implemented - requires token exchange with %s", p.config.IssuerURL)
}

func (p *OAuthProvider) GetLogoutURL(redirectURL string) (string, error) {
	return fmt.Sprintf("%s/oauth/logout?redirect=%s", p.config.IssuerURL, redirectURL), nil
}

func (p *OAuthProvider) ValidateSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("OAuth session validation not implemented")
}

func (p *OAuthProvider) RefreshSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("OAuth session refresh not implemented")
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
