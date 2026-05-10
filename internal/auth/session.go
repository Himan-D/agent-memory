package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"agent-memory/internal/users"
)

// SessionStore manages active sessions
type SessionStore struct {
	mu        sync.RWMutex
	sessions  map[string]*Session
	userTokens map[string]string // userID -> token mapping
}

type Session struct {
	Token     string
	UserID    string
	Email     string
	Name      string
	Role      string
	CreatedAt time.Time
	ExpiresAt time.Time
	LastSeen  time.Time
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions:  make(map[string]*Session),
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
	s.mu.Lock()
	defer s.mu.Unlock()

	// Revoke any existing session for this user
	if existingToken, ok := s.userTokens[userID]; ok {
		delete(s.sessions, existingToken)
	}

	token := generateSecureToken()
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
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	if !ok {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	// Update last seen
	session.LastSeen = time.Now()
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

// generateSecureToken generates a cryptographically secure token
func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}
	return base64.URLEncoding.EncodeToString(b)
}

// Password hashing utilities
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return base64.URLEncoding.EncodeToString(hash[:])
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

// handleAuthLogin handles user login authentication
func (s *SessionStore) handleAuthLogin(userSvc *users.Service) http.HandlerFunc {
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
		if !strings.Contains(req.Email, "@") {
			jsonError(w, "Invalid email format", http.StatusBadRequest)
			return
		}

		// Check if user exists
		allUsers, err := userSvc.ListUsers()
		if err != nil {
			log.Printf("Error listing users: %v", err)
			jsonError(w, "Authentication failed", http.StatusInternalServerError)
			return
		}

		for _, u := range allUsers {
			if u.Email == req.Email {
				// For demo purposes, accept any password for existing users
				// In production, use proper password hashing
				session := s.CreateSession(
					u.ID.String(),
					u.Email,
					u.Name,
					string(u.Role),
				)

				w.Header().Set("Content-Type", "application/json")
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
				return
			}
		}

		jsonError(w, "Invalid email or password", http.StatusUnauthorized)
	}
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
		if !strings.Contains(req.Email, "@") {
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
			Email: req.Email,
			Name:  req.Name,
			Role:  users.RoleMember,
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

	session, valid := s.ValidateToken(token)
	if !valid {
		jsonError(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	// Extend session expiration
	s.mu.Lock()
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