package users

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/auth/password"
	"agent-memory/internal/notification"
	"github.com/google/uuid"
)

type Store interface {
	ListUsers() ([]User, error)
	GetUser(id uuid.UUID) (*User, error)
	CreateUser(user *User) error
	UpdateUser(id uuid.UUID, updates *UpdateUserRequest) error
	DeleteUser(id uuid.UUID) error
	ListInvites() ([]Invite, error)
	GetInvite(id uuid.UUID) (*Invite, error)
	CreateInvite(invite *Invite) error
	UpdateInvite(id uuid.UUID, status string) error
	DeleteInvite(id uuid.UUID) error
}

type InMemoryStore struct {
	mu      sync.RWMutex
	users   map[uuid.UUID]*User
	invites map[uuid.UUID]*Invite
}

func NewInMemoryStore() *InMemoryStore {
	store := &InMemoryStore{
		users:   make(map[uuid.UUID]*User),
		invites: make(map[uuid.UUID]*Invite),
	}
	store.seed()
	return store
}

func (s *InMemoryStore) seed() {
	adminID := uuid.New()
	now := time.Now()
	s.users[adminID] = &User{
		ID:        adminID,
		Email:     "admin@hystersis.io",
		Name:      "System Admin",
		Role:      RoleAdmin,
		Status:    "active",
		AvatarURL: GenerateAvatarURL("System Admin", "admin@hystersis.io"),
		CreatedAt: now,
		UpdatedAt: now,
		LastLogin: &now,
	}
}

func (s *InMemoryStore) ListUsers() ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, *u)
	}
	return users, nil
}

func (s *InMemoryStore) GetUser(id uuid.UUID) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user, ok := s.users[id]; ok {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (s *InMemoryStore) CreateUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	s.users[user.ID] = user
	return nil
}

func (s *InMemoryStore) UpdateUser(id uuid.UUID, updates *UpdateUserRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return fmt.Errorf("user not found")
	}
	if updates.Name != "" {
		user.Name = updates.Name
	}
	if updates.Role != "" {
		user.Role = NormalizeRole(updates.Role)
	}
	if updates.Status != "" {
		user.Status = updates.Status
	}
	if updates.AvatarURL != "" {
		user.AvatarURL = updates.AvatarURL
	}
	if updates.PasswordHash != "" {
		user.PasswordHash = updates.PasswordHash
	}
	user.UpdatedAt = time.Now()
	return nil
}

func (s *InMemoryStore) DeleteUser(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return fmt.Errorf("user not found")
	}
	delete(s.users, id)
	return nil
}

func (s *InMemoryStore) ListInvites() ([]Invite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	invites := make([]Invite, 0, len(s.invites))
	for _, i := range s.invites {
		invites = append(invites, *i)
	}
	return invites, nil
}

func (s *InMemoryStore) CreateInvite(invite *Invite) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if invite.ID == uuid.Nil {
		invite.ID = uuid.New()
	}
	invite.CreatedAt = time.Now()
	invite.Status = "pending"
	s.invites[invite.ID] = invite
	return nil
}

func (s *InMemoryStore) UpdateInvite(id uuid.UUID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	invite, ok := s.invites[id]
	if !ok {
		return fmt.Errorf("invite not found")
	}
	invite.Status = status
	return nil
}

func (s *InMemoryStore) DeleteInvite(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.invites[id]; !ok {
		return fmt.Errorf("invite not found")
	}
	delete(s.invites, id)
	return nil
}

func (s *InMemoryStore) GetInvite(id uuid.UUID) (*Invite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if invite, ok := s.invites[id]; ok {
		return invite, nil
	}
	return nil, fmt.Errorf("invite not found")
}

type Service struct {
	store    Store
	notifSvc *notification.Service
}

func NewService(store Store, notifSvc ...*notification.Service) *Service {
	s := &Service{store: store}
	if len(notifSvc) > 0 && notifSvc[0] != nil {
		s.notifSvc = notifSvc[0]
	}
	return s
}

func (s *Service) ListUsers() ([]User, error) {
	return s.store.ListUsers()
}

func (s *Service) GetUser(id uuid.UUID) (*User, error) {
	return s.store.GetUser(id)
}

