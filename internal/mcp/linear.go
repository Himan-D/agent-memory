package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Linear call types
type LinearCall struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type LinearRequest struct {
	Calls []LinearCall `json:"calls"`
}

type CallResult struct {
	Index   int           `json:"index"`
	Tool    string        `json:"tool"`
	Success bool          `json:"success"`
	Result  *ToolResult   `json:"result,omitempty"`
	Error   string        `json:"error,omitempty"`
	Elapsed time.Duration `json:"elapsed_ms,omitempty"`
}

type LinearResponse struct {
	Results []CallResult `json:"results"`
}

// ToolExecutor is an interface used by the linear executor so tests can inject
// a mock implementation. ToolHandler already implements this method.
type ToolExecutor interface {
	HandleToolCall(ctx context.Context, toolName string, params map[string]interface{}) (*ToolResult, error)
}

// ExecuteLinearCalls executes the provided calls sequentially using the
// provided executor. It returns the per-call results and does not stop on
// individual failures (the error is recorded per-call). Context cancellation
// stops further execution.
func ExecuteLinearCalls(ctx context.Context, executor ToolExecutor, calls []LinearCall) ([]CallResult, error) {
	results := make([]CallResult, 0, len(calls))

	for i, c := range calls {
		start := time.Now()
		select {
		case <-ctx.Done():
			results = append(results, CallResult{
				Index:   i,
				Tool:    c.Tool,
				Success: false,
				Error:   "context cancelled",
				Elapsed: time.Since(start) / time.Millisecond,
			})
			return results, ctx.Err()
		default:
		}

		res, err := executor.HandleToolCall(ctx, c.Tool, c.Params)
		cr := CallResult{
			Index:   i,
			Tool:    c.Tool,
			Elapsed: time.Since(start) / time.Millisecond,
		}
		if err != nil {
			cr.Success = false
			cr.Error = err.Error()
		} else {
			cr.Success = true
			cr.Result = res
		}
		results = append(results, cr)
	}

	return results, nil
}

// handleLinear is the HTTP handler wired at /mcp/linear. It expects a JSON
// body with {"calls": [...] } where each call contains tool and params.
func (s *MCPServer) handleLinear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LinearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("bad request: %v", err), http.StatusBadRequest)
		return
	}

	handler := NewToolHandler(s.memSvc)
	ctx := r.Context()
	results, err := ExecuteLinearCalls(ctx, handler, req.Calls)
	if err != nil {
		// return what we have along with error
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(LinearResponse{Results: results})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LinearResponse{Results: results})
}
