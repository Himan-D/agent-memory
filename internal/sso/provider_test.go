package sso

import (
	"context"
	"testing"
)

func TestNewOAuthProvider_MissingClientID(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
	}
	_, err := NewOAuthProvider(cfg)
	if err == nil {
		t.Error("expected error for missing client ID")
	}
	if err.Error() != "OAuth client ID is required" {
		t.Errorf("expected 'OAuth client ID is required' error, got %v", err)
	}
}

func TestNewOAuthProvider_MissingClientSecret(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		IssuerURL:    "https://example.com",
	}
	_, err := NewOAuthProvider(cfg)
	if err == nil {
		t.Error("expected error for missing client secret")
	}
	if err.Error() != "OAuth client secret is required" {
		t.Errorf("expected 'OAuth client secret is required' error, got %v", err)
	}
}

func TestNewOAuthProvider_MissingIssuerURL(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
	}
	_, err := NewOAuthProvider(cfg)
	if err == nil {
		t.Error("expected error for missing issuer URL")
	}
	if err.Error() != "OAuth issuer URL is required" {
		t.Errorf("expected 'OAuth issuer URL is required' error, got %v", err)
	}
}

func TestNewOAuthProvider_Success(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
	}
	provider, err := NewOAuthProvider(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider to be non-nil")
	}
}

func TestOAuthProvider_Name(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
	}
	provider, _ := NewOAuthProvider(cfg)
	if provider.Name() != "OAuth2" {
		t.Errorf("expected Name() = 'OAuth2', got %q", provider.Name())
	}
}

func TestOAuthProvider_Type(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
	}
	provider, _ := NewOAuthProvider(cfg)
	if provider.Type() != ProviderTypeOAuth {
		t.Errorf("expected Type() = %q, got %q", ProviderTypeOAuth, provider.Type())
	}
}

func TestOAuthProvider_GetLogoutURL(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
		IssuerURL:    "https://auth.example.com",
	}
	provider, _ := NewOAuthProvider(cfg)

	url, err := provider.GetLogoutURL("https://app.example.com/dashboard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://auth.example.com/oauth/logout?redirect=https%3A%2F%2Fapp.example.com%2Fdashboard" {
		t.Errorf("unexpected logout URL: %s", url)
	}
}

func TestOAuthProvider_GetLogoutURL_EmptyRedirect(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
		IssuerURL:    "https://auth.example.com",
	}
	provider, _ := NewOAuthProvider(cfg)

	url, err := provider.GetLogoutURL("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://auth.example.com/oauth/logout?redirect=" {
		t.Errorf("unexpected logout URL: %s", url)
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.providers == nil {
		t.Error("providers map is nil")
	}
	if m.configs == nil {
		t.Error("configs map is nil")
	}
}

func TestManager_RegisterProvider_OAuth(t *testing.T) {
	m := NewManager()
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
	}

	err := m.RegisterProvider("tenant-1", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	provider, err := m.GetProvider("tenant-1")
	if err != nil {
		t.Fatalf("unexpected error getting provider: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider to be non-nil")
	}
	if provider.Name() != "OAuth2" {
		t.Errorf("expected provider name 'OAuth2', got %q", provider.Name())
	}
}

func TestManager_RegisterProvider_EmptyTenantID(t *testing.T) {
	m := NewManager()
	cfg := &Config{ProviderType: ProviderTypeOAuth}

	err := m.RegisterProvider("", cfg)
	if err == nil {
		t.Error("expected error for empty tenant ID")
	}
	if err.Error() != "tenant ID is required" {
		t.Errorf("expected 'tenant ID is required' error, got %v", err)
	}
}

func TestManager_RegisterProvider_NilConfig(t *testing.T) {
	m := NewManager()

	err := m.RegisterProvider("tenant-1", nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
	if err.Error() != "config is required" {
		t.Errorf("expected 'config is required' error, got %v", err)
	}
}

func TestManager_RegisterProvider_UnsupportedType(t *testing.T) {
	m := NewManager()
	cfg := &Config{
		ProviderType: "unknown",
		ClientID:     "id",
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
	}

	err := m.RegisterProvider("tenant-1", cfg)
	if err == nil {
		t.Error("expected error for unsupported provider type")
	}
}

func TestManager_GetProvider_NotFound(t *testing.T) {
	m := NewManager()

	_, err := m.GetProvider("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent provider")
	}
	if err.Error() != "no SSO provider registered for tenant: nonexistent" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestManager_GetConfig(t *testing.T) {
	m := NewManager()
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
	}

	m.RegisterProvider("tenant-1", cfg)

	retrievedCfg, err := m.GetConfig("tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrievedCfg.ClientID != "client-id" {
		t.Errorf("expected ClientID 'client-id', got %q", retrievedCfg.ClientID)
	}
}

func TestManager_GetConfig_NotFound(t *testing.T) {
	m := NewManager()
	_, err := m.GetConfig("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent config")
	}
}

