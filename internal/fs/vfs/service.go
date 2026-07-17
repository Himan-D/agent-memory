package vfs

import (
	"context"
	"fmt"

	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/tenant"
)

// ServiceInterface is the memory backend AgentFS talks to.
type ServiceInterface interface {
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
	GetMemory(ctx context.Context, id string) (*types.Memory, error)
	CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error)
	UpdateMemory(ctx context.Context, id, content string, meta map[string]interface{}) error
	DeleteMemory(ctx context.Context, id string) error
	ListSkills(ctx context.Context, tenantID, domain string, lim, off int) ([]*types.Skill, error)
	ListSessions(ctx context.Context, userID string) ([]*types.Session, error)
	GetEntity(ctx context.Context, id string) (*types.Entity, error)
	GetMemoriesByTenant(ctx context.Context, tenantID string, limit int) ([]*types.Memory, error)
}

// MemoryServiceAdapter wraps *memory.Service for AgentFS.
type MemoryServiceAdapter struct {
	svc      *memory.Service
	tenantID string
}

// NewMemoryServiceAdapter binds a tenant context for all operations.
func NewMemoryServiceAdapter(svc *memory.Service, tenantID string) *MemoryServiceAdapter {
	if tenantID == "" {
		tenantID = "default"
	}
	return &MemoryServiceAdapter{svc: svc, tenantID: tenantID}
}

func (a *MemoryServiceAdapter) ctx(ctx context.Context) context.Context {
	return tenant.WithContext(ctx, tenant.TenantContext{TenantID: a.tenantID})
}

func (a *MemoryServiceAdapter) SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	if req == nil {
		req = &types.SearchRequest{}
	}
	req.TenantID = a.tenantID
	return a.svc.SearchMemories(a.ctx(ctx), req)
}

func (a *MemoryServiceAdapter) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	return a.svc.GetMemory(a.ctx(ctx), id)
}

func (a *MemoryServiceAdapter) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	if mem.TenantID == "" {
		mem.TenantID = a.tenantID
	}
	return a.svc.CreateMemory(a.ctx(ctx), mem)
}

func (a *MemoryServiceAdapter) UpdateMemory(ctx context.Context, id, content string, meta map[string]interface{}) error {
	return a.svc.UpdateMemory(a.ctx(ctx), id, content, meta)
}

func (a *MemoryServiceAdapter) DeleteMemory(ctx context.Context, id string) error {
	return a.svc.DeleteMemory(a.ctx(ctx), id)
}

func (a *MemoryServiceAdapter) ListSkills(ctx context.Context, tenantID, domain string, lim, off int) ([]*types.Skill, error) {
	if tenantID == "" {
		tenantID = a.tenantID
	}
	return a.svc.ListSkills(a.ctx(ctx), tenantID, domain, lim, off)
}

func (a *MemoryServiceAdapter) ListSessions(ctx context.Context, userID string) ([]*types.Session, error) {
	return a.svc.ListSessions(a.ctx(ctx), userID)
}

func (a *MemoryServiceAdapter) GetEntity(ctx context.Context, id string) (*types.Entity, error) {
	return a.svc.GetEntity(id)
}

func (a *MemoryServiceAdapter) GetMemoriesByTenant(ctx context.Context, tenantID string, limit int) ([]*types.Memory, error) {
	if tenantID == "" {
		tenantID = a.tenantID
	}
	return a.svc.GetMemoriesByTenant(a.ctx(ctx), tenantID, limit)
}

// Ensure adapter implements interface
var _ ServiceInterface = (*MemoryServiceAdapter)(nil)

// NullService is a no-op backend for tests.
type NullService struct{}

func (NullService) SearchMemories(context.Context, *types.SearchRequest) ([]types.MemoryResult, error) {
	return nil, nil
}
func (NullService) GetMemory(context.Context, string) (*types.Memory, error) {
	return nil, fmt.Errorf("not found")
}
func (NullService) CreateMemory(context.Context, *types.Memory) (*types.Memory, error) {
	return nil, fmt.Errorf("null service")
}
func (NullService) UpdateMemory(context.Context, string, string, map[string]interface{}) error {
	return fmt.Errorf("null service")
}
func (NullService) DeleteMemory(context.Context, string) error { return fmt.Errorf("null service") }
func (NullService) ListSkills(context.Context, string, string, int, int) ([]*types.Skill, error) {
	return nil, nil
}
func (NullService) ListSessions(context.Context, string) ([]*types.Session, error) { return nil, nil }
func (NullService) GetEntity(context.Context, string) (*types.Entity, error) {
	return nil, fmt.Errorf("not found")
}
func (NullService) GetMemoriesByTenant(context.Context, string, int) ([]*types.Memory, error) {
	return nil, nil
}
