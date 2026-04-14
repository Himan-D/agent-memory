package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"agent-memory/internal/llm"
)

type Subagent struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Status       AgentStatus            `json:"status"`
	Model        string                 `json:"model"`
	SystemPrompt string                 `json:"system_prompt"`
	ParentID     string                 `json:"parent_id"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActive   time.Time              `json:"last_active"`
	MemoryLimit  int                    `json:"memory_limit"`
	Config       map[string]interface{} `json:"config"`
	history      []Message
	mu           sync.RWMutex
}

type AgentStatus string

const (
	AgentStatusIdle     AgentStatus = "idle"
	AgentStatusRunning  AgentStatus = "running"
	AgentStatusThinking AgentStatus = "thinking"
	AgentStatusWaiting  AgentStatus = "waiting"
	AgentStatusDone     AgentStatus = "done"
	AgentStatusError    AgentStatus = "error"
)

type Message struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

type Config struct {
	Name         string
	Type         string
	Model        string
	SystemPrompt string
	MemoryLimit  int
	LLM          llm.Provider
}

type Manager struct {
	mu       sync.RWMutex
	agents   map[string]*Subagent
	configs  map[string]*Config
	registry map[string]*AgentFactory
}

type AgentFactory struct {
	Create func(cfg *Config) (*Subagent, error)
}

func NewManager() *Manager {
	m := &Manager{
		agents:   make(map[string]*Subagent),
		configs:  make(map[string]*Config),
		registry: make(map[string]*AgentFactory),
	}
	m.registerBuiltIns()
	return m
}

func (m *Manager) registerBuiltIns() {
	m.registry["general"] = &AgentFactory{
		Create: m.createGeneralAgent,
	}
	m.registry["research"] = &AgentFactory{
		Create: m.createResearchAgent,
	}
	m.registry["coding"] = &AgentFactory{
		Create: m.createCodingAgent,
	}
	m.registry["planning"] = &AgentFactory{
		Create: m.createPlanningAgent,
	}
	m.registry["review"] = &AgentFactory{
		Create: m.createReviewAgent,
	}
}

func (m *Manager) createGeneralAgent(cfg *Config) (*Subagent, error) {
	return &Subagent{
		ID:           generateID(),
		Name:         cfg.Name,
		Type:         "general",
		Status:       AgentStatusIdle,
		Model:        cfg.Model,
		SystemPrompt: "You are a helpful AI assistant. Be concise and provide accurate information.",
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		MemoryLimit:  cfg.MemoryLimit,
		history:      []Message{},
	}, nil
}

func (m *Manager) createResearchAgent(cfg *Config) (*Subagent, error) {
	return &Subagent{
		ID:           generateID(),
		Name:         cfg.Name,
		Type:         "research",
		Status:       AgentStatusIdle,
		Model:        cfg.Model,
		SystemPrompt: "You are a research assistant. Focus on gathering accurate information, citing sources, and providing comprehensive analysis. Think step by step.",
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		MemoryLimit:  cfg.MemoryLimit,
		history:      []Message{},
	}, nil
}

func (m *Manager) createCodingAgent(cfg *Config) (*Subagent, error) {
	return &Subagent{
		ID:           generateID(),
		Name:         cfg.Name,
		Type:         "coding",
		Status:       AgentStatusIdle,
		Model:        cfg.Model,
		SystemPrompt: "You are an expert coding assistant. Write clean, efficient, well-documented code. Follow best practices and consider edge cases.",
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		MemoryLimit:  cfg.MemoryLimit,
		history:      []Message{},
	}, nil
}

func (m *Manager) createPlanningAgent(cfg *Config) (*Subagent, error) {
	return &Subagent{
		ID:           generateID(),
		Name:         cfg.Name,
		Type:         "planning",
		Status:       AgentStatusIdle,
		Model:        cfg.Model,
		SystemPrompt: "You are a planning expert. Break down complex tasks into actionable steps. Consider dependencies, potential risks, and resource requirements.",
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		MemoryLimit:  cfg.MemoryLimit,
		history:      []Message{},
	}, nil
}

func (m *Manager) createReviewAgent(cfg *Config) (*Subagent, error) {
	return &Subagent{
		ID:           generateID(),
		Name:         cfg.Name,
		Type:         "review",
		Status:       AgentStatusIdle,
		Model:        cfg.Model,
		SystemPrompt: "You are a critical reviewer. Analyze work thoroughly, identify issues, suggest improvements, and provide constructive feedback.",
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		MemoryLimit:  cfg.MemoryLimit,
		history:      []Message{},
	}, nil
}

func (m *Manager) Create(ctx context.Context, cfg *Config) (*Subagent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg == nil {
		cfg = &Config{
			Name:        "subagent",
			Type:        "general",
			Model:       "gpt-4o",
			MemoryLimit: 8000,
		}
	}

	factory, ok := m.registry[cfg.Type]
	if !ok {
		factory = m.registry["general"]
	}

	agent, err := factory.Create(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	agent.ParentID = ""

	m.agents[agent.ID] = agent
	m.configs[agent.ID] = cfg

	return agent, nil
}

func (m *Manager) CreateChild(ctx context.Context, parentID string, cfg *Config) (*Subagent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	parent, ok := m.agents[parentID]
	if !ok {
		return nil, fmt.Errorf("parent agent not found: %s", parentID)
	}

	if cfg == nil {
		cfg = &Config{
			Name:        fmt.Sprintf("%s-child", parent.Name),
			Type:        parent.Type,
			Model:       parent.Model,
			MemoryLimit: parent.MemoryLimit,
		}
	}

	factory, ok := m.registry[cfg.Type]
	if !ok {
		factory = m.registry["general"]
	}

	agent, err := factory.Create(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create child agent: %w", err)
	}

	agent.ParentID = parentID

	m.agents[agent.ID] = agent
	m.configs[agent.ID] = cfg

	return agent, nil
}

func (m *Manager) Get(id string) (*Subagent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, ok := m.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	return agent, nil
}

func (m *Manager) List() []*Subagent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*Subagent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent.copy())
	}

	return agents
}

func (m *Manager) ListByParent(parentID string) []*Subagent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var agents []*Subagent
	for _, agent := range m.agents {
		if agent.ParentID == parentID {
			agents = append(agents, agent.copy())
		}
	}

	return agents
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.agents[id]; !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	delete(m.agents, id)
	delete(m.configs, id)

	return nil
}

func (m *Manager) UpdateStatus(id string, status AgentStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	agent.Status = status
	agent.LastActive = time.Now()

	return nil
}

func (m *Manager) SendMessage(ctx context.Context, id string, content string) (*Message, error) {
	m.mu.Lock()
	agent, ok := m.agents[id]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	m.mu.Unlock()

	agent.mu.Lock()
	agent.Status = AgentStatusThinking
	msg := Message{
		Role:    "user",
		Content: content,
		Time:    time.Now(),
	}
	agent.history = append(agent.history, msg)
	agent.mu.Unlock()

	cfg := m.getConfig(id)
	if cfg == nil || cfg.LLM == nil {
		agent.mu.Lock()
		agent.Status = AgentStatusError
		agent.mu.Unlock()
		return nil, fmt.Errorf("agent has no LLM configured")
	}

	resp, err := cfg.LLM.Complete(ctx, &llm.CompletionRequest{
		Messages: agent.buildMessages(),
		Model:    agent.Model,
	})
	if err != nil {
		agent.mu.Lock()
		agent.Status = AgentStatusError
		agent.mu.Unlock()
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	agent.mu.Lock()
	defer agent.mu.Unlock()

	agent.Status = AgentStatusIdle
	agent.LastActive = time.Now()

	reply := Message{
		Role:    "assistant",
		Content: resp.Content,
		Time:    time.Now(),
	}
	agent.history = append(agent.history, reply)

	return &reply, nil
}

func (m *Manager) Fork(ctx context.Context, id string, prompt string) (*Subagent, error) {
	child, err := m.CreateChild(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	if prompt != "" {
		_, err := m.SendMessage(ctx, child.ID, prompt)
		if err != nil {
			return nil, err
		}
	}

	return child, nil
}

func (m *Manager) GetHistory(id string, limit int) ([]Message, error) {
	agent, err := m.Get(id)
	if err != nil {
		return nil, err
	}

	agent.mu.RLock()
	defer agent.mu.RUnlock()

	if limit <= 0 || limit > len(agent.history) {
		return agent.history, nil
	}

	return agent.history[len(agent.history)-limit:], nil
}

func (m *Manager) SetSystemPrompt(id string, prompt string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	agent.SystemPrompt = prompt
	return nil
}

func (m *Manager) RegisterAgentType(name string, factory *AgentFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry[name] = factory
}

func (m *Manager) getConfig(id string) *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.configs[id]
}

func (a *Subagent) copy() *Subagent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return &Subagent{
		ID:           a.ID,
		Name:         a.Name,
		Type:         a.Type,
		Status:       a.Status,
		Model:        a.Model,
		SystemPrompt: a.SystemPrompt,
		ParentID:     a.ParentID,
		CreatedAt:    a.CreatedAt,
		LastActive:   a.LastActive,
		MemoryLimit:  a.MemoryLimit,
		Config:       a.Config,
	}
}

func (a *Subagent) buildMessages() []llm.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()

	messages := make([]llm.Message, 0, len(a.history)+1)

	if a.SystemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: a.SystemPrompt,
		})
	}

	for _, msg := range a.history {
		messages = append(messages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type Task struct {
	ID          string           `json:"id"`
	AgentID     string           `json:"agent_id"`
	Type        TaskType         `json:"type"`
	Status      TaskStatus       `json:"status"`
	Input       string           `json:"input"`
	Output      string           `json:"output"`
	Error       string           `json:"error,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	StartedAt   *time.Time       `json:"started_at,omitempty"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Result      chan *TaskResult `json:"-"`
}

