package project

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

type Service struct {
	projects map[string]*types.Project
	mu       sync.RWMutex
	cfg      *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		projects: make(map[string]*types.Project),
		cfg:      cfg,
	}
}

func (s *Service) CreateProject(ctx context.Context, proj *types.Project) (*types.Project, error) {
	if proj.ID == "" {
		proj.ID = uuid.New().String()
	}
	proj.CreatedAt = time.Now()
	proj.UpdatedAt = time.Now()

	if proj.Settings.MemoryTypes == nil {
		proj.Settings.MemoryTypes = []types.MemoryType{
			types.MemoryTypeUser,
			types.MemoryTypeSession,
			types.MemoryTypeConversation,
			types.MemoryTypeOrg,
		}
	}

	s.mu.Lock()
	s.projects[proj.ID] = proj
	s.mu.Unlock()

	return proj, nil
}

func (s *Service) GetProject(id string) (*types.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if proj, ok := s.projects[id]; ok {
		return proj, nil
	}
	return nil, fmt.Errorf("project not found: %s", id)
}

func (s *Service) GetProjectByUser(userID string) (*types.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, proj := range s.projects {
		if proj.UserID == userID {
			return proj, nil
		}
	}
	return nil, fmt.Errorf("project not found for user: %s", userID)
}

func (s *Service) UpdateProject(ctx context.Context, id string, updates *types.Project) (*types.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	proj, ok := s.projects[id]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", id)
	}

	if updates.Name != "" {
		proj.Name = updates.Name
	}
	if updates.Description != "" {
		proj.Description = updates.Description
	}
	if updates.CustomInstructions != "" {
		proj.CustomInstructions = updates.CustomInstructions
	}
	if updates.Settings.RerankingEnabled {
		proj.Settings.RerankingEnabled = updates.Settings.RerankingEnabled
	}
	if updates.Settings.ConflictResolution {
		proj.Settings.ConflictResolution = updates.Settings.ConflictResolution
	}
	if updates.Metadata != nil {
		proj.Metadata = updates.Metadata
	}

	proj.UpdatedAt = time.Now()
	s.projects[id] = proj

	return proj, nil
}

func (s *Service) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.projects[id]; !ok {
		return fmt.Errorf("project not found: %s", id)
	}

	delete(s.projects, id)
	return nil
}

func (s *Service) ListProjects(userID string, orgID string) []*types.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*types.Project
	for _, proj := range s.projects {
		if userID != "" && proj.UserID != userID {
			continue
		}
		if orgID != "" && proj.OrgID != orgID {
			continue
		}
		result = append(result, proj)
	}
	return result
}

func (s *Service) GetCustomInstructions(projectID string) string {
	proj, err := s.GetProject(projectID)
	if err != nil {
		return ""
	}
	return proj.CustomInstructions
}

func (s *Service) IsRerankingEnabled(projectID string) bool {
	proj, err := s.GetProject(projectID)
	if err != nil {
		return true
	}
	return proj.Settings.RerankingEnabled
}

func (s *Service) IsConflictResolutionEnabled(projectID string) bool {
	proj, err := s.GetProject(projectID)
	if err != nil {
		return false
	}
	return proj.Settings.ConflictResolution
}

func (s *Service) ApplyProjectSettings(req *types.SearchRequest, projectID string) *types.SearchRequest {
	proj, err := s.GetProject(projectID)
	if err != nil {
		return req
	}

	if len(proj.Settings.MemoryTypes) > 0 {
		if req.MemoryType == "" {
			req.MemoryType = proj.Settings.MemoryTypes[0]
		}
	}

	if req.RerankTopK == 0 {
		req.RerankTopK = 20
	}

	if proj.Settings.RerankingEnabled && !req.Rerank {
		req.Rerank = true
	}

	return req
}

func (s *Service) ValidateMemoryCreation(projID string, mem *types.Memory) error {
	proj, err := s.GetProject(projID)
	if err != nil {
		return nil
	}

	if len(proj.Settings.Categories) > 0 {
		if mem.Category != "" {
			valid := false
			for _, cat := range proj.Settings.Categories {
				if cat == mem.Category {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid category %s for project, allowed: %v", mem.Category, proj.Settings.Categories)
			}
		}
	}

	if len(proj.Settings.MemoryTypes) > 0 {
		valid := false
		for _, mt := range proj.Settings.MemoryTypes {
			if string(mt) == string(mem.Type) || mem.Type == "" {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid memory type %s for project", mem.Type)
		}
	}

	if proj.Settings.MaxMemoriesPerUser > 0 && mem.UserID != "" {
		return nil
	}

	return nil
}