func TestManager_ListProviders(t *testing.T) {
	m := NewManager()

	if len(m.ListProviders()) != 0 {
		t.Error("expected empty list initially")
	}

	cfg1 := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "id1",
		ClientSecret: "secret1",
		IssuerURL:    "https://example1.com",
	}
	cfg2 := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "id2",
		ClientSecret: "secret2",
		IssuerURL:    "https://example2.com",
	}

	m.RegisterProvider("tenant-1", cfg1)
	m.RegisterProvider("tenant-2", cfg2)

	providers := m.ListProviders()
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}
}

func TestManager_UnregisterProvider(t *testing.T) {
	m := NewManager()
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
	}

	m.RegisterProvider("tenant-1", cfg)

	err := m.UnregisterProvider("tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = m.GetProvider("tenant-1")
	if err == nil {
		t.Error("expected error after unregistering provider")
	}
}

func TestManager_UnregisterProvider_NotFound(t *testing.T) {
	m := NewManager()

	err := m.UnregisterProvider("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent provider")
	}
}

func TestManager_RegisterProvider_SAML(t *testing.T) {
	m := NewManager()
	cfg := &Config{
		ProviderType: ProviderTypeSAML,
		IssuerURL:    "https://saml.example.com",
		CallbackURL:  "https://app.example.com/saml/callback",
	}

	err := m.RegisterProvider("tenant-saml", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	provider, err := m.GetProvider("tenant-saml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "SAML" {
		t.Errorf("expected provider name 'SAML', got %q", provider.Name())
	}
	if provider.Type() != ProviderTypeSAML {
		t.Errorf("expected provider type %q, got %q", ProviderTypeSAML, provider.Type())
	}
}

func TestManager_RegisterProvider_OIDC(t *testing.T) {
	m := NewManager()
	cfg := &Config{
		ProviderType: ProviderTypeOIDC,
		ClientID:     "oidc-client-id",
		IssuerURL:    "https://oidc.example.com",
	}

	err := m.RegisterProvider("tenant-oidc", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	provider, err := m.GetProvider("tenant-oidc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "OIDC" {
		t.Errorf("expected provider name 'OIDC', got %q", provider.Name())
	}
}

func TestManager_RegisterProvider_LDAP(t *testing.T) {
	m := NewManager()
	cfg := &Config{
		ProviderType: ProviderTypeLDAP,
		IssuerURL:    "ldap://localhost:389",
	}

	err := m.RegisterProvider("tenant-ldap", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	provider, err := m.GetProvider("tenant-ldap")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name() != "LDAP" {
		t.Errorf("expected provider name 'LDAP', got %q", provider.Name())
	}
	if provider.Type() != ProviderTypeLDAP {
		t.Errorf("expected provider type %q, got %q", ProviderTypeLDAP, provider.Type())
	}
}

func TestNewMiddleware(t *testing.T) {
	m := NewManager()
	mw := NewMiddleware(m)
	if mw == nil {
		t.Fatal("NewMiddleware() returned nil")
	}
}

func TestMiddleware_RequireAuth_NoProvider(t *testing.T) {
	m := NewManager()
	mw := NewMiddleware(m)

	err := mw.RequireAuth(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tenant")
	}
}

func TestMiddleware_RequireAuth_WithProvider(t *testing.T) {
	m := NewManager()
	cfg := &Config{
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
	}
	m.RegisterProvider("tenant-1", cfg)

	mw := NewMiddleware(m)
	err := mw.RequireAuth(context.Background(), "tenant-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMiddleware_RequireRole_UserHasRole(t *testing.T) {
	m := NewManager()
	mw := NewMiddleware(m)

	user := &User{
		ID:    "user1",
		Email: "user@example.com",
		Roles: []string{"admin", "editor"},
	}

	err := mw.RequireRole(context.Background(), user, "admin")
	if err != nil {
		t.Errorf("expected no error for user with role, got %v", err)
	}
}

func TestMiddleware_RequireRole_UserMissingRole(t *testing.T) {
	m := NewManager()
	mw := NewMiddleware(m)

	user := &User{
		ID:    "user1",
		Email: "user@example.com",
		Roles: []string{"viewer"},
	}

	err := mw.RequireRole(context.Background(), user, "admin")
	if err == nil {
		t.Error("expected error for user without required role")
	}
}

func TestMiddleware_RequireRole_NilUser(t *testing.T) {
	m := NewManager()
	mw := NewMiddleware(m)

	err := mw.RequireRole(context.Background(), nil, "admin")
	if err == nil {
		t.Error("expected error for nil user")
	}
}

func TestMiddleware_Authenticate(t *testing.T) {
	m := NewManager()
	mw := NewMiddleware(m)

	_, err := mw.Authenticate(context.Background(), "tenant-1")
	if err == nil {
		t.Error("expected error when no provider registered")
	}
}

func TestProviderType_Values(t *testing.T) {
	tests := []struct {
		pt       ProviderType
		expected string
	}{
		{ProviderTypeSAML, "saml"},
		{ProviderTypeOAuth, "oauth"},
		{ProviderTypeOIDC, "oidc"},
		{ProviderTypeLDAP, "ldap"},
	}

	for _, tt := range tests {
		if string(tt.pt) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.pt))
		}
	}
}
