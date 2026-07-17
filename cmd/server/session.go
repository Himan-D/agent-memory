package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/auth"
	"agent-memory/internal/config"
	"agent-memory/internal/memory/neo4j"
	"agent-memory/internal/tenant"
	"agent-memory/internal/users"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SessionStore manages active sessions. When redis is set, all session ops
// delegate to Redis for multi-replica deployments.
type SessionStore struct {
	mu         sync.RWMutex
	sessions   map[string]*Session
	userTokens map[string]string // userID -> token mapping
	redis      *RedisSessionStore
}

// SetRedisBackend enables Redis-backed sessions (multi-replica).
func (s *SessionStore) SetRedisBackend(r *RedisSessionStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redis = r
}

type Session struct {
	Token           string
	UserID          string
	Email           string
	Name            string
	Role            string
	ActiveTenantID  string // current tenant for multi-tenant sessions
	CreatedAt       time.Time
	ExpiresAt       time.Time
	LastSeen        time.Time
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions:   make(map[string]*Session),
		userTokens: make(map[string]string),
	}
}

// CleanupLoop removes expired sessions every hour
func (s *SessionStore) CleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanupExpired()
	}
}

func (s *SessionStore) cleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, token)
			if session.UserID != "" {
				delete(s.userTokens, session.UserID)
			}
		}
	}
}

// CreateSession creates a new session for a user
func (s *SessionStore) CreateSession(userID, email, name, role string) *Session {
	s.mu.RLock()
	redis := s.redis
	s.mu.RUnlock()
	if redis != nil {
		return redis.CreateSession(userID, email, name, role)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Revoke any existing session for this user
	if existingToken, ok := s.userTokens[userID]; ok {
		delete(s.sessions, existingToken)
	}

	token, err := generateSecureToken()
	if err != nil {
		token = uuid.New().String()
	}
	now := time.Now()
	session := &Session{
		Token:     token,
		UserID:    userID,
		Email:     email,
		Name:      name,
		Role:      role,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
		LastSeen:  now,
	}

	s.sessions[token] = session
	s.userTokens[userID] = token

	return session
}

// ValidateToken validates a session token and returns the session
func (s *SessionStore) ValidateToken(token string) (*Session, bool) {
	s.mu.RLock()
	redis := s.redis
	s.mu.RUnlock()
	if redis != nil {
		return redis.ValidateToken(token)
	}

	s.mu.RLock()
	session, ok := s.sessions[token]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	s.mu.Lock()
	session.LastSeen = time.Now()
	s.mu.Unlock()

	return session, true
}

// RevokeSession revokes a session by token
func (s *SessionStore) RevokeSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[token]
	if ok {
		delete(s.sessions, token)
		if session.UserID != "" {
			delete(s.userTokens, session.UserID)
		}
	}
}

// RevokeUserSessions revokes all sessions for a user
func (s *SessionStore) RevokeUserSessions(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token, ok := s.userTokens[userID]; ok {
		delete(s.sessions, token)
		delete(s.userTokens, userID)
	}
}

// GetUserFromToken extracts user info from a valid token
func (s *SessionStore) GetUserFromToken(token string) (map[string]interface{}, bool) {
	session, valid := s.ValidateToken(token)
	if !valid {
		return nil, false
	}

	return map[string]interface{}{
		"id":    session.UserID,
		"email": session.Email,
		"name":  session.Name,
		"role":  session.Role,
	}, true
}

// generateSecureToken generates a cryptographically secure session token.
func generateSecureToken() (string, error) {
	return auth.GenerateSessionToken()
}

func defaultAllowedOrigins() map[string]bool {
	return map[string]bool{
		"http://localhost:5173":     true,
		"http://localhost:3000":     true,
		"http://localhost:8080":     true,
		"https://hystersis.com":     true,
		"https://www.hystersis.com": true,
		"https://app.hystersis.com": true,
		"https://api.hystersis.com": true,
	}
}

func allowedOriginsFor(cfg *config.Config) map[string]bool {
	origins := defaultAllowedOrigins()
	if cfg != nil {
		for _, o := range cfg.Auth.AllowedOrigins {
			if o != "" && o != "*" {
				origins[o] = true
			}
		}
	}
	return origins
}

// Password hashing utilities
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return base64.URLEncoding.EncodeToString([]byte(password))
	}
	return string(hash)
}