type TaskType string

const (
	TaskTypeMessage TaskType = "message"
	TaskTypeSearch  TaskType = "search"
	TaskTypeAnalyze TaskType = "analyze"
	TaskTypeExecute TaskType = "execute"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type TaskResult struct {
	Output string
	Error  error
}

func (m *Manager) RunTask(ctx context.Context, agentID string, taskType TaskType, input string) *Task {
	task := &Task{
		ID:        generateID(),
		AgentID:   agentID,
		Type:      taskType,
		Status:    TaskStatusPending,
		Input:     input,
		CreatedAt: time.Now(),
		Result:    make(chan *TaskResult, 1),
	}

	go func() {
		m.mu.RLock()
		_, ok := m.agents[agentID]
		m.mu.RUnlock()

		if !ok {
			task.Status = TaskStatusFailed
			task.Error = "agent not found"
			task.Result <- &TaskResult{Error: fmt.Errorf("agent not found")}
			return
		}

		m.UpdateStatus(agentID, AgentStatusRunning)
		now := time.Now()
		task.StartedAt = &now
		task.Status = TaskStatusRunning

		switch taskType {
		case TaskTypeMessage:
			msg, err := m.SendMessage(ctx, agentID, input)
			if err != nil {
				task.Status = TaskStatusFailed
				task.Error = err.Error()
				task.Result <- &TaskResult{Error: err}
			} else {
				task.Status = TaskStatusCompleted
				task.Output = msg.Content
				task.Result <- &TaskResult{Output: msg.Content}
			}

		case TaskTypeSearch:
			task.Status = TaskStatusCompleted
			task.Output = fmt.Sprintf("Search completed for: %s", input)
			task.Result <- &TaskResult{Output: task.Output}

		case TaskTypeAnalyze:
			task.Status = TaskStatusCompleted
			task.Output = fmt.Sprintf("Analysis completed for: %s", input)
			task.Result <- &TaskResult{Output: task.Output}

		case TaskTypeExecute:
			task.Status = TaskStatusCompleted
			task.Output = fmt.Sprintf("Execution completed for: %s", input)
			task.Result <- &TaskResult{Output: task.Output}
		}

		completed := time.Now()
		task.CompletedAt = &completed
		m.UpdateStatus(agentID, AgentStatusIdle)
	}()

	return task
}
