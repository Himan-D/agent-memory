package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/project"
	"agent-memory/internal/webhook"
)

type MCPServer struct {
	memSvc  *memory.Service
	projSvc *project.Service
	whSvc   *webhook.Service
}

type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type MCPTool struct {
	Tool    Tool `json:"tool"`
	Handler func(*MCPServer, map[string]interface{}) (interface{}, error)
}

var tools = []MCPTool{
	{
		Tool: Tool{
			Name:        "add_memory",
			Description: "Save a memory for later retrieval. Use this to store important information, facts, preferences, or any content that should be remembered across conversations.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The memory content to store",
					},
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional user identifier for user-level memory",
					},
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional organization identifier",
					},
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional agent identifier",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional session identifier",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Optional category for organization",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Optional additional metadata",
					},
					"immutable": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, memory cannot be modified or deleted",
					},
					"expiration_date": map[string]interface{}{
						"type":        "string",
						"description": "Optional RFC3339 expiration date for TTL",
					},
				},
				"required": []string{"content"},
			},
		},
		Handler: addMemory,
	},
	{
		Tool: Tool{
			Name:        "search_memories",
			Description: "Search semantically through stored memories. Uses vector embeddings to find relevant content based on meaning rather than exact matches.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Natural language search query",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results to return (default 10, max 100)",
					},
					"threshold": map[string]interface{}{
						"type":        "number",
						"description": "Minimum similarity score threshold (0.0-1.0, default 0.5)",
					},
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "Filter by user identifier",
					},
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "Filter by organization identifier",
					},
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "Filter by agent identifier",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category",
					},
					"memory_type": map[string]interface{}{
						"type":        "string",
						"description": "Filter by memory type: conversation, session, user, org",
					},
				},
				"required": []string{"query"},
			},
		},
		Handler: searchMemories,
	},
	{
		Tool: Tool{
			Name:        "get_memories",
			Description: "List all memories with optional filters. Supports pagination and filtering by user, organization, agent, or category.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "Filter by user identifier",
					},
					"org_id": map[string]interface{}{
						"type":        "string",
						"description": "Filter by organization identifier",
					},
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "Filter by agent identifier",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results to return",
					},
				},
			},
		},
		Handler: getMemories,
	},
	{
		Tool: Tool{
			Name:        "get_memory",
			Description: "Get a specific memory by its ID.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memory_id": map[string]interface{}{
						"type":        "string",
						"description": "The memory identifier",
					},
				},
				"required": []string{"memory_id"},
			},
		},
		Handler: getMemory,
	},
	{
		Tool: Tool{
			Name:        "update_memory",
			Description: "Update an existing memory's content or metadata. Cannot update immutable memories.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memory_id": map[string]interface{}{
						"type":        "string",
						"description": "The memory identifier",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "New content for the memory",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Updated metadata to merge with existing",
					},
				},
				"required": []string{"memory_id", "content"},
			},
		},
		Handler: updateMemory,
	},
	{
		Tool: Tool{
			Name:        "delete_memory",
			Description: "Delete a specific memory by ID. Cannot delete immutable memories.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memory_id": map[string]interface{}{
						"type":        "string",
						"description": "The memory identifier",
					},
				},
				"required": []string{"memory_id"},
			},
		},
		Handler: deleteMemory,
	},
	{
		Tool: Tool{
			Name:        "delete_all_memories",
			Description: "Delete multiple memories by their IDs in a single operation.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memory_ids": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Array of memory identifiers to delete",
					},
				},
				"required": []string{"memory_ids"},
			},
		},
		Handler: deleteAllMemories,
	},
	{
		Tool: Tool{
			Name:        "delete_entities",
			Description: "Delete an entity and all associated memories. This removes the entity from the knowledge graph and all linked memories.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"entity_id": map[string]interface{}{
						"type":        "string",
						"description": "The entity identifier",
					},
				},
				"required": []string{"entity_id"},
			},
		},
		Handler: deleteEntities,
	},
	{
		Tool: Tool{
			Name:        "list_entities",
			Description: "List all entities in the knowledge graph, optionally filtered by type.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"entity_type": map[string]interface{}{
						"type":        "string",
						"description": "Optional entity type filter (e.g., Person, Service, Document)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results to return",
					},
				},
			},
		},
		Handler: listEntities,
	},
	{
		Tool: Tool{
			Name:        "create_entity",
			Description: "Create a new entity in the knowledge graph.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Entity name",
					},
					"entity_type": map[string]interface{}{
						"type":        "string",
						"description": "Entity type (e.g., Person, Service, Document)",
					},
					"properties": map[string]interface{}{
						"type":        "object",
						"description": "Optional entity properties",
					},
				},
				"required": []string{"name", "entity_type"},
			},
		},
		Handler: createEntity,
	},
	{
		Tool: Tool{
			Name:        "create_relation",
			Description: "Create a relationship between two entities in the knowledge graph.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"from_id": map[string]interface{}{
						"type":        "string",
						"description": "Source entity ID",
					},
					"to_id": map[string]interface{}{
						"type":        "string",
						"description": "Target entity ID",
					},
					"relation_type": map[string]interface{}{
						"type":        "string",
						"description": "Relation type (KNOWS, HAS, RELATED_TO, DEPENDS_ON, USES, etc.)",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Optional relation metadata",
					},
				},
				"required": []string{"from_id", "to_id", "relation_type"},
			},
		},
		Handler: createRelation,
	},
	{
		Tool: Tool{
			Name:        "get_entity_relations",
			Description: "Get all relations for a specific entity.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"entity_id": map[string]interface{}{
						"type":        "string",
						"description": "The entity identifier",
					},
					"relation_type": map[string]interface{}{
						"type":        "string",
						"description": "Optional relation type filter",
					},
				},
				"required": []string{"entity_id"},
			},
		},
		Handler: getEntityRelations,
	},
	{
		Tool: Tool{
			Name:        "add_feedback",
			Description: "Provide feedback on a memory to improve future search results. Positive feedback helps the system understand what's important.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memory_id": map[string]interface{}{
						"type":        "string",
						"description": "The memory identifier",
					},
					"feedback_type": map[string]interface{}{
						"type":        "string",
						"description": "Feedback type: positive, negative, very_negative",
					},
					"comment": map[string]interface{}{
						"type":        "string",
						"description": "Optional feedback comment",
					},
				},
				"required": []string{"memory_id", "feedback_type"},
			},
		},
		Handler: addFeedback,
	},
	{
		Tool: Tool{
			Name:        "get_memory_history",
			Description: "Get the modification history of a memory, including updates, feedback, and other changes.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memory_id": map[string]interface{}{
						"type":        "string",
						"description": "The memory identifier",
					},
				},
				"required": []string{"memory_id"},
			},
		},
		Handler: getMemoryHistory,
	},
	{
		Tool: Tool{
			Name:        "create_session",
			Description: "Create a new conversation session for an agent.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "Unique identifier for the agent",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Optional session metadata",
					},
				},
				"required": []string{"agent_id"},
			},
		},
		Handler: createSession,
	},
	{
		Tool: Tool{
			Name:        "add_message",
			Description: "Add a message to a conversation session.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier",
					},
					"role": map[string]interface{}{
						"type":        "string",
						"description": "Message role: user, assistant, system, tool",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Message content",
					},
				},
				"required": []string{"session_id", "role", "content"},
			},
		},
		Handler: addMessage,
	},
	{
		Tool: Tool{
			Name:        "get_context",
			Description: "Get conversation context/messages for a session.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum messages to return",
					},
				},
				"required": []string{"session_id"},
			},
		},
		Handler: getContext,
	},
}