func (s *Service) CreateUser(req *CreateUserRequest) (*User, error) {
	role := NormalizeRole(req.Role)
	if role == "" {
		role = RoleMember
	}
	user := &User{
		ID:        uuid.New(),
		Email:     req.Email,
		Name:      req.Name,
		Role:      role,
		Status:    "active",
		AvatarURL: GenerateAvatarURL(req.Name, req.Email),
	}
	if req.Password != "" {
		hash, err := password.Hash(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = hash
	}
	if err := s.store.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) Authenticate(email, pwd string) (*User, error) {
	users, err := s.store.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}
	for i := range users {
		if users[i].Email == email {
			if users[i].PasswordHash == "" {
				return nil, fmt.Errorf("account not configured for password login")
			}
			ok, err := password.Verify(pwd, users[i].PasswordHash)
			if err != nil {
				return nil, fmt.Errorf("authentication failed")
			}
			if !ok {
				return nil, fmt.Errorf("invalid email or password")
			}
			if users[i].AvatarURL == "" {
				users[i].AvatarURL = GenerateAvatarURL(users[i].Name, users[i].Email)
				_ = s.store.UpdateUser(users[i].ID, &UpdateUserRequest{AvatarURL: users[i].AvatarURL})
			}
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("invalid email or password")
}

func (s *Service) UpdateUser(id uuid.UUID, req *UpdateUserRequest) (*User, error) {
	if req.Role != "" {
		req.Role = NormalizeRole(req.Role)
	}
	if err := s.store.UpdateUser(id, req); err != nil {
		return nil, err
	}
	return s.store.GetUser(id)
}

func (s *Service) DeleteUser(id uuid.UUID) error {
	return s.store.DeleteUser(id)
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	user, err := s.store.GetUser(uid)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password
	ok, err := password.Verify(currentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("authentication failed")
	}
	if !ok {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hash, err := password.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.store.UpdateUser(uid, &UpdateUserRequest{PasswordHash: hash})
}

func GenerateAvatarURL(name, email string) string {
	seed := strings.TrimSpace(email)
	if seed == "" {
		seed = strings.TrimSpace(name)
	}
	if seed == "" {
		seed = "hystersis-user"
	}
	sum := sha256.Sum256([]byte(strings.ToLower(seed)))
	hexSeed := hex.EncodeToString(sum[:])
	palette := [][2]string{
		{"0f766e", "d9f99d"},
		{"1d4ed8", "bfdbfe"},
		{"be123c", "ffe4e6"},
		{"7c2d12", "fed7aa"},
		{"4338ca", "ddd6fe"},
		{"166534", "bbf7d0"},
	}
	choice := int(sum[0]) % len(palette)
	initials := avatarInitials(name, email)
	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128" viewBox="0 0 128 128"><rect width="128" height="128" rx="32" fill="#%s"/><circle cx="100" cy="28" r="34" fill="#%s" opacity=".24"/><text x="64" y="76" text-anchor="middle" font-family="Arial, sans-serif" font-size="42" font-weight="700" fill="#%s">%s</text></svg>`,
		palette[choice][0],
		hexSeed[:6],
		palette[choice][1],
		html.EscapeString(initials),
	)
	return "data:image/svg+xml," + url.PathEscape(svg)
}

func avatarInitials(name, email string) string {
	source := strings.TrimSpace(name)
	if source == "" {
		source = strings.TrimSpace(strings.Split(email, "@")[0])
	}
	if source == "" {
		return "HU"
	}
	parts := strings.Fields(source)
	if len(parts) == 1 {
		runes := []rune(parts[0])
		if len(runes) == 1 {
			return strings.ToUpper(string(runes[0]))
		}
		return strings.ToUpper(string(runes[0:2]))
	}
	first := []rune(parts[0])
	last := []rune(parts[len(parts)-1])
	return strings.ToUpper(string([]rune{first[0], last[0]}))
}

func (s *Service) ListInvites() ([]Invite, error) {
	return s.store.ListInvites()
}

func (s *Service) CreateInvite(req *CreateInviteRequest, invitedBy uuid.UUID) (*Invite, error) {
	role := NormalizeRole(req.Role)
	if role == "" {
		role = RoleMember
	}
	invite := &Invite{
		ID:        uuid.New(),
		Email:     req.Email,
		Role:      role,
		Status:    "pending",
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(72 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := s.store.CreateInvite(invite); err != nil {
		return nil, err
	}

	// Send email notification if notification service is available
	if s.notifSvc != nil {
		inviteLink := fmt.Sprintf("https://hystersis.com/auth/signup?invite=%s", invite.ID.String())
		emailMsg := fmt.Sprintf(`
		<html>
		<body>
			<h2>You've been invited to Hystersis</h2>
			<p>You have been invited to join Hystersis as a <strong>%s</strong>.</p>
			<p>Click the link below to accept the invitation:</p>
			<p><a href="%s">Accept Invitation</a></p>
			<p>This invitation expires on %s.</p>
			<p>If you didn't expect this invitation, you can safely ignore this email.</p>
		</body>
		</html>
		`, invite.Role, inviteLink, invite.ExpiresAt.Format("January 2, 2006"))

		notifReq := notification.CreateNotificationRequest{
			UserID:  invite.Email,
			Type:    notification.NotificationTypeInfo,
			Title:   "Invitation to Hystersis",
			Message: emailMsg,
			Channel: notification.ChannelEmail,
			Data: map[string]interface{}{
				"email":     invite.Email,
				"invite_id": invite.ID.String(),
			},
			ExpiresIn: func() *time.Duration { d := 72 * time.Hour; return &d }(),
		}
		ctx := context.Background()
		s.notifSvc.Create(ctx, "default", notifReq)
	}

	return invite, nil
}

func NormalizeRole(role Role) Role {
	switch strings.ToLower(strings.TrimSpace(string(role))) {
	case "admin":
		return RoleAdmin
	case "viewer", "read", "readonly":
		return RoleViewer
	case "member", "user", "editor", "write":
		return RoleMember
	default:
		return ""
	}
}

func (s *Service) AcceptInvite(id uuid.UUID) error {
	invite, err := s.store.GetInvite(id)
	if err != nil {
		return err
	}
	if invite.Status != "pending" {
		return fmt.Errorf("invite is not pending")
	}
	if time.Now().After(invite.ExpiresAt) {
		return fmt.Errorf("invite has expired")
	}
	user := &User{
		Email:  invite.Email,
		Name:   invite.Email[:strings.Index(invite.Email, "@")],
		Role:   invite.Role,
		Status: "active",
	}
	if err := s.store.CreateUser(user); err != nil {
		return err
	}

	// Send welcome email if notification service is available
	if s.notifSvc != nil {
		welcomeMsg := fmt.Sprintf(`
		<html>
		<body>
			<h2>Welcome to Hystersis!</h2>
			<p>Hi %s,</p>
			<p>Your account has been successfully created. You now have <strong>%s</strong> access to the Hystersis platform.</p>
			<p>You can now <a href="https://hystersis.com/auth/signin">sign in</a> to your account.</p>
			<p>Get started by exploring the dashboard and setting up your first memory agent.</p>
		</body>
		</html>
		`, user.Name, user.Role)

		notifReq := notification.CreateNotificationRequest{
			UserID:  user.Email,
			Type:    notification.NotificationTypeSuccess,
			Title:   "Welcome to Hystersis!",
			Message: welcomeMsg,
			Channel: notification.ChannelEmail,
			Data: map[string]interface{}{
				"email": user.Email,
			},
		}
		ctx := context.Background()
		s.notifSvc.Create(ctx, "default", notifReq)
	}

	return s.store.UpdateInvite(id, "accepted")
}

func (s *Service) CancelInvite(id uuid.UUID) error {
	return s.store.DeleteInvite(id)
}

func (s *Service) GetInvite(id uuid.UUID) (*Invite, error) {
	invites, err := s.store.ListInvites()
	if err != nil {
		return nil, err
	}
	for _, inv := range invites {
		if inv.ID == id {
			return &inv, nil
		}
	}
	return nil, fmt.Errorf("invite not found")
}

type ListUsersParams struct {
	Page     int
	PageSize int
	Role     string
	Status   string
	Search   string
}

func (s *Service) ListUsersFiltered(params ListUsersParams) (*UserListResponse, error) {
	users, err := s.store.ListUsers()
	if err != nil {
		return nil, err
	}

	if params.PageSize == 0 {
		params.PageSize = 20
	}
	if params.Page == 0 {
		params.Page = 1
	}

	start := (params.Page - 1) * params.PageSize
	end := start + params.PageSize

	if start > len(users) {
		start = len(users)
	}
	if end > len(users) {
		end = len(users)
	}

	result := users[start:end]
	if result == nil {
		result = []User{}
	}

	return &UserListResponse{
		Users:    result,
		Total:    len(users),
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}
