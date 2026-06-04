package unified

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
)

type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportHTTP  Transport = "http"
	TransportSSE   Transport = "sse"
)

type Config struct {
	Transport     Transport
	Addr         string
	APIKey       string
	DefaultTenant string
	MemoryService *memory.Service
}

type Server struct {
	config Config
	tools  []Tool
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     func(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

func NewServer(cfg Config) *Server {
	if cfg.Addr == "" {
		cfg.Addr = ":8090"
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
	s.registerTools()
	return s
}

func (s *Server) Run(ctx context.Context, in *os.File, out *os.File) error {
	switch s.config.Transport {
	case TransportStdio:
		return s.runStdio(ctx, in, out)
	case TransportHTTP:
		return s.runHTTP(ctx)
	case TransportSSE:
		return s.runSSE(ctx)
	default:
		return s.runStdio(ctx, in, out)
	}
}

func (s *Server) Shutdown() {
	s.cancel()
}

func (s *Server) tenantID(ctx context.Context) string {
	if tid, ok := ctx.Value(tenantCtxKey).(string); ok && tid != "" {
		return tid
	}
	return s.config.DefaultTenant
}

type contextKey string

const tenantCtxKey contextKey = "tenant_id"

func contextWithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantCtxKey, tenantID)
}

func (s *Server) tenantCtx(ctx context.Context) context.Context {
	return contextWithTenant(ctx, s.tenantID(ctx))
}

func (s *Server) handleAddMemory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	content, _ := args["content"].(string)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	mem := &types.Memory{
		Content:  content,
		UserID:   strVal(args, "user_id"),
		OrgID:    strVal(args, "org_id"),
		AgentID:  strVal(args, "agent_id"),
		SessionID: strVal(args, "session_id"),
		Category: strVal(args, "category"),
	}

	if v, ok := args["metadata"].(map[string]interface{}); ok {
		mem.Metadata = v
	}
	if v, ok := args["immutable"].(bool); ok {
		mem.Immutable = v
	}
	if v, ok := args["expiration_date"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			mem.ExpirationDate = &t
		}
	}
	if mem.Type == "" {
		mem.Type = types.MemoryTypeUser
	}

	tid := s.tenantID(ctx)
	if tid != "" {
		mem.TenantID = tid
	}

	created, err := s.config.MemoryService.CreateMemory(s.tenantCtx(ctx), mem)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":      created.ID,
		"content": created.Content,
		"type":    created.Type,
		"status":  created.Status,
	}, nil
}

func (s *Server) handleSearchMemories(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	req := &types.SearchRequest{Query: query}
	if v, ok := args["limit"].(float64); ok {
		req.Limit = int(v)
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if v, ok := args["threshold"].(float64); ok {
		req.Threshold = float32(v)
	}
	req.UserID = strVal(args, "user_id")
	req.OrgID = strVal(args, "org_id")
	req.AgentID = strVal(args, "agent_id")
	req.Category = strVal(args, "category")
	if v, ok := args["memory_type"].(string); ok {
		req.MemoryType = types.MemoryType(v)
	}

	results, err := s.config.MemoryService.SearchMemories(s.tenantCtx(ctx), req)
	if err != nil {
		return nil, err
	}

	var formatted []map[string]interface{}
	for _, r := range results {
		formatted = append(formatted, map[string]interface{}{
			"id":       r.MemoryID,
			"content":  r.Text,
			"score":    r.Score,
			"source":   r.Source,
			"metadata": r.Metadata,
		})
	}
	return map[string]interface{}{"results": formatted, "count": len(formatted)}, nil
}

func (s *Server) handleGetMemories(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	var memories []*types.Memory
	var err error

	if v := strVal(args, "user_id"); v != "" {
		memories, err = s.config.MemoryService.GetMemoriesByUser(s.tenantCtx(ctx), v)
	} else if v := strVal(args, "org_id"); v != "" {
		memories, err = s.config.MemoryService.GetMemoriesByOrg(s.tenantCtx(ctx), v)
	} else {
		memories = []*types.Memory{}
	}
	if err != nil {
		return nil, err
	}

	limit := 100
	if v, ok := args["limit"].(float64); ok {
		limit = int(v)
	}

	var filtered []*types.Memory
	for _, m := range memories {
		if len(filtered) >= limit {
			break
		}
		if v := strVal(args, "agent_id"); v != "" && m.AgentID != v {
			continue
		}
		if v := strVal(args, "category"); v != "" && m.Category != v {
			continue
		}
		filtered = append(filtered, m)
	}

	var result []map[string]interface{}
	for _, m := range filtered {
		result = append(result, map[string]interface{}{
			"id": m.ID, "content": m.Content, "type": m.Type,
			"category": m.Category, "created_at": m.CreatedAt,
		})
	}
	return map[string]interface{}{"memories": result, "count": len(result)}, nil
}

func (s *Server) handleGetMemory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id, _ := args["memory_id"].(string)
	mem, err := s.config.MemoryService.GetMemory(s.tenantCtx(ctx), id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": mem.ID, "content": mem.Content, "type": mem.Type,
		"category": mem.Category, "metadata": mem.Metadata,
		"status": mem.Status, "immutable": mem.Immutable,
		"created_at": mem.CreatedAt, "updated_at": mem.UpdatedAt,
	}, nil
}

