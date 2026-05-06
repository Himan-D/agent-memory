package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
)

type Router struct {
	config      *Config
	memSvc      *memory.Service
	llmProvider llm.Provider
	httpClient  *http.Client
	sessionStore *SessionStore
}

type Config struct {
	UpstreamURL      string        `env:"ROUTER_UPSTREAM_URL" envDefault:"https://api.openai.com/v1"`
	APIKey          string        `env:"ROUTER_API_KEY" envDefault:""`
	Model           string        `env:"ROUTER_MODEL" envDefault:"gpt-4o"`
	MaxContextTokens int           `env:"ROUTER_MAX_CONTEXT_TOKENS" envDefault:"128000"`
	MaxMemoryTokens int          `env:"ROUTER_MAX_MEMORY_TOKENS" envDefault:"32000"`
	ChunkSize       int           `env:"ROUTER_CHUNK_SIZE" envDefault:"8000"`
	RedisAddr       string        `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	RedisTTL        time.Duration `env:"ROUTER_CACHE_TTL" envDefault:"24h"`
}

type Session struct {
	ID         string        `json:"id"`
	UserID     string        `json:"user_id"`
	Messages   []ChatMessage `json:"messages"`
	Summary    string        `json:"summary"`
	TokenCount int         `json:"token_count"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Tokens  int    `json:"tokens,omitempty"`
}

type SessionStore struct {
	client *redis.Client
	ttl    time.Duration
}

type RouterResponse struct {
	Content       string `json:"content"`
	Tokens        int    `json:"tokens"`
	Model         string `json:"model"`
	SessionTokens int    `json:"session_tokens,omitempty"`
}

type ChatRequest struct {
	SessionID string `json:"session_id,omitempty"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	System    string `json:"system,omitempty"`
}

func NewRouter(cfg *Config, memSvc *memory.Service, llmProvider llm.Provider) (*Router, error) {
	r := &Router{
		config:      cfg,
		memSvc:     memSvc,
		llmProvider: llmProvider,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}

	if cfg.RedisAddr != "" {
		store, err := NewSessionStore(cfg.RedisAddr, cfg.RedisTTL)
		if err != nil {
			return nil, err
		}
		r.sessionStore = store
	}

	return r, nil
}

func (r *Router) HandleChat(ctx context.Context, req *ChatRequest) (*RouterResponse, error) {
	session, err := r.getOrCreateSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	session.Messages = append(session.Messages, ChatMessage{
		Role:    "user",
		Content: req.Message,
	})

	var contextTokens int
	var contextStr string

	if r.memSvc != nil && session.TokenCount > r.config.ChunkSize {
		contextStr, contextTokens, err = r.retrieveContext(ctx, req.UserID, req.Message)
		if err != nil {
			contextStr = ""
			contextTokens = 0
		}
	}

	messages := r.buildMessages(contextStr, session.Messages)

	resp, err := r.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model:      r.config.Model,
		Messages:   messages,
		Temperature: 0.7,
		MaxTokens:  4096,
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	session.Messages = append(session.Messages, ChatMessage{
		Role:    "assistant",
		Content: resp.Content,
	})

	session.TokenCount += estimateTokens(req.Message) + estimateTokens(resp.Content)
	session.UpdatedAt = time.Now()

	if err := r.saveSession(ctx, session); err != nil {
		return nil, err
	}

	return &RouterResponse{
		Content:       resp.Content,
		Tokens:        resp.Tokens,
		Model:         resp.Model,
		SessionTokens: contextTokens,
	}, nil
}

func (r *Router) retrieveContext(ctx context.Context, userID, query string) (string, int, error) {
	results, err := r.memSvc.SearchMemories(ctx, &types.SearchRequest{
		Query:  query,
		Limit:  10,
		UserID: userID,
	})
	if err != nil {
		return "", 0, err
	}

	var b strings.Builder
	var tokens int

	for _, mem := range results {
		if mem.Metadata == nil {
			continue
		}
		if tokens > r.config.MaxMemoryTokens {
			break
		}
		b.WriteString(fmt.Sprintf("[%s]: %s\n\n", mem.Metadata.Type, mem.Metadata.Content))
		tokens += estimateTokens(mem.Metadata.Content)
	}

	return b.String(), tokens, nil
}

func (r *Router) buildMessages(systemContext string, messages []ChatMessage) []llm.Message {
	var result []llm.Message

	systemPrompt := "You are a helpful AI assistant."
	if systemContext != "" {
		systemPrompt += "\n\nRelevant context from memory:\n" + systemContext
	}

	result = append(result, llm.Message{Role: "system", Content: systemPrompt})

	for _, msg := range messages {
		result = append(result, llm.Message{Role: msg.Role, Content: msg.Content})
	}

	return result
}

func (r *Router) getOrCreateSession(ctx context.Context, sessionID, userID string) (*Session, error) {
	if sessionID == "" {
		sessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}

	if r.sessionStore != nil {
		session, err := r.sessionStore.Get(ctx, sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return &Session{
		ID:         sessionID,
		UserID:     userID,
		Messages:   []ChatMessage{},
		CreatedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (r *Router) saveSession(ctx context.Context, session *Session) error {
	if r.sessionStore != nil {
		return r.sessionStore.Set(ctx, session)
	}
	return nil
}

func (r *Router) ClearSession(ctx context.Context, sessionID string) error {
	if r.sessionStore != nil {
		return r.sessionStore.Delete(ctx, sessionID)
	}
	return nil
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func NewSessionStore(addr string, ttl time.Duration) (*SessionStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &SessionStore{client: client, ttl: ttl}, nil
}

func (s *SessionStore) Get(ctx context.Context, id string) (*Session, error) {
	data, err := s.client.Get(ctx, "session:"+id).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *SessionStore) Set(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "session:"+session.ID, data, s.ttl).Err()
}

func (s *SessionStore) Delete(ctx context.Context, id string) error {
	return s.client.Del(ctx, "session:"+id).Err()
}

func (s *SessionStore) Close() error {
	return s.client.Close()
}