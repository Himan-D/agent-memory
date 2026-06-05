package tools

import (
	"context"
	"fmt"
)

// ToolDefinition describes a self-directed memory management tool that an agent
// can invoke via MCP/function-calling to manage its own memory lifecycle.
// This implements the MemGPT innovation: the agent controls its own memory.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Interfaces to avoid import cycles — the memory package is the cycle root.

// SavedMemoryStore provides access to the saved/explicit memory layer.
type SavedMemoryStore interface {
	Save(userID string, fact interface{}) error
	Search(userID, query string) interface{}
	Delete(userID, memoryID string) error
}

// CoreMemoryStore provides access to the always-loaded core memory.
type CoreMemoryStore interface {
	Get(section string) string
	Set(section, content string)
}

// MarkdownMemoryStore provides access to the filesystem-based markdown memory.
type MarkdownMemoryStore interface {
	WriteFact(userID, topic, fact string) error
	LoadTopic(userID, topic string) (string, error)
}

// ToolHandler dispatches agent memory tool calls to the appropriate backend.
type ToolHandler struct {
	savedStore  SavedMemoryStore
	coreMemory  CoreMemoryStore
	searchFn    func(ctx context.Context, query, userID string, limit int) (interface{}, error)
	markdownMem MarkdownMemoryStore
}

// NewToolHandler creates a ToolHandler. All parameters are optional (may be nil),
// but the corresponding tools will return errors if invoked without a backend.
func NewToolHandler(
	saved SavedMemoryStore,
	core CoreMemoryStore,
	search func(ctx context.Context, query, userID string, limit int) (interface{}, error),
	md MarkdownMemoryStore,
) *ToolHandler {
	return &ToolHandler{
		savedStore:  saved,
		coreMemory:  core,
		searchFn:    search,
		markdownMem: md,
	}
}

// GetToolDefinitions returns all available memory tools for MCP/function-calling registration.
func (th *ToolHandler) GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "memory_save_fact",
			Description: "Save an explicit fact about the user to long-term memory. Use when the user shares a preference, goal, constraint, or important personal detail.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"fact":     map[string]string{"type": "string", "description": "The fact to save"},
					"category": map[string]string{"type": "string", "description": "Category: preference, fact, decision, goal, skill, constraint"},
				},
				"required": []string{"fact"},
			},
		},
		{
			Name:        "memory_search",
			Description: "Search long-term memory for facts relevant to the current conversation.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]string{"type": "string", "description": "Search query"},
					"limit": map[string]string{"type": "integer", "description": "Max results (default 5)"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "memory_delete",
			Description: "Delete a specific saved memory by ID. Use when the user asks to forget something.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memory_id": map[string]string{"type": "string", "description": "ID of the memory to delete"},
				},
				"required": []string{"memory_id"},
			},
		},
		{
			Name:        "core_memory_get",
			Description: "Read a section of core memory (always-loaded persistent context about the user).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"section": map[string]string{"type": "string", "description": "Section name (e.g., user_bio, agent_persona, preferences)"},
				},
				"required": []string{"section"},
			},
		},
		{
			Name:        "core_memory_update",
			Description: "Update a section of core memory. Use to maintain an accurate, up-to-date model of the user.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"section": map[string]string{"type": "string", "description": "Section name"},
					"content": map[string]string{"type": "string", "description": "New content for the section"},
				},
				"required": []string{"section", "content"},
			},
		},
		{
			Name:        "memory_write_note",
			Description: "Write a note to a topic-based markdown memory file. Use for organizing observations and patterns.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"topic": map[string]string{"type": "string", "description": "Topic category (e.g., preferences, decisions, patterns)"},
					"fact":  map[string]string{"type": "string", "description": "The note to write"},
				},
				"required": []string{"topic", "fact"},
			},
		},
		{
			Name:        "memory_read_topic",
			Description: "Read all notes from a specific topic file in markdown memory.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"topic": map[string]string{"type": "string", "description": "Topic to read"},
				},
				"required": []string{"topic"},
			},
		},
	}
}