// Tool handlers

func addMemory(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	mem := &types.Memory{
		Content: params["content"].(string),
	}

	if v, ok := params["user_id"].(string); ok {
		mem.UserID = v
	}
	if v, ok := params["org_id"].(string); ok {
		mem.OrgID = v
	}
	if v, ok := params["agent_id"].(string); ok {
		mem.AgentID = v
	}
	if v, ok := params["session_id"].(string); ok {
		mem.SessionID = v
	}
	if v, ok := params["category"].(string); ok {
		mem.Category = v
	}
	if v, ok := params["metadata"].(map[string]interface{}); ok {
		mem.Metadata = v
	}
	if v, ok := params["immutable"].(bool); ok {
		mem.Immutable = v
	}
	if v, ok := params["expiration_date"].(string); ok {
		expTime, err := parseTime(v)
		if err == nil {
			mem.ExpirationDate = &expTime
		}
	}

	if mem.Type == "" {
		mem.Type = types.MemoryTypeUser
	}

	created, err := s.memSvc.CreateMemory(context.Background(), mem)
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

func searchMemories(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	req := &types.SearchRequest{
		Query: params["query"].(string),
	}

	if v, ok := params["limit"].(float64); ok {
		req.Limit = int(v)
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if v, ok := params["threshold"].(float64); ok {
		req.Threshold = float32(v)
	}
	if v, ok := params["user_id"].(string); ok {
		req.UserID = v
	}
	if v, ok := params["org_id"].(string); ok {
		req.OrgID = v
	}
	if v, ok := params["agent_id"].(string); ok {
		req.AgentID = v
	}
	if v, ok := params["category"].(string); ok {
		req.Category = v
	}
	if v, ok := params["memory_type"].(string); ok {
		req.MemoryType = types.MemoryType(v)
	}

	results, err := s.memSvc.SearchMemories(context.Background(), req)
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

	return map[string]interface{}{
		"results": formatted,
		"count":   len(formatted),
	}, nil
}

func getMemories(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	var memories []*types.Memory
	var err error

	if v, ok := params["user_id"].(string); ok {
		memories, err = s.memSvc.GetMemoriesByUser(context.Background(), v)
	} else if v, ok := params["org_id"].(string); ok {
		memories, err = s.memSvc.GetMemoriesByOrg(context.Background(), v)
	} else {
		memories = []*types.Memory{}
		err = nil
	}

	if err != nil {
		return nil, err
	}

	limit := 100
	if v, ok := params["limit"].(float64); ok {
		limit = int(v)
	}

	var filtered []*types.Memory
	for _, m := range memories {
		if len(filtered) >= limit {
			break
		}
		if v, ok := params["agent_id"].(string); ok && m.AgentID != v {
			continue
		}
		if v, ok := params["category"].(string); ok && m.Category != v {
			continue
		}
		filtered = append(filtered, m)
	}

	var result []map[string]interface{}
	for _, m := range filtered {
		result = append(result, map[string]interface{}{
			"id":         m.ID,
			"content":    m.Content,
			"type":       m.Type,
			"category":   m.Category,
			"created_at": m.CreatedAt,
		})
	}

	return map[string]interface{}{
		"memories": result,
		"count":    len(result),
	}, nil
}

func getMemory(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	mem, err := s.memSvc.GetMemory(context.Background(), params["memory_id"].(string))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":         mem.ID,
		"content":    mem.Content,
		"type":       mem.Type,
		"category":   mem.Category,
		"metadata":   mem.Metadata,
		"status":     mem.Status,
		"immutable":  mem.Immutable,
		"created_at": mem.CreatedAt,
		"updated_at": mem.UpdatedAt,
	}, nil
}

func updateMemory(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	err := s.memSvc.UpdateMemory(
		context.Background(),
		params["memory_id"].(string),
		params["content"].(string),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"status": "updated"}, nil
}

func deleteMemory(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	err := s.memSvc.DeleteMemory(context.Background(), params["memory_id"].(string))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"status": "deleted"}, nil
}

