package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type OIDCProvider struct {
	config     *Config
	client     *http.Client
	jwks       *JWKS
	jwksMutex  sync.RWMutex
	jwksExpiry time.Time
}

type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JwksURI               string `json:"jwks_uri"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type IDTokenClaims struct {
	jwt.RegisteredClaims
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Groups            []string `json:"groups"`
	Roles             []string `json:"roles"`
}

func NewOIDCProvider(cfg *Config) (*OIDCProvider, error) {
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("OIDC client ID is required")
	}
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("OIDC issuer URL is required")
	}

	return &OIDCProvider{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (p *OIDCProvider) Name() string {
	return "OIDC"
}

func (p *OIDCProvider) Type() ProviderType {
	return ProviderTypeOIDC
}

func (p *OIDCProvider) Authenticate(ctx context.Context, code string) (*User, error) {
	discovery, err := p.getDiscovery(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch discovery document: %w", err)
	}

	tokenResp, err := p.exchangeCode(ctx, discovery.TokenEndpoint, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	claims, err := p.validateIDToken(ctx, tokenResp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("validate ID token: %w", err)
	}

	session := &Session{
		ID:        generateSessionID(),
		UserID:    claims.Subject,
		TenantID:  p.config.TenantID,
		Token:     tokenResp.AccessToken,
		ExpiresAt: time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339),
	}

	user := &User{
		ID:        claims.Subject,
		Email:     claims.Email,
		Name:      claims.Name,
		TenantID:  p.config.TenantID,
		Roles:     claims.Roles,
		Groups:    claims.Groups,
		ExpiresAt: &session.ExpiresAt,
	}

	p.cacheUser(session.ID, user)

	return user, nil
}

func (p *OIDCProvider) GetLogoutURL(redirectURL string) (string, error) {
	discovery, err := p.getDiscovery(context.Background())
	if err != nil {
		return fmt.Sprintf("%s/oauth/logout?redirect=%s", p.config.IssuerURL, redirectURL), nil
	}

	logoutURL := discovery.Issuer + "/oauth/logout"
	if redirectURL != "" {
		logoutURL += "?redirect_uri=" + redirectURL
	}
	return logoutURL, nil
}

func (p *OIDCProvider) ValidateSession(ctx context.Context, token string) (*Session, error) {
	user := p.getUserFromCache(token)
	if user == nil {
		return nil, fmt.Errorf("session not found or expired")
	}

	claims, err := p.parseAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("parse access token: %w", err)
	}

	expTime, err := claims.GetExpirationTime()
	if err != nil || expTime == nil {
		return nil, fmt.Errorf("invalid expiration")
	}

	if expTime.Before(time.Now()) {
		p.removeUserFromCache(token)
		return nil, fmt.Errorf("session expired")
	}

	return &Session{
		ID:       token,
		UserID:   claims.Subject,
		TenantID: p.config.TenantID,
		Token:    token,
	}, nil
}

func (p *OIDCProvider) RefreshSession(ctx context.Context, token string) (*Session, error) {
	discovery, err := p.getDiscovery(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch discovery: %w", err)
	}

	reqBody := strings.NewReader(fmt.Sprintf(
		"grant_type=refresh_token&refresh_token=%s&client_id=%s",
		token, p.config.ClientID,
	))

	req, err := http.NewRequestWithContext(ctx, "POST", discovery.TokenEndpoint, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if p.config.ClientSecret != "" {
		req.SetBasicAuth(p.config.ClientID, p.config.ClientSecret)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &Session{
		ID:        generateSessionID(),
		UserID:    p.config.TenantID,
		TenantID:  p.config.TenantID,
		Token:     tokenResp.AccessToken,
		ExpiresAt: time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339),
	}, nil
}

func (p *OIDCProvider) getDiscovery(ctx context.Context) (*OIDCDiscovery, error) {
	url := p.config.IssuerURL + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery request failed: %d", resp.StatusCode)
	}

	var discovery OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, err
	}

	return &discovery, nil
}

func (p *OIDCProvider) exchangeCode(ctx context.Context, tokenURL, code string) (*TokenResponse, error) {
	reqBody := strings.NewReader(fmt.Sprintf(
		"grant_type=authorization_code&code=%s&redirect_uri=%s&client_id=%s",
		code, p.config.CallbackURL, p.config.ClientID,
	))

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if p.config.ClientSecret != "" {
		req.SetBasicAuth(p.config.ClientID, p.config.ClientSecret)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (p *OIDCProvider) validateIDToken(ctx context.Context, idToken string) (*IDTokenClaims, error) {
	claims := &IDTokenClaims{}
	_, err := jwt.ParseWithClaims(idToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
				return []byte(p.config.ClientSecret), nil
			}
			return nil, fmt.Errorf("unexpected signing method")
		}
		return p.getSigningKey(token.Header["kid"].(string))
	})
	if err != nil {
		return nil, err
	}

	issuer, _ := claims.GetIssuer()
	if issuer != p.config.IssuerURL {
		return nil, fmt.Errorf("invalid issuer")
	}

	aud, _ := claims.GetAudience()
	if !containsString(aud, p.config.ClientID) {
		return nil, fmt.Errorf("invalid audience")
	}

	return claims, nil
}

func (p *OIDCProvider) parseAccessToken(token string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
				return []byte(p.config.ClientSecret), nil
			}
			return nil, fmt.Errorf("unexpected signing method")
		}
		return p.getSigningKey(token.Header["kid"].(string))
	})
	return claims, err
}

func (p *OIDCProvider) getSigningKey(kid string) (interface{}, error) {
	p.jwksMutex.RLock()
	if time.Now().Before(p.jwksExpiry) && p.jwks != nil {
		p.jwksMutex.RUnlock()
		for _, key := range p.jwks.Keys {
			if key.Kid == kid {
				return keyToRSA(key)
			}
		}
	}
	p.jwksMutex.RUnlock()

	if err := p.refreshJWKS(context.Background()); err != nil {
		return nil, err
	}

	p.jwksMutex.RLock()
	defer p.jwksMutex.RUnlock()
	for _, key := range p.jwks.Keys {
		if key.Kid == kid {
			return keyToRSA(key)
		}
	}

	return nil, fmt.Errorf("signing key not found: %s", kid)
}

func (p *OIDCProvider) refreshJWKS(ctx context.Context) error {
	discovery, err := p.getDiscovery(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", discovery.JwksURI, nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS request failed: %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	p.jwksMutex.Lock()
	p.jwks = &jwks
	p.jwksExpiry = time.Now().Add(1 * time.Hour)
	p.jwksMutex.Unlock()

	return nil
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
}

type userCache struct {
	user   *User
	expiry time.Time
}

var (
	userCacheStore = make(map[string]*userCache)
	userCacheMu    sync.RWMutex
)

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (p *OIDCProvider) cacheUser(sessionID string, user *User) {
	userCacheMu.Lock()
	defer userCacheMu.Unlock()
	userCacheStore[sessionID] = &userCache{
		user:   user,
		expiry: time.Now().Add(24 * time.Hour),
	}
}

func (p *OIDCProvider) getUserFromCache(sessionID string) *User {
	userCacheMu.RLock()
	defer userCacheMu.RUnlock()
	if uc, ok := userCacheStore[sessionID]; ok && time.Now().Before(uc.expiry) {
		return uc.user
	}
	return nil
}

func (p *OIDCProvider) removeUserFromCache(sessionID string) {
	userCacheMu.Lock()
	defer userCacheMu.Unlock()
	delete(userCacheStore, sessionID)
}

func keyToRSA(jwk JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("decode N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("decode E: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