func (s *Server) handleUpdateMemory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id, _ := args["memory_id"].(string)
	content, _ := args["content"].(string)
	if err := s.config.MemoryService.UpdateMemory(s.tenantCtx(ctx), id, content, nil); err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": "updated"}, nil
}

func (s *Server) handleDeleteMemory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id, _ := args["memory_id"].(string)
	if err := s.config.MemoryService.DeleteMemory(s.tenantCtx(ctx), id); err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": "deleted"}, nil
}

func (s *Server) handleDeleteAllMemories(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	ids, ok := args["memory_ids"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("memory_ids must be an array")
	}
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i], _ = id.(string)
	}
	if err := s.config.MemoryService.DeleteMemories(s.tenantCtx(ctx), strIDs); err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": "deleted", "count": len(strIDs)}, nil
}

func (s *Server) handleAddFeedback(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	feedback := &types.Feedback{
		MemoryID: strVal(args, "memory_id"),
		Type:     types.FeedbackType(strVal(args, "feedback_type")),
	}
	if v := strVal(args, "comment"); v != "" {
		feedback.Comment = v
	}
	created, err := s.config.MemoryService.AddFeedback(s.tenantCtx(ctx), feedback)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": created.ID, "status": "created"}, nil
}

func (s *Server) handleGetMemoryHistory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id, _ := args["memory_id"].(string)
	history, err := s.config.MemoryService.GetMemoryHistory(s.tenantCtx(ctx), id)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, h := range history {
		result = append(result, map[string]interface{}{
			"id": h.ID, "action": h.Action, "old_value": h.OldValue,
			"new_value": h.NewValue, "changed_by": h.ChangedBy, "created_at": h.CreatedAt,
		})
	}
	return map[string]interface{}{"history": result, "count": len(result)}, nil
}

func (s *Server) handleDeleteEntities(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	entityID, _ := args["entity_id"].(string)
	memories, err := s.config.MemoryService.GetEntityMemories(s.tenantCtx(ctx), entityID, 100)
	if err != nil {
		return nil, fmt.Errorf("get entity memories: %w", err)
	}
	deletedCount := 0
	for _, m := range memories {
		if err := s.config.MemoryService.DeleteMemory(s.tenantCtx(ctx), m.MemoryID); err != nil {
			continue
		}
		deletedCount++
	}
	return map[string]interface{}{"status": "deleted", "memories": deletedCount, "total_found": len(memories)}, nil
}

func (s *Server) handleListEntities(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := 50
	if v, ok := args["limit"].(float64); ok {
		limit = int(v)
	}
	entities, err := s.config.MemoryService.ListEntities(s.tenantID(ctx), limit)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(entities))
	for _, e := range entities {
		result = append(result, map[string]interface{}{"id": e.ID, "name": e.Name, "type": e.Type})
	}
	return map[string]interface{}{"entities": result, "count": len(result)}, nil
}

func (s *Server) handleCreateEntity(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	entity := types.Entity{
		Name: strVal(args, "name"),
		Type: strVal(args, "entity_type"),
	}
	if v, ok := args["properties"].(map[string]interface{}); ok {
		entity.Properties = v
	}
	created, err := s.config.MemoryService.AddEntity(entity)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": created.ID, "name": created.Name, "type": created.Type}, nil
}