func checkPassword(hashedPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

// AuthRequest represents authentication request data
type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Success bool                   `json:"success"`
	Token   string                 `json:"token,omitempty"`
	User    map[string]interface{} `json:"user,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// authMiddleware creates middleware for protected routes
func (s *SessionStore) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			jsonError(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		session, valid := s.ValidateToken(token)
		if !valid {
			jsonError(w, "Invalid or expired session", http.StatusUnauthorized)
			return
		}

		// Add user info to request context/headers for downstream handlers
		r.Header.Set("X-User-ID", session.UserID)
		r.Header.Set("X-User-Email", session.Email)
		r.Header.Set("X-User-Name", session.Name)
		r.Header.Set("X-User-Role", session.Role)

		next(w, r)
	}
}

// authMiddlewareWithRoles creates middleware for role-based access
func (s *SessionStore) authMiddlewareWithRoles(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				jsonError(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			session, valid := s.ValidateToken(token)
			if !valid {
				jsonError(w, "Invalid or expired session", http.StatusUnauthorized)
				return
			}

			// Check if user has one of the allowed roles
			hasRole := false
			for _, role := range allowedRoles {
				if session.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				jsonError(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add user info to request
			r.Header.Set("X-User-ID", session.UserID)
			r.Header.Set("X-User-Email", session.Email)
			r.Header.Set("X-User-Name", session.Name)
			r.Header.Set("X-User-Role", session.Role)

			next(w, r)
		}
	}
}

// scopeToRole maps API key scopes to RBAC roles
func scopeToRole(scope string) string {
	switch scope {
	case "admin":
		return "admin"
	case "write":
		return "editor"
	case "read":
		return "viewer"
	default:
		return "user"
	}
}

// extractToken extracts the token from the Authorization header or cookie
func extractToken(r *http.Request) string {
	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Check session cookie
	cookie, err := r.Cookie("session_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}

// handleAuthRegister handles user registration
func (s *SessionStore) handleAuthRegister(userSvc *users.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" {
			jsonError(w, "Email and password are required", http.StatusBadRequest)
			return
		}

		// Validate email format
		if !isValidEmail(req.Email) {
			jsonError(w, "Invalid email format", http.StatusBadRequest)
			return
		}

		// Validate password length
		if len(req.Password) < 6 {
			jsonError(w, "Password must be at least 6 characters", http.StatusBadRequest)
			return
		}

		// Extract name from email if not provided
		if req.Name == "" {
			parts := strings.Split(req.Email, "@")
			req.Name = parts[0]
		}

		// Check if user already exists
		allUsers, err := userSvc.ListUsers()
		if err != nil {
			log.Printf("Error listing users: %v", err)
			jsonError(w, "Registration failed", http.StatusInternalServerError)
			return
		}

		for _, u := range allUsers {
			if u.Email == req.Email {
				jsonError(w, "User with this email already exists", http.StatusConflict)
				return
			}
		}

		// Create new user
		user, err := userSvc.CreateUser(&users.CreateUserRequest{
			Email:    req.Email,
			Name:     req.Name,
			Role:     users.RoleMember,
			Password: req.Password,
		})
		if err != nil {
			log.Printf("Error creating user: %v", err)
			jsonError(w, "Registration failed", http.StatusInternalServerError)
			return
		}

		// Create session
		session := s.CreateSession(
			user.ID.String(),
			user.Email,
			user.Name,
			string(user.Role),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(AuthResponse{
			Success: true,
			Token:   session.Token,
			User: map[string]interface{}{
				"id":    session.UserID,
				"name":  session.Name,
				"email": session.Email,
				"role":  session.Role,
			},
		})
	}
}

// handleAuthLogout handles user logout
func (s *SessionStore) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token != "" {
		s.RevokeSession(token)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"success": "true",
		"message": "Logged out successfully",
	})
}

// handleAuthMe returns current user info from token
func (s *SessionStore) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		jsonError(w, "No session token provided", http.StatusUnauthorized)
		return
	}

	userInfo, valid := s.GetUserFromToken(token)
	if !valid {
		jsonError(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    userInfo,
	})
}

// handleAuthRefresh refreshes an existing session
func (s *SessionStore) handleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		jsonError(w, "No session token provided", http.StatusUnauthorized)
		return
	}

	s.mu.Lock()
	session, ok := s.sessions[token]
	if !ok {
		s.mu.Unlock()
		jsonError(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, token)
		s.mu.Unlock()
		jsonError(w, "Session expired", http.StatusUnauthorized)
		return
	}

	session.ExpiresAt = time.Now().Add(24 * time.Hour)
	session.LastSeen = time.Now()
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   session.Token,
		"user": map[string]interface{}{
			"id":    session.UserID,
			"email": session.Email,
			"name":  session.Name,
			"role":  session.Role,
		},
	})
}

// routerAuthMiddleware creates middleware that validates both API keys and session tokens
func (s *SessionStore) routerAuthMiddleware(cfg *config.Config, store neo4j.APIKeyStore) func(http.Handler) http.Handler {
	apiKeys := make(map[string]string)
	for _, key := range cfg.Auth.APIKeys {
		parts := splitKey(key)
		if len(parts) == 2 {
			apiKeys[parts[0]] = parts[1]
		} else {
			apiKeys[key] = "default"
		}
	}

	adminKeys := make(map[string]bool)
	for _, key := range cfg.Auth.AdminAPIKeys {
		adminKeys[key] = true
	}

	return func(next http.Handler) http.Handler {
		allowedOrigins := allowedOriginsFor(cfg)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				origin := r.Header.Get("Origin")
				if allowedOrigins[origin] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusOK)
				return
			}

			if isPublicAuthPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Check for session token first
			authHeader := r.Header.Get("Authorization")
			sessionToken := ""
			if authHeader != "" && strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				sessionToken = strings.TrimPrefix(authHeader, "Bearer ")
				sessionToken = strings.TrimPrefix(sessionToken, "bearer ")
			}
			if sessionToken == "" {
				sessionToken = r.URL.Query().Get("token")
			}

			boundTenant := ""
			userID := ""
			isAdmin := false
			valid := false
			keyScope := ""

			// Validate session token if present
			if sessionToken != "" {
				session, validSession := s.ValidateToken(sessionToken)
				if validSession {
					userID = session.UserID
					// Prefer explicit active tenant; do not treat user ID as tenant.
					if session.ActiveTenantID != "" {
						boundTenant = session.ActiveTenantID
					} else {
						// Legacy fallback for single-tenant deploys
						boundTenant = session.UserID
					}
					isAdmin = session.Role == "admin" || session.Role == "Admin"
					valid = true
					if isAdmin {
						keyScope = "admin"
					} else {
						keyScope = "write"
					}
				}
			}

			// If no valid session, fall back to API key validation
			if !valid {
				apiKey := r.Header.Get("X-API-Key")

				if adminKeys[apiKey] {
					boundTenant = "admin"
					isAdmin = true
					valid = true
					keyScope = "admin"
				} else if boundTenant = apiKeys[apiKey]; boundTenant != "" {
					valid = true
					keyScope = "write"
				} else if store != nil {
					storedKey, err := store.GetByKey(r.Context(), apiKey)
					if err == nil && storedKey != nil && !storedKey.IsExpired() {
						boundTenant = storedKey.TenantID
						keyScope = storedKey.Scope
						valid = true
					}
				}
			}

			if !valid {
				http.Error(w, "Unauthorized: Invalid or missing credentials", http.StatusUnauthorized)
				return
			}

			// Multi-tenant isolation: resolve acting tenant; reject spoof for non-admin.
			isolation := tenant.IsolationStrict
			defaultTenant := "default"
			if cfg != nil {
				if strings.EqualFold(cfg.Tenant.Isolation, "off") {
					isolation = tenant.IsolationOff
				}
				if cfg.Tenant.DefaultTenantID != "" {
					defaultTenant = cfg.Tenant.DefaultTenantID
				}
			}
			headerTenant := r.Header.Get("X-Tenant-ID")
			if headerTenant == "" {
				headerTenant = r.URL.Query().Get("tenant_id")
			}
			actingTenant, err := tenant.ResolveActingTenant(boundTenant, headerTenant, isAdmin, isolation, defaultTenant)
			if err == tenant.ErrTenantSpoof {
				http.Error(w, "Forbidden: cannot override tenant for this credential", http.StatusForbidden)
				return
			}
			if err != nil && isolation == tenant.IsolationStrict {
				http.Error(w, "Unauthorized: tenant required", http.StatusUnauthorized)
				return
			}
			if actingTenant == "" {
				actingTenant = defaultTenant
			}

			var keyScopes []string
			if keyScope != "" {
				keyScopes = strings.Split(keyScope, ",")
				for i, s := range keyScopes {
					keyScopes[i] = strings.TrimSpace(s)
				}
			}
			if len(keyScopes) == 0 {
				if isAdmin {
					keyScopes = []string{"admin"}
				} else {
					keyScopes = []string{"write"}
				}
			}

			role := "user"
			if isAdmin {
				role = "admin"
			} else if sessionToken != "" {
				if session, ok := s.ValidateToken(sessionToken); ok {
					role = session.Role
				}
			} else {
				role = scopeToRole(keyScope)
			}

			tc := tenant.TenantContext{
				TenantID: actingTenant,
				UserID:   userID,
				IsAdmin:  isAdmin,
				KeyScope: keyScope,
			}
			if isAdmin {
				tc.Role = tenant.RoleAdmin
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "tenant_id", actingTenant)
			ctx = context.WithValue(ctx, "is_admin", isAdmin)
			ctx = context.WithValue(ctx, "key_scope", keyScope)
			ctx = context.WithValue(ctx, "key_scopes", keyScopes)
			ctx = context.WithValue(ctx, "role", role)
			if userID != "" {
				ctx = context.WithValue(ctx, "user_id", userID)
			}
			ctx = tenant.WithContext(ctx, tc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SetActiveTenant updates the session's active tenant (caller must validate membership).
func (s *SessionStore) SetActiveTenant(token, tenantID string) bool {
	s.mu.RLock()
	redis := s.redis
	s.mu.RUnlock()
	if redis != nil {
		return redis.SetActiveTenant(token, tenantID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[token]
	if !ok {
		return false
	}
	session.ActiveTenantID = tenantID
	session.LastSeen = time.Now()
	return true
}
