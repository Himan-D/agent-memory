package users

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/notification"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
		user.Role = updates.Role
	}
	if updates.Status != "" {
		user.Status = updates.Status
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
	user := &User{
		Email:  req.Email,
		Name:   req.Name,
		Role:   req.Role,
		Status: "active",
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = string(hash)
	}
	if err := s.store.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) Authenticate(email, password string) (*User, error) {
	users, err := s.store.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}
	for i := range users {
		if users[i].Email == email {
			if users[i].PasswordHash == "" {
				return &users[i], nil
			}
			if err := bcrypt.CompareHashAndPassword([]byte(users[i].PasswordHash), []byte(password)); err != nil {
				return nil, fmt.Errorf("invalid email or password")
			}
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("invalid email or password")
}

func (s *Service) UpdateUser(id uuid.UUID, req *UpdateUserRequest) (*User, error) {
	if err := s.store.UpdateUser(id, req); err != nil {
		return nil, err
	}
	return s.store.GetUser(id)
}

func (s *Service) DeleteUser(id uuid.UUID) error {
	return s.store.DeleteUser(id)
}

func (s *Service) ListInvites() ([]Invite, error) {
	return s.store.ListInvites()
}

func (s *Service) CreateInvite(req *CreateInviteRequest, invitedBy uuid.UUID) (*Invite, error) {
	invite := &Invite{
		Email:     req.Email,
		Role:      req.Role,
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(72 * time.Hour),
	}
	if err := s.store.CreateInvite(invite); err != nil {
		return nil, err
	}

	// Send email notification if notification service is available
	if s.notifSvc != nil {
		inviteLink := fmt.Sprintf("https://hystersis.ai/auth/signup?invite=%s", invite.ID.String())
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
			<p>You can now <a href="https://hystersis.ai/auth/signin">sign in</a> to your account.</p>
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
