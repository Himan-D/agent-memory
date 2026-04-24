package oauth

import (
	
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type OAuthHandler struct {
	issuer     string
	secretKey  []byte
	clients    map[string]*OAuthClient
	authCodes  map[string]*AuthCode
	accessTokens map[string]*AccessToken
}

type OAuthClient struct {
	ID          string    `json:"id"`
	Secret      string    `json:"client_secret"`
	Name       string    `json:"client_name"`
	RedirectURL string   `json:"redirect_url"`
	CreatedAt  time.Time `json:"created_at"`
}

type AuthCode struct {
	Code        string    `json:"code"`
	ClientID   string    `json:"client_id"`
	RedirectURL string   `json:"redirect_url"`
	Scope      string    `json:"scope"`
	UserID     string    `json:"user_id"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type AccessToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn  int       `json:"expires_in"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	Scope      string    `json:"scope"`
	UserID     string    `json:"user_id"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code        string `json:"code,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	ClientID    string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn  int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope       string `json:"scope"`
}

func NewOAuthHandler(secretKey string) *OAuthHandler {
	return &OAuthHandler{
		issuer:     "hystersis",
		secretKey:  []byte(secretKey),
		clients:    make(map[string]*OAuthClient),
		authCodes:  make(map[string]*AuthCode),
		accessTokens: make(map[string]*AccessToken),
	}
}

func (h *OAuthHandler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	responseType := r.URL.Query().Get("response_type")
	scope := r.URL.Query().Get("scope")
	state := r.URL.Query().Get("state")

	if responseType != "code" {
		http.Error(w, "unsupported response_type", http.StatusBadRequest)
		return
	}

	client, ok := h.clients[clientID]
	if !ok {
		http.Error(w, "invalid client_id", http.StatusBadRequest)
		return
	}

	if redirectURI != client.RedirectURL {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	code := generateCode()
	h.authCodes[code] = &AuthCode{
		Code:        code,
		ClientID:   clientID,
		RedirectURL: redirectURI,
		Scope:      scope,
		UserID:     "default",
		ExpiresAt:  time.Now().Add(10 * time.Minute),
	}

	redirectURL := fmt.Sprintf("%s?code=%s", redirectURI, code)
	if state != "" {
		redirectURL += "&state=" + state
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *OAuthHandler) HandleToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	switch req.GrantType {
	case "authorization_code":
		h.handleAuthorizationCodeGrant(w, r, req)
	case "refresh_token":
		h.handleRefreshTokenGrant(w, r, req)
	default:
		http.Error(w, "unsupported grant_type", http.StatusBadRequest)
	}
}

func (h *OAuthHandler) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request, req TokenRequest) {
	code, ok := h.authCodes[req.Code]
	if !ok {
		http.Error(w, "invalid code", http.StatusBadRequest)
		return
	}

	if time.Now().After(code.ExpiresAt) {
		delete(h.authCodes, req.Code)
		http.Error(w, "code expired", http.StatusBadRequest)
		return
	}

	client, ok := h.clients[req.ClientID]
	if !ok || client.Secret != req.ClientSecret {
		http.Error(w, "invalid client credentials", http.StatusUnauthorized)
		return
	}

	delete(h.authCodes, req.Code)

	token := h.generateJWT(code.UserID, code.Scope)
	refreshToken := generateRefreshToken()

	h.accessTokens[token.AccessToken] = &AccessToken{
		AccessToken:  token.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:  3600,
		RefreshToken: refreshToken,
		Scope:      code.Scope,
		UserID:     code.UserID,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(TokenResponse{
		AccessToken:  token.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:  3600,
		RefreshToken: refreshToken,
		Scope:       code.Scope,
	})
}

func (h *OAuthHandler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request, req TokenRequest) {
	http.Error(w, "refresh_token not implemented", http.StatusNotImplemented)
}

func (h *OAuthHandler) handleRevoke(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}

func (h *OAuthHandler) generateJWT(userID, scope string) *AccessToken {
	now := time.Now()
	exp := now.Add(1 * time.Hour)

	claims := jwt.MapClaims{
		"sub":   userID,
		"scope": scope,
		"iat":   now.Unix(),
		"exp":   exp.Unix(),
		"iss":   h.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(h.secretKey)
	if err != nil {
		return &AccessToken{}
	}

	return &AccessToken{
		AccessToken: tokenString,
		TokenType:  "Bearer",
		ExpiresIn:  3600,
		Scope:      scope,
		UserID:     userID,
		ExpiresAt:  exp,
	}
}

func (h *OAuthHandler) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return h.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (h *OAuthHandler) RegisterClient(name, redirectURL string) *OAuthClient {
	client := &OAuthClient{
		ID:          uuid.New().String(),
		Secret:      generateClientSecret(),
		Name:       name,
		RedirectURL: redirectURL,
		CreatedAt:  time.Now(),
	}
	h.clients[client.ID] = client
	return client
}

func generateCode() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:32]
}

func generateRefreshToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateClientSecret() string {
	b := make([]byte, 40)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

type ProtectedResourceHandler struct {
	oauth *OAuthHandler
}

func NewProtectedResourceHandler(oauth *OAuthHandler) *ProtectedResourceHandler {
	return &ProtectedResourceHandler{oauth: oauth}
}

func (h *ProtectedResourceHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Link", `</.well-known/oauth-protected-resource>; rel="protected-resource"`)

	token := extractBearerToken(r)
	if token == "" {
		http.Error(w, "missing authorization", http.StatusUnauthorized)
		return
	}

	claims, err := h.oauth.ValidateToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"sub":  (*claims)["sub"],
		"scope": (*claims)["scope"],
	})
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}