// ExecuteTool dispatches a tool call to the appropriate handler based on tool name.
func (th *ToolHandler) ExecuteTool(ctx context.Context, userID, toolName string, args map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "memory_save_fact":
		return th.executeSaveFact(userID, args)
	case "memory_search":
		return th.executeSearch(ctx, userID, args)
	case "memory_delete":
		return th.executeDelete(userID, args)
	case "core_memory_get":
		return th.executeCoreGet(args)
	case "core_memory_update":
		return th.executeCoreUpdate(args)
	case "memory_write_note":
		return th.executeWriteNote(userID, args)
	case "memory_read_topic":
		return th.executeReadTopic(userID, args)
	default:
		return nil, fmt.Errorf("tools: unknown tool: %s", toolName)
	}
}

func (th *ToolHandler) executeSaveFact(userID string, args map[string]interface{}) (interface{}, error) {
	if th.savedStore == nil {
		return nil, fmt.Errorf("tools: save fact: saved memory store not configured")
	}
	fact, _ := args["fact"].(string)
	if fact == "" {
		return nil, fmt.Errorf("tools: save fact: 'fact' parameter required")
	}
	category, _ := args["category"].(string)

	factObj := map[string]interface{}{
		"fact":     fact,
		"category": category,
	}
	if err := th.savedStore.Save(userID, factObj); err != nil {
		return nil, fmt.Errorf("tools: save fact: %w", err)
	}
	return map[string]string{"status": "saved", "fact": fact}, nil
}

func (th *ToolHandler) executeSearch(ctx context.Context, userID string, args map[string]interface{}) (interface{}, error) {
	if th.searchFn == nil {
		return nil, fmt.Errorf("tools: search: search function not configured")
	}
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("tools: search: 'query' parameter required")
	}
	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}
	results, err := th.searchFn(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("tools: search: %w", err)
	}
	return results, nil
}

func (th *ToolHandler) executeDelete(userID string, args map[string]interface{}) (interface{}, error) {
	if th.savedStore == nil {
		return nil, fmt.Errorf("tools: delete: saved memory store not configured")
	}
	memoryID, _ := args["memory_id"].(string)
	if memoryID == "" {
		return nil, fmt.Errorf("tools: delete: 'memory_id' parameter required")
	}
	if err := th.savedStore.Delete(userID, memoryID); err != nil {
		return nil, fmt.Errorf("tools: delete: %w", err)
	}
	return map[string]string{"status": "deleted", "memory_id": memoryID}, nil
}

func (th *ToolHandler) executeCoreGet(args map[string]interface{}) (interface{}, error) {
	if th.coreMemory == nil {
		return nil, fmt.Errorf("tools: core get: core memory not configured")
	}
	section, _ := args["section"].(string)
	if section == "" {
		return nil, fmt.Errorf("tools: core get: 'section' parameter required")
	}
	content := th.coreMemory.Get(section)
	return map[string]string{"section": section, "content": content}, nil
}

func (th *ToolHandler) executeCoreUpdate(args map[string]interface{}) (interface{}, error) {
	if th.coreMemory == nil {
		return nil, fmt.Errorf("tools: core update: core memory not configured")
	}
	section, _ := args["section"].(string)
	content, _ := args["content"].(string)
	if section == "" {
		return nil, fmt.Errorf("tools: core update: 'section' parameter required")
	}
	th.coreMemory.Set(section, content)
	return map[string]string{"status": "updated", "section": section}, nil
}

func (th *ToolHandler) executeWriteNote(userID string, args map[string]interface{}) (interface{}, error) {
	if th.markdownMem == nil {
		return nil, fmt.Errorf("tools: write note: markdown memory not configured")
	}
	topic, _ := args["topic"].(string)
	fact, _ := args["fact"].(string)
	if topic == "" || fact == "" {
		return nil, fmt.Errorf("tools: write note: 'topic' and 'fact' parameters required")
	}
	if err := th.markdownMem.WriteFact(userID, topic, fact); err != nil {
		return nil, fmt.Errorf("tools: write note: %w", err)
	}
	return map[string]string{"status": "written", "topic": topic}, nil
}

func (th *ToolHandler) executeReadTopic(userID string, args map[string]interface{}) (interface{}, error) {
	if th.markdownMem == nil {
		return nil, fmt.Errorf("tools: read topic: markdown memory not configured")
	}
	topic, _ := args["topic"].(string)
	if topic == "" {
		return nil, fmt.Errorf("tools: read topic: 'topic' parameter required")
	}
	content, err := th.markdownMem.LoadTopic(userID, topic)
	if err != nil {
		return nil, fmt.Errorf("tools: read topic: %w", err)
	}
	return map[string]string{"topic": topic, "content": content}, nil
}