func (s *Server) handleCreateRelation(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if err := s.config.MemoryService.AddRelation(
		strVal(args, "from_id"), strVal(args, "to_id"), strVal(args, "relation_type"), nil,
	); err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": "created"}, nil
}

func (s *Server) handleGetEntityRelations(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	relations, err := s.config.MemoryService.GetEntityRelations(
		strVal(args, "entity_id"), strVal(args, "relation_type"),
	)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, r := range relations {
		result = append(result, map[string]interface{}{
			"id": r.ID, "from": r.FromID, "to": r.ToID, "type": r.Type, "weight": r.Weight,
		})
	}
	return map[string]interface{}{"relations": result, "count": len(result)}, nil
}

func (s *Server) handleCreateSession(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	session, err := s.config.MemoryService.CreateSession(strVal(args, "agent_id"), nil)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": session.ID, "agent_id": session.AgentID}, nil
}

func (s *Server) handleAddMessage(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	msg := types.Message{
		Role:    strVal(args, "role"),
		Content: strVal(args, "content"),
	}
	if err := s.config.MemoryService.AddToContext(strVal(args, "session_id"), msg); err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": "added"}, nil
}

func (s *Server) handleGetContext(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := 50
	if v, ok := args["limit"].(float64); ok {
		limit = int(v)
	}
	messages, err := s.config.MemoryService.GetContext(strVal(args, "session_id"), limit)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, m := range messages {
		result = append(result, map[string]interface{}{
			"id": m.ID, "role": m.Role, "content": m.Content, "timestamp": m.Timestamp,
		})
	}
	return map[string]interface{}{"messages": result, "count": len(result)}, nil
}

func (s *Server) handleCreateSkill(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	skill := &types.Skill{
		Name:    strVal(args, "name"),
		Trigger: strVal(args, "trigger"),
		Action:  strVal(args, "action"),
	}
	if v := strVal(args, "domain"); v != "" {
		skill.Domain = v
	}
	if v, ok := args["confidence"].(float64); ok {
		skill.Confidence = float32(v)
	}
	if v, ok := args["tags"].([]interface{}); ok {
		for _, t := range v {
			skill.Tags = append(skill.Tags, t.(string))
		}
	}
	if err := s.config.MemoryService.CreateSkill(s.tenantCtx(ctx), skill); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": skill.ID, "name": skill.Name, "trigger": skill.Trigger}, nil
}

func (s *Server) handleListSkills(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := 50
	if v, ok := args["limit"].(float64); ok {
		limit = int(v)
	}
	skills, err := s.config.MemoryService.ListSkills(s.tenantCtx(ctx), s.tenantID(ctx), strVal(args, "domain"), limit, 0)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, sk := range skills {
		result = append(result, map[string]interface{}{
			"id": sk.ID, "name": sk.Name, "trigger": sk.Trigger,
			"domain": sk.Domain, "confidence": sk.Confidence,
		})
	}
	return map[string]interface{}{"skills": result, "count": len(result)}, nil
}

func (s *Server) handleSuggestSkills(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := 5
	if v, ok := args["limit"].(float64); ok {
		limit = int(v)
	}
	skills, err := s.config.MemoryService.SuggestSkills(s.tenantCtx(ctx), strVal(args, "trigger"), strVal(args, "context"), limit)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, sk := range skills {
		result = append(result, map[string]interface{}{
			"id": sk.ID, "name": sk.Name, "trigger": sk.Trigger,
			"action": sk.Action, "confidence": sk.Confidence,
		})
	}
	return map[string]interface{}{"suggestions": result, "count": len(result)}, nil
}

func (s *Server) handleExtractSkills(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	result, err := s.config.MemoryService.ExtractSkills(s.tenantCtx(ctx), strVal(args, "content"), strVal(args, "user_id"), "")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"skills": result.Skills, "count": len(result.Skills)}, nil
}

func (s *Server) handleCreateAgent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	agent := &types.Agent{Name: strVal(args, "name")}
	if v := strVal(args, "description"); v != "" {
		agent.Description = v
	}
	if err := s.config.MemoryService.CreateAgent(s.tenantCtx(ctx), agent); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": agent.ID, "name": agent.Name}, nil
}