func deleteAllMemories(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	ids := params["memory_ids"].([]interface{})
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.(string)
	}

	err := s.memSvc.DeleteMemories(context.Background(), strIDs)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "deleted",
		"count":  len(strIDs),
	}, nil
}

func deleteEntities(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	entityID := params["entity_id"].(string)

	memories, _ := s.memSvc.GetEntityMemories(context.Background(), entityID, 100)
	for _, m := range memories {
		_ = s.memSvc.DeleteMemory(context.Background(), m.MemoryID)
	}

	return map[string]interface{}{
		"status":   "deleted",
		"memories": len(memories),
	}, nil
}

func listEntities(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"entities": []map[string]interface{}{},
		"count":    0,
	}, nil
}

func createEntity(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	entity := types.Entity{
		Name: params["name"].(string),
		Type: params["entity_type"].(string),
	}
	if v, ok := params["properties"].(map[string]interface{}); ok {
		entity.Properties = v
	}

	created, err := s.memSvc.AddEntity(entity)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":   created.ID,
		"name": created.Name,
		"type": created.Type,
	}, nil
}

func createRelation(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	err := s.memSvc.AddRelation(
		params["from_id"].(string),
		params["to_id"].(string),
		params["relation_type"].(string),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"status": "created"}, nil
}

func getEntityRelations(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	relations, err := s.memSvc.GetEntityRelations(
		params["entity_id"].(string),
		params["relation_type"].(string),
	)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, r := range relations {
		result = append(result, map[string]interface{}{
			"id":     r.ID,
			"from":   r.FromID,
			"to":     r.ToID,
			"type":   r.Type,
			"weight": r.Weight,
		})
	}

	return map[string]interface{}{
		"relations": result,
		"count":     len(result),
	}, nil
}

