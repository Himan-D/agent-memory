package mcp

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockExecutor struct {
	responses map[string]*ToolResult
	errors    map[string]error
}

func (m *mockExecutor) HandleToolCall(ctx context.Context, toolName string, params map[string]interface{}) (*ToolResult, error) {
	if e, ok := m.errors[toolName]; ok {
		return nil, e
	}
	if r, ok := m.responses[toolName]; ok {
		return r, nil
	}
	return &ToolResult{Content: []ContentBlock{{Type: "text", Text: "ok"}}}, nil
}

func TestExecuteLinearCalls_AllSuccess(t *testing.T) {
	exec := &mockExecutor{responses: map[string]*ToolResult{
		"addMemory": {Content: []ContentBlock{{Type: "text", Text: "added"}}},
		"recall":    {Content: []ContentBlock{{Type: "text", Text: "found"}}},
	}, errors: map[string]error{}}

	calls := []LinearCall{{Tool: "addMemory", Params: map[string]interface{}{"content": "x"}}, {Tool: "recall", Params: map[string]interface{}{"query": "x"}}}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := ExecuteLinearCalls(ctx, exec, calls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	for i, r := range res {
		if !r.Success {
			t.Fatalf("call %d expected success, got error: %s", i, r.Error)
		}
	}
}

func TestExecuteLinearCalls_PartialFailure(t *testing.T) {
	exec := &mockExecutor{responses: map[string]*ToolResult{}, errors: map[string]error{"recall": errors.New("search failed")}}

	calls := []LinearCall{{Tool: "addMemory", Params: map[string]interface{}{"content": "x"}}, {Tool: "recall", Params: map[string]interface{}{"query": "x"}}}

	ctx := context.Background()
	res, err := ExecuteLinearCalls(ctx, exec, calls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if !res[0].Success {
		t.Fatalf("first call should succeed")
	}
	if res[1].Success {
		t.Fatalf("second call should have failed")
	}
}