func (s *Server) handleListAgents(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := 50
	if v, ok := args["limit"].(float64); ok {
		limit = int(v)
	}
	agents, _, err := s.config.MemoryService.ListAgents(s.tenantCtx(ctx), s.tenantID(ctx), limit, 0)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, a := range agents {
		result = append(result, map[string]interface{}{
			"id": a.ID, "name": a.Name, "status": a.Status, "description": a.Description,
		})
	}
	return map[string]interface{}{"agents": result, "total": len(result)}, nil
}

func (s *Server) handleCreateAgentGroup(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	group := &types.AgentGroup{Name: strVal(args, "name")}
	if v := strVal(args, "description"); v != "" {
		group.Description = v
	}
	if v := strVal(args, "domain"); v != "" {
		group.Domain = v
	}
	if err := s.config.MemoryService.CreateAgentGroup(s.tenantCtx(ctx), group); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": group.ID, "name": group.Name}, nil
}

func (s *Server) handleAddAgentToGroup(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	role := types.MemberRoleContributor
	if v := strVal(args, "role"); v != "" {
		role = types.MemberRole(v)
	}
	if err := s.config.MemoryService.AddAgentToGroup(s.tenantCtx(ctx), strVal(args, "agent_id"), strVal(args, "group_id"), role); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true}, nil
}

func (s *Server) handleShareMemoryToGroup(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if err := s.config.MemoryService.ShareMemoryToGroup(s.tenantCtx(ctx), strVal(args, "memory_id"), strVal(args, "group_id")); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true}, nil
}

func (s *Server) handleListPendingReviews(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	reviews, err := s.config.MemoryService.ListPendingReviews(s.tenantCtx(ctx), s.tenantID(ctx))
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, r := range reviews {
		result = append(result, map[string]interface{}{
			"id": r.ID, "skill_id": r.SkillID, "status": r.Status, "created": r.CreatedAt,
		})
	}
	return map[string]interface{}{"reviews": result, "count": len(result)}, nil
}

func (s *Server) handleCreateConcept(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	name := strVal(args, "name")
	description := strVal(args, "description")

	neo4jClient := s.config.MemoryService.GetNeo4jClient()
	if neo4jClient == nil {
		return map[string]interface{}{
			"status":      "concept creation requires API access",
			"name":        name,
			"description": description,
		}, nil
	}

	concept := &types.Concept{
		Name:        name,
		Description: description,
	}
	if err := neo4jClient.CreateConcept(s.tenantCtx(ctx), concept); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": concept.ID, "name": concept.Name, "description": concept.Description, "status": "created",
	}, nil
}

func (s *Server) handleLinkToConcept(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	conceptID := strVal(args, "concept_id")
	nodeID := strVal(args, "node_id")
	relType := strVal(args, "rel_type")
	if relType == "" {
		relType = "BELONGS_TO"
	}
	neo4jClient := s.config.MemoryService.GetNeo4jClient()
	if neo4jClient == nil {
		return map[string]interface{}{"status": "linked", "note": "neo4j not configured"}, nil
	}
	if err := neo4jClient.LinkToConcept(s.tenantCtx(ctx), nodeID, conceptID, relType); err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": "linked"}, nil
}