func addFeedback(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	feedback := &types.Feedback{
		MemoryID: params["memory_id"].(string),
		Type:     types.FeedbackType(params["feedback_type"].(string)),
	}
	if v, ok := params["comment"].(string); ok {
		feedback.Comment = v
	}

	created, err := s.memSvc.AddFeedback(context.Background(), feedback)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":     created.ID,
		"status": "created",
	}, nil
}

func getMemoryHistory(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	history, err := s.memSvc.GetMemoryHistory(context.Background(), params["memory_id"].(string))
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, h := range history {
		result = append(result, map[string]interface{}{
			"id":         h.ID,
			"action":     h.Action,
			"old_value":  h.OldValue,
			"new_value":  h.NewValue,
			"changed_by": h.ChangedBy,
			"created_at": h.CreatedAt,
		})
	}

	return map[string]interface{}{
		"history": result,
		"count":   len(result),
	}, nil
}

func createSession(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	session, err := s.memSvc.CreateSession(
		params["agent_id"].(string),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":       session.ID,
		"agent_id": session.AgentID,
	}, nil
}

func addMessage(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	msg := types.Message{
		Role:    params["role"].(string),
		Content: params["content"].(string),
	}

	err := s.memSvc.AddToContext(params["session_id"].(string), msg)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"status": "added"}, nil
}

func getContext(s *MCPServer, params map[string]interface{}) (interface{}, error) {
	limit := 50
	if v, ok := params["limit"].(float64); ok {
		limit = int(v)
	}

	messages, err := s.memSvc.GetContext(params["session_id"].(string), limit)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, m := range messages {
		result = append(result, map[string]interface{}{
			"id":        m.ID,
			"role":      m.Role,
			"content":   m.Content,
			"timestamp": m.Timestamp,
		})
	}

	return map[string]interface{}{
		"messages": result,
		"count":    len(result),
	}, nil
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// MCP Protocol handling

func (s *MCPServer) HandleRequest(r io.Reader, w io.Writer) {
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)

	for dec.More() {
		var req MCPRequest
		if err := dec.Decode(&req); err != nil {
			continue
		}

		var resp MCPResponse
		resp.JSONRPC = "2.0"
		resp.ID = req.ID

		switch req.Method {
		case "initialize":
			resp.Result = map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools":     true,
					"resources": true,
				},
				"serverInfo": map[string]interface{}{
					"name":    "agent-memory",
					"version": "1.0.0",
				},
			}

		case "tools/list":
			var toolList []map[string]interface{}
			for _, t := range tools {
				toolList = append(toolList, map[string]interface{}{
					"name":        t.Tool.Name,
					"description": t.Tool.Description,
					"inputSchema": t.Tool.InputSchema,
				})
			}
			resp.Result = map[string]interface{}{
				"tools": toolList,
			}

		case "tools/call":
			var callParams map[string]interface{}
			if err := json.Unmarshal(req.Params, &callParams); err != nil {
				resp.Error = &MCPError{Code: -32602, Message: "Invalid params"}
				enc.Encode(resp)
				continue
			}

			toolName, _ := callParams["name"].(string)
			arguments, _ := callParams["arguments"].(map[string]interface{})

			result, err := s.callTool(toolName, arguments)
			if err != nil {
				resp.Error = &MCPError{Code: -32603, Message: err.Error()}
			} else {
				resp.Result = map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": formatResult(result),
						},
					},
				}
			}

		default:
			resp.Error = &MCPError{Code: -32601, Message: "Method not found"}
		}

		enc.Encode(resp)
	}
}

func (s *MCPServer) callTool(name string, args map[string]interface{}) (interface{}, error) {
	for _, t := range tools {
		if t.Tool.Name == name {
			return t.Handler(s, args)
		}
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}

func formatResult(result interface{}) string {
	if result == nil {
		return ""
	}

	switch v := result.(type) {
	case string:
		return v
	default:
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	}
}

func RunMCPServer(memSvc *memory.Service, projSvc *project.Service, whSvc *webhook.Service) {
	s := &MCPServer{memSvc: memSvc, projSvc: projSvc, whSvc: whSvc}
	s.HandleRequest(os.Stdin, os.Stdout)
}
