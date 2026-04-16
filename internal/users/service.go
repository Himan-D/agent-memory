package users

import (
	"fmt"
	"strings"
	"sync"
	"time"

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
	mu     sync.RWMutex
	users  map[uuid.UUID]*User
	invites map[uuid.UUID]*Invite
}

func NewInMemoryStore() *InMemoryStore {
	store := &InMemoryStore{
		users:  make(map[uuid.UUID]*User),
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
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
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
	if err := s.store.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
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