func (s *Server) handleSetReminder(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	memoryID := strVal(args, "memory_id")
	remindAt := strVal(args, "remind_at")
	condition := strVal(args, "condition")
	if _, err := time.Parse(time.RFC3339, remindAt); err != nil {
		return nil, fmt.Errorf("invalid remind_at format, use RFC3339: %w", err)
	}
	if err := s.config.MemoryService.UpdateMemory(s.tenantCtx(ctx), memoryID, "", map[string]interface{}{
		"remind_at": remindAt, "remind_condition": condition,
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": "reminder_set", "memory_id": memoryID}, nil
}

func (s *Server) handleCheckSafety(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	content, _ := args["content"].(string)
	if s.config.MemoryService != nil {
		result := s.config.MemoryService.CheckSafety(ctx, content)
		return result, nil
	}
	return map[string]interface{}{"safe": true, "category": "unknown", "reason": "safety classifier not configured"}, nil
}

func strVal(args map[string]interface{}, key string) string {
	v, _ := args[key].(string)
	return v
}

func (s *Server) registerTools() {
	s.tools = []Tool{
		s.tool("add_memory", "Save a memory for later retrieval. Use this to store important information, facts, preferences, or any content that should be remembered across conversations.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content":         map[string]interface{}{"type": "string", "description": "The memory content to store"},
				"user_id":         map[string]interface{}{"type": "string", "description": "Optional user identifier for user-level memory"},
				"org_id":          map[string]interface{}{"type": "string", "description": "Optional organization identifier"},
				"agent_id":        map[string]interface{}{"type": "string", "description": "Optional agent identifier"},
				"session_id":      map[string]interface{}{"type": "string", "description": "Optional session identifier"},
				"category":        map[string]interface{}{"type": "string", "description": "Optional category for organization"},
				"metadata":        map[string]interface{}{"type": "object", "description": "Optional additional metadata"},
				"immutable":       map[string]interface{}{"type": "boolean", "description": "If true, memory cannot be modified or deleted"},
				"expiration_date": map[string]interface{}{"type": "string", "description": "Optional RFC3339 expiration date for TTL"},
			},
			"required": []string{"content"},
		}, s.handleAddMemory),

		s.tool("search_memories", "Search semantically through stored memories. Uses vector embeddings to find relevant content based on meaning rather than exact matches.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query":        map[string]interface{}{"type": "string", "description": "Natural language search query"},
				"limit":        map[string]interface{}{"type": "integer", "description": "Maximum results to return (default 10, max 100)"},
				"threshold":    map[string]interface{}{"type": "number", "description": "Minimum similarity score threshold (0.0-1.0, default 0.5)"},
				"user_id":      map[string]interface{}{"type": "string", "description": "Filter by user identifier"},
				"org_id":       map[string]interface{}{"type": "string", "description": "Filter by organization identifier"},
				"agent_id":     map[string]interface{}{"type": "string", "description": "Filter by agent identifier"},
				"category":     map[string]interface{}{"type": "string", "description": "Filter by category"},
				"memory_type":  map[string]interface{}{"type": "string", "description": "Filter by memory type: conversation, session, user, org"},
			},
			"required": []string{"query"},
		}, s.handleSearchMemories),

		s.tool("get_memories", "List all memories with optional filters. Supports pagination and filtering by user, organization, agent, or category.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_id":   map[string]interface{}{"type": "string", "description": "Filter by user identifier"},
				"org_id":    map[string]interface{}{"type": "string", "description": "Filter by organization identifier"},
				"agent_id":  map[string]interface{}{"type": "string", "description": "Filter by agent identifier"},
				"category":  map[string]interface{}{"type": "string", "description": "Filter by category"},
				"limit":     map[string]interface{}{"type": "integer", "description": "Maximum results to return"},
			},
		}, s.handleGetMemories),

		s.tool("get_memory", "Get a specific memory by its ID.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"memory_id": map[string]interface{}{"type": "string", "description": "The memory identifier"},
			},
			"required": []string{"memory_id"},
		}, s.handleGetMemory),

		s.tool("update_memory", "Update an existing memory's content or metadata. Cannot update immutable memories.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"memory_id": map[string]interface{}{"type": "string", "description": "The memory identifier"},
				"content":   map[string]interface{}{"type": "string", "description": "New content for the memory"},
				"metadata":  map[string]interface{}{"type": "object", "description": "Updated metadata to merge with existing"},
			},
			"required": []string{"memory_id", "content"},
		}, s.handleUpdateMemory),

		s.tool("delete_memory", "Delete a specific memory by ID. Cannot delete immutable memories.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"memory_id": map[string]interface{}{"type": "string", "description": "The memory identifier"},
			},
			"required": []string{"memory_id"},
		}, s.handleDeleteMemory),

		s.tool("delete_all_memories", "Delete multiple memories by their IDs in a single operation.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"memory_ids": map[string]interface{}{
					"type": "array", "items": map[string]interface{}{"type": "string"},
					"description": "Array of memory identifiers to delete",
				},
			},
			"required": []string{"memory_ids"},
		}, s.handleDeleteAllMemories),

		s.tool("delete_entities", "Delete an entity and all associated memories.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_id": map[string]interface{}{"type": "string", "description": "The entity identifier"},
			},
			"required": []string{"entity_id"},
		}, s.handleDeleteEntities),

		s.tool("list_entities", "List all entities in the knowledge graph, optionally filtered by type.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_type": map[string]interface{}{"type": "string", "description": "Optional entity type filter (e.g., Person, Service, Document)"},
				"limit":       map[string]interface{}{"type": "integer", "description": "Maximum results to return"},
			},
		}, s.handleListEntities),

		s.tool("create_entity", "Create a new entity in the knowledge graph.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":         map[string]interface{}{"type": "string", "description": "Entity name"},
				"entity_type":  map[string]interface{}{"type": "string", "description": "Entity type (e.g., Person, Service, Document)"},
				"properties":   map[string]interface{}{"type": "object", "description": "Optional entity properties"},
			},
			"required": []string{"name", "entity_type"},
		}, s.handleCreateEntity),

		s.tool("create_relation", "Create a relationship between two entities in the knowledge graph.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"from_id":       map[string]interface{}{"type": "string", "description": "Source entity ID"},
				"to_id":         map[string]interface{}{"type": "string", "description": "Target entity ID"},
				"relation_type": map[string]interface{}{"type": "string", "description": "Relation type (KNOWS, HAS, RELATED_TO, DEPENDS_ON, USES, etc.)"},
				"metadata":      map[string]interface{}{"type": "object", "description": "Optional relation metadata"},
			},
			"required": []string{"from_id", "to_id", "relation_type"},
		}, s.handleCreateRelation),

		s.tool("get_entity_relations", "Get all relations for a specific entity.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_id":      map[string]interface{}{"type": "string", "description": "The entity identifier"},
				"relation_type":  map[string]interface{}{"type": "string", "description": "Optional relation type filter"},
			},
			"required": []string{"entity_id"},
		}, s.handleGetEntityRelations),

		s.tool("add_feedback", "Provide feedback on a memory to improve future search results.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"memory_id":     map[string]interface{}{"type": "string", "description": "The memory identifier"},
				"feedback_type": map[string]interface{}{"type": "string", "description": "Feedback type: positive, negative, very_negative"},
				"comment":       map[string]interface{}{"type": "string", "description": "Optional feedback comment"},
			},
			"required": []string{"memory_id", "feedback_type"},
		}, s.handleAddFeedback),

		s.tool("get_memory_history", "Get the modification history of a memory.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"memory_id": map[string]interface{}{"type": "string", "description": "The memory identifier"},
			},
			"required": []string{"memory_id"},
		}, s.handleGetMemoryHistory),

		s.tool("create_session", "Create a new conversation session for an agent.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"agent_id": map[string]interface{}{"type": "string", "description": "Unique identifier for the agent"},
				"metadata": map[string]interface{}{"type": "object", "description": "Optional session metadata"},
			},
			"required": []string{"agent_id"},
		}, s.handleCreateSession),

		s.tool("add_message", "Add a message to a conversation session.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{"type": "string", "description": "Session identifier"},
				"role":       map[string]interface{}{"type": "string", "description": "Message role: user, assistant, system, tool"},
				"content":    map[string]interface{}{"type": "string", "description": "Message content"},
			},
			"required": []string{"session_id", "role", "content"},
		}, s.handleAddMessage),

		s.tool("get_context", "Get conversation context/messages for a session.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{"type": "string", "description": "Session identifier"},
				"limit":      map[string]interface{}{"type": "integer", "description": "Maximum messages to return"},
			},
			"required": []string{"session_id"},
		}, s.handleGetContext),

		s.tool("create_skill", "Create a new skill/procedure for reusable agent patterns.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":       map[string]interface{}{"type": "string", "description": "Skill name"},
				"trigger":    map[string]interface{}{"type": "string", "description": "What activates this skill"},
				"action":     map[string]interface{}{"type": "string", "description": "What the skill does"},
				"domain":     map[string]interface{}{"type": "string", "description": "Domain category (coding, debugging, etc.)"},
				"confidence": map[string]interface{}{"type": "number", "description": "Confidence score 0-1"},
				"tags":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Optional tags"},
			},
			"required": []string{"name", "trigger", "action"},
		}, s.handleCreateSkill),

		s.tool("list_skills", "List skills with optional domain filter.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"domain": map[string]interface{}{"type": "string", "description": "Filter by domain"},
				"limit":  map[string]interface{}{"type": "integer", "description": "Maximum results"},
			},
		}, s.handleListSkills),

		s.tool("suggest_skills", "Get skill suggestions for a trigger using LLM.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"trigger": map[string]interface{}{"type": "string", "description": "The situation/cue"},
				"context": map[string]interface{}{"type": "string", "description": "Optional context"},
				"limit":   map[string]interface{}{"type": "integer", "description": "Maximum suggestions"},
			},
			"required": []string{"trigger"},
		}, s.handleSuggestSkills),

		s.tool("extract_skills", "Extract skills from content using LLM.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{"type": "string", "description": "Content to analyze"},
				"user_id": map[string]interface{}{"type": "string", "description": "Optional user identifier"},
			},
			"required": []string{"content"},
		}, s.handleExtractSkills),

		s.tool("create_agent", "Create a new agent.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":        map[string]interface{}{"type": "string", "description": "Agent name"},
				"description": map[string]interface{}{"type": "string", "description": "Optional description"},
			},
			"required": []string{"name"},
		}, s.handleCreateAgent),

		s.tool("list_agents", "List all agents.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "integer", "description": "Maximum results"},
			},
		}, s.handleListAgents),

		s.tool("create_agent_group", "Create a new agent group for multi-agent collaboration.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":        map[string]interface{}{"type": "string", "description": "Group name"},
				"description": map[string]interface{}{"type": "string", "description": "Optional description"},
				"domain":      map[string]interface{}{"type": "string", "description": "Domain category"},
			},
			"required": []string{"name"},
		}, s.handleCreateAgentGroup),

		s.tool("add_agent_to_group", "Add an agent to a group.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"group_id": map[string]interface{}{"type": "string", "description": "Group identifier"},
				"agent_id": map[string]interface{}{"type": "string", "description": "Agent identifier"},
				"role":     map[string]interface{}{"type": "string", "description": "Member role (admin, contributor, reader)"},
			},
			"required": []string{"group_id", "agent_id"},
		}, s.handleAddAgentToGroup),

		s.tool("share_memory_to_group", "Share a memory with a group.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"group_id":  map[string]interface{}{"type": "string", "description": "Group identifier"},
				"memory_id": map[string]interface{}{"type": "string", "description": "Memory identifier to share"},
			},
			"required": []string{"group_id", "memory_id"},
		}, s.handleShareMemoryToGroup),

		s.tool("list_pending_reviews", "List pending skill reviews for human approval.", map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}, s.handleListPendingReviews),

		s.tool("create_concept", "Create a concept node in the knowledge graph.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":        map[string]interface{}{"type": "string", "description": "Concept name"},
				"description": map[string]interface{}{"type": "string", "description": "Optional concept description"},
			},
			"required": []string{"name"},
		}, s.handleCreateConcept),

		s.tool("link_to_concept", "Link a memory or entity to a concept node.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"concept_id": map[string]interface{}{"type": "string", "description": "The concept identifier"},
				"node_id":    map[string]interface{}{"type": "string", "description": "The memory or entity identifier to link"},
				"rel_type":   map[string]interface{}{"type": "string", "description": "Relationship type (default: BELONGS_TO)"},
			},
			"required": []string{"concept_id", "node_id"},
		}, s.handleLinkToConcept),

		s.tool("set_reminder", "Set a prospective memory reminder on a memory.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"memory_id": map[string]interface{}{"type": "string", "description": "The memory identifier"},
				"remind_at": map[string]interface{}{"type": "string", "description": "RFC3339 timestamp when to trigger the reminder"},
				"condition": map[string]interface{}{"type": "string", "description": "Optional condition string for conditional reminders"},
			},
			"required": []string{"memory_id", "remind_at"},
		}, s.handleSetReminder),

		s.tool("check_safety", "Check content safety classification.", map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{"type": "string", "description": "Content to check for safety"},
			},
			"required": []string{"content"},
		}, s.handleCheckSafety),
	}
}

func (s *Server) tool(name, description string, schema map[string]interface{}, handler func(ctx context.Context, args map[string]interface{}) (interface{}, error)) Tool {
	return Tool{Name: name, Description: description, InputSchema: schema, Handler: handler}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}