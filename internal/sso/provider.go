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

type SAMLProvider struct {
	config *Config
}

func NewSAMLProvider(cfg *Config) (*SAMLProvider, error) {
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("SAML issuer URL is required")
	}
	if cfg.CallbackURL == "" {
		return nil, fmt.Errorf("SAML callback URL is required")
	}

	return &SAMLProvider{config: cfg}, nil
}

func (p *SAMLProvider) Name() string {
	return "SAML"
}

func (p *SAMLProvider) Type() ProviderType {
	return ProviderTypeSAML
}

func (p *SAMLProvider) Authenticate(ctx context.Context, code string) (*User, error) {
	return nil, fmt.Errorf("SAML authentication not implemented - requires IdP metadata")
}

func (p *SAMLProvider) GetLogoutURL(redirectURL string) (string, error) {
	return fmt.Sprintf("%s/saml/logout?redirect=%s", p.config.IssuerURL, redirectURL), nil
}

func (p *SAMLProvider) ValidateSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("SAML session validation not implemented")
}

func (p *SAMLProvider) RefreshSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("SAML session refresh not implemented")
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

type OIDCProvider struct {
	config *Config
}

func NewOIDCProvider(cfg *Config) (*OIDCProvider, error) {
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("OIDC client ID is required")
	}
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("OIDC issuer URL is required")
	}

	return &OIDCProvider{config: cfg}, nil
}

func (p *OIDCProvider) Name() string {
	return "OIDC"
}

func (p *OIDCProvider) Type() ProviderType {
	return ProviderTypeOIDC
}

func (p *OIDCProvider) Authenticate(ctx context.Context, code string) (*User, error) {
	return nil, fmt.Errorf("OIDC authentication not implemented - requires token exchange with %s", p.config.IssuerURL)
}

func (p *OIDCProvider) GetLogoutURL(redirectURL string) (string, error) {
	return fmt.Sprintf("%s/oauth/logout?redirect=%s", p.config.IssuerURL, redirectURL), nil
}

func (p *OIDCProvider) ValidateSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("OIDC session validation not implemented")
}

func (p *OIDCProvider) RefreshSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("OIDC session refresh not implemented")
}

type LDAPProvider struct {
	config *Config
}

func NewLDAPProvider(cfg *Config) (*LDAPProvider, error) {
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("LDAP server URL is required")
	}

	return &LDAPProvider{config: cfg}, nil
}

func (p *LDAPProvider) Name() string {
	return "LDAP"
}

func (p *LDAPProvider) Type() ProviderType {
	return ProviderTypeLDAP
}

func (p *LDAPProvider) Authenticate(ctx context.Context, code string) (*User, error) {
	return nil, fmt.Errorf("LDAP authentication not implemented - requires LDAP server connection to %s", p.config.IssuerURL)
}

func (p *LDAPProvider) GetLogoutURL(redirectURL string) (string, error) {
	return redirectURL, nil
}

func (p *LDAPProvider) ValidateSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("LDAP session validation not implemented")
}

func (p *LDAPProvider) RefreshSession(ctx context.Context, token string) (*Session, error) {
	return nil, fmt.Errorf("LDAP session refresh not implemented")
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
