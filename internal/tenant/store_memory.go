package tenant

import (
	"context"
	"fmt"
	"sync"
)

// MemoryStore is an in-process tenant store (single-node / tests).
// For multi-replica production, prefer a Neo4j-backed store.
type MemoryStore struct {
	mu         sync.RWMutex
	tenants    map[string]*Tenant
	bySlug     map[string]string // slug -> id
	members    map[string]map[string]*Membership // tenantID -> userID -> membership
	userIndex  map[string]map[string]struct{}    // userID -> set of tenantIDs
	invites    map[string]*Invite                // id -> invite
	inviteTok  map[string]string                 // token -> invite id
}

// NewMemoryStore creates an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tenants:   make(map[string]*Tenant),
		bySlug:    make(map[string]string),
		members:   make(map[string]map[string]*Membership),
		userIndex: make(map[string]map[string]struct{}),
		invites:   make(map[string]*Invite),
		inviteTok: make(map[string]string),
	}
}

func (s *MemoryStore) CreateTenant(_ context.Context, t *Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[t.ID]; ok {
		return fmt.Errorf("tenant: id already exists")
	}
	if _, ok := s.bySlug[t.Slug]; ok {
		return fmt.Errorf("tenant: slug already exists")
	}
	cp := *t
	s.tenants[t.ID] = &cp
	s.bySlug[t.Slug] = t.ID
	return nil
}

func (s *MemoryStore) GetTenant(_ context.Context, id string) (*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tenants[id]
	if !ok {
		return nil, fmt.Errorf("tenant: not found")
	}
	cp := *t
	return &cp, nil
}

func (s *MemoryStore) GetTenantBySlug(_ context.Context, slug string) (*Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.bySlug[slug]
	if !ok {
		return nil, fmt.Errorf("tenant: not found")
	}
	t := s.tenants[id]
	cp := *t
	return &cp, nil
}

func (s *MemoryStore) UpdateTenant(_ context.Context, t *Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[t.ID]; !ok {
		return fmt.Errorf("tenant: not found")
	}
	cp := *t
	s.tenants[t.ID] = &cp
	s.bySlug[t.Slug] = t.ID
	return nil
}

func (s *MemoryStore) ListTenantsForUser(_ context.Context, userID string) ([]Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.userIndex[userID]
	out := make([]Tenant, 0, len(ids))
	for id := range ids {
		if t, ok := s.tenants[id]; ok {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (s *MemoryStore) ListAllTenants(_ context.Context, limit, offset int) ([]Tenant, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		all = append(all, *t)
	}
	total := len(all)
	if offset > total {
		return []Tenant{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (s *MemoryStore) AddMember(_ context.Context, m *Membership) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[m.TenantID]; !ok {
		return fmt.Errorf("tenant: not found")
	}
	if s.members[m.TenantID] == nil {
		s.members[m.TenantID] = make(map[string]*Membership)
	}
	cp := *m
	s.members[m.TenantID][m.UserID] = &cp
	if s.userIndex[m.UserID] == nil {
		s.userIndex[m.UserID] = make(map[string]struct{})
	}
	s.userIndex[m.UserID][m.TenantID] = struct{}{}
	if t := s.tenants[m.TenantID]; t != nil {
		t.MemberCount = len(s.members[m.TenantID])
	}
	return nil
}

func (s *MemoryStore) RemoveMember(_ context.Context, tenantID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.members[tenantID] != nil {
		delete(s.members[tenantID], userID)
	}
	if s.userIndex[userID] != nil {
		delete(s.userIndex[userID], tenantID)
	}
	if t := s.tenants[tenantID]; t != nil && s.members[tenantID] != nil {
		t.MemberCount = len(s.members[tenantID])
	}
	return nil
}

func (s *MemoryStore) GetMember(_ context.Context, tenantID, userID string) (*Membership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.members[tenantID] == nil {
		return nil, fmt.Errorf("tenant: member not found")
	}
	m, ok := s.members[tenantID][userID]
	if !ok {
		return nil, fmt.Errorf("tenant: member not found")
	}
	cp := *m
	return &cp, nil
}

func (s *MemoryStore) ListMembers(_ context.Context, tenantID string) ([]Membership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Membership, 0)
	for _, m := range s.members[tenantID] {
		out = append(out, *m)
	}
	return out, nil
}

func (s *MemoryStore) SaveInvite(_ context.Context, inv *Invite) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *inv
	s.invites[inv.ID] = &cp
	s.inviteTok[inv.Token] = inv.ID
	return nil
}

func (s *MemoryStore) GetInviteByToken(_ context.Context, token string) (*Invite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.inviteTok[token]
	if !ok {
		return nil, fmt.Errorf("tenant: invite not found")
	}
	inv := s.invites[id]
	if inv == nil {
		return nil, fmt.Errorf("tenant: invite not found")
	}
	cp := *inv
	return &cp, nil
}

func (s *MemoryStore) DeleteInvite(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inv, ok := s.invites[id]; ok {
		delete(s.inviteTok, inv.Token)
		delete(s.invites, id)
	}
	return nil
}

func (s *MemoryStore) ListInvites(_ context.Context, tenantID string) ([]Invite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Invite, 0)
	for _, inv := range s.invites {
		if inv.TenantID == tenantID {
			out = append(out, *inv)
		}
	}
	return out, nil
}
