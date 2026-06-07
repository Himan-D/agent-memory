package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeMemoryCreate EventType = "memory.create"
	EventTypeMemoryRead   EventType = "memory.read"
	EventTypeMemoryUpdate EventType = "memory.update"
	EventTypeMemoryDelete EventType = "memory.delete"
	EventTypeMemorySearch EventType = "memory.search"
	EventTypeMemoryShare  EventType = "memory.share"

	EventTypeSkillCreate     EventType = "skill.create"
	EventTypeSkillUpdate     EventType = "skill.update"
	EventTypeSkillDelete     EventType = "skill.delete"
	EventTypeSkillUse        EventType = "skill.use"
	EventTypeSkillExecute    EventType = "skill.execute"
	EventTypeSkillExtract    EventType = "skill.extract"
	EventTypeSkillSynthesize EventType = "skill.synthesize"
	EventTypeSkillApprove    EventType = "skill.approve"
	EventTypeSkillReject     EventType = "skill.reject"
	EventTypeChainExecute    EventType = "chain.execute"

	EventTypeAgentCreate EventType = "agent.create"
	EventTypeAgentUpdate EventType = "agent.update"
	EventTypeAgentDelete EventType = "agent.delete"

	EventTypeGroupCreate       EventType = "group.create"
	EventTypeGroupAddMember    EventType = "group.add_member"
	EventTypeGroupRemoveMember EventType = "group.remove_member"

	EventTypeReviewRequest EventType = "review.request"
	EventTypeReviewApprove EventType = "review.approve"
	EventTypeReviewReject  EventType = "review.reject"

	EventTypeLicenseValidate EventType = "license.validate"
	EventTypeLicenseUpgrade  EventType = "license.upgrade"
	EventTypeLicenseRevoke   EventType = "license.revoke"

	EventTypeAuthLogin  EventType = "auth.login"
	EventTypeAuthLogout EventType = "auth.logout"
	EventTypeAuthSSO    EventType = "auth.sso"

	EventTypeAPI     EventType = "api.access"
	EventTypeAPIList EventType = "api.list"
)

type Event struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	Timestamp    time.Time              `json:"timestamp"`
	Type         EventType              `json:"type"`
	ActorID      string                 `json:"actor_id"`
	ActorType    string                 `json:"actor_type"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Action       string                 `json:"action"`
	Status       string                 `json:"status"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Error        string                 `json:"error,omitempty"`
	DurationMs   int64                  `json:"duration_ms,omitempty"`
}

type Logger interface {
	Log(ctx context.Context, event *Event) error
	Query(ctx context.Context, filter *Filter) ([]*Event, error)
	Export(ctx context.Context, filter *Filter, format string) ([]byte, error)
	GetStorage() Storage
}

type Filter struct {
	TenantID     string
	StartTime    time.Time
	EndTime      time.Time
	Types        []EventType
	ActorID      string
	ResourceType string
	ResourceID   string
	Status       string
	Limit        int
	Offset       int
}

type Storage interface {
	Store(ctx context.Context, event *Event) error
	Query(ctx context.Context, filter *Filter) ([]*Event, error)
	Count(ctx context.Context, filter *Filter) (int, error)
	DeleteOld(ctx context.Context, before time.Time) (int, error)
}

type logger struct {
	storage    Storage
	mu         sync.RWMutex
	buffer     []*Event
	bufferSize int
	flushMs    int
}

type LoggerConfig struct {
	BufferSize int
	FlushMs    int
	Storage    Storage
}

func NewLogger(cfg *LoggerConfig) (Logger, error) {
	if cfg == nil {
		cfg = &LoggerConfig{
			BufferSize: 100,
			FlushMs:    5000,
		}
	}

	if cfg.Storage == nil {
		cfg.Storage = NewInMemoryStorage()
	}

	l := &logger{
		storage:    cfg.Storage,
		buffer:     make([]*Event, 0, cfg.BufferSize),
		bufferSize: cfg.BufferSize,
		flushMs:    cfg.FlushMs,
	}

	go l.flushLoop()

	return l, nil
}

func (l *logger) flushLoop() {
	ticker := time.NewTicker(time.Duration(l.flushMs) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		l.flush()
	}
}

func (l *logger) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.buffer) == 0 {
		return
	}

	events := l.buffer
	l.buffer = make([]*Event, 0, l.bufferSize)

	ctx := context.Background()
	for _, event := range events {
		l.storage.Store(ctx, event)
	}
}

func (l *logger) Log(ctx context.Context, event *Event) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	l.mu.Lock()
	l.buffer = append(l.buffer, event)
	shouldFlush := len(l.buffer) >= l.bufferSize
	l.mu.Unlock()

	if shouldFlush {
		l.flush()
	}

	return nil
}

func (l *logger) Query(ctx context.Context, filter *Filter) ([]*Event, error) {
	return l.storage.Query(ctx, filter)
}

func (l *logger) Export(ctx context.Context, filter *Filter, format string) ([]byte, error) {
	events, err := l.storage.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(events, "", "  ")
	case "csv":
		return l.exportCSV(events)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

func (l *logger) exportCSV(events []*Event) ([]byte, error) {
	csv := "ID,TenantID,Timestamp,Type,ActorID,ActorType,ResourceType,ResourceID,Action,Status,IPAddress\n"

	for _, e := range events {
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			e.ID, e.TenantID, e.Timestamp.Format(time.RFC3339), e.Type,
			e.ActorID, e.ActorType, e.ResourceType, e.ResourceID,
			e.Action, e.Status, e.IPAddress)
	}

	return []byte(csv), nil
}

func (l *logger) GetStorage() Storage {
	return l.storage
}

type InMemoryStorage struct {
	mu     sync.RWMutex
	events []*Event
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		events: make([]*Event, 0),
	}
}

func (s *InMemoryStorage) Store(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)
	return nil
}

func (s *InMemoryStorage) Query(ctx context.Context, filter *Filter) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Event

	for _, e := range s.events {
		if filter != nil && filter.TenantID != "" && e.TenantID != filter.TenantID {
			continue
		}
		if filter != nil && filter.StartTime != (time.Time{}) && e.Timestamp.Before(filter.StartTime) {
			continue
		}
		if filter != nil && filter.EndTime != (time.Time{}) && e.Timestamp.After(filter.EndTime) {
			continue
		}
		if filter != nil && len(filter.Types) > 0 && !containsType(filter.Types, e.Type) {
			continue
		}
		if filter != nil && filter.ActorID != "" && e.ActorID != filter.ActorID {
			continue
		}
		if filter != nil && filter.ResourceType != "" && e.ResourceType != filter.ResourceType {
			continue
		}
		if filter != nil && filter.ResourceID != "" && e.ResourceID != filter.ResourceID {
			continue
		}
		if filter != nil && filter.Status != "" && e.Status != filter.Status {
			continue
		}

		results = append(results, e)
	}

	if filter != nil {
		if filter.Limit > 0 && len(results) > filter.Limit {
			results = results[:filter.Limit]
		}
		if filter.Offset > 0 && filter.Offset < len(results) {
			results = results[filter.Offset:]
		}
	}

	return results, nil
}

func (s *InMemoryStorage) Count(ctx context.Context, filter *Filter) (int, error) {
	events, err := s.Query(ctx, filter)
	if err != nil {
		return 0, err
	}
	return len(events), nil
}

func (s *InMemoryStorage) DeleteOld(ctx context.Context, before time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldCount := 0
	for _, e := range s.events {
		if e.Timestamp.Before(before) {
			oldCount++
		}
	}

	newEvents := make([]*Event, 0)
	for _, e := range s.events {
		if !e.Timestamp.Before(before) {
			newEvents = append(newEvents, e)
		}
	}

	s.events = newEvents
	return oldCount, nil
}

func containsType(types []EventType, t EventType) bool {
	for _, typ := range types {
		if typ == t {
			return true
		}
	}
	return false
}

// FileAuditStorage persists audit events to a JSON-lines file and caches
// them in memory for fast querying. It implements the Storage interface.
type FileAuditStorage struct {
	mu       sync.RWMutex
	filePath string
	file     *os.File
	events   []*Event
}

// NewFileAuditStorage opens (or creates) the file at path, loads any
// existing events into the in-memory cache, and returns the storage.
func NewFileAuditStorage(path string) (*FileAuditStorage, error) {
	if err := os.MkdirAll(filepath(path), 0755); err != nil {
		return nil, fmt.Errorf("audit file storage: mkdir: %w", err)
	}

	// Load existing events before opening for append.
	existing, err := loadAuditEvents(path)
	if err != nil {
		return nil, fmt.Errorf("audit file storage: load: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("audit file storage: open: %w", err)
	}

	return &FileAuditStorage{
		filePath: path,
		file:     f,
		events:   existing,
	}, nil
}

// filepath returns the directory portion of a file path.
func filepath(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

// loadAuditEvents reads an existing JSON-lines file and returns all events.
func loadAuditEvents(path string) ([]*Event, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make([]*Event, 0), nil
		}
		return nil, err
	}
	defer f.Close()

	var events []*Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err == nil {
			events = append(events, &e)
		}
	}
	return events, scanner.Err()
}

func (s *FileAuditStorage) Store(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit file storage: marshal: %w", err)
	}
	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("audit file storage: write: %w", err)
	}
	s.events = append(s.events, event)
	return nil
}

func (s *FileAuditStorage) Query(ctx context.Context, filter *Filter) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Event
	for _, e := range s.events {
		if filter != nil && filter.TenantID != "" && e.TenantID != filter.TenantID {
			continue
		}
		if filter != nil && filter.StartTime != (time.Time{}) && e.Timestamp.Before(filter.StartTime) {
			continue
		}
		if filter != nil && filter.EndTime != (time.Time{}) && e.Timestamp.After(filter.EndTime) {
			continue
		}
		if filter != nil && len(filter.Types) > 0 && !containsType(filter.Types, e.Type) {
			continue
		}
		if filter != nil && filter.ActorID != "" && e.ActorID != filter.ActorID {
			continue
		}
		if filter != nil && filter.ResourceType != "" && e.ResourceType != filter.ResourceType {
			continue
		}
		if filter != nil && filter.ResourceID != "" && e.ResourceID != filter.ResourceID {
			continue
		}
		if filter != nil && filter.Status != "" && e.Status != filter.Status {
			continue
		}
		results = append(results, e)
	}

	if filter != nil {
		if filter.Offset > 0 && filter.Offset < len(results) {
			results = results[filter.Offset:]
		}
		if filter.Limit > 0 && len(results) > filter.Limit {
			results = results[:filter.Limit]
		}
	}

	return results, nil
}

func (s *FileAuditStorage) Count(ctx context.Context, filter *Filter) (int, error) {
	events, err := s.Query(ctx, filter)
	if err != nil {
		return 0, err
	}
	return len(events), nil
}

// DeleteOld removes events older than before from memory and rewrites the
// file to reflect the deletion.
func (s *FileAuditStorage) DeleteOld(ctx context.Context, before time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var kept []*Event
	deleted := 0
	for _, e := range s.events {
		if e.Timestamp.Before(before) {
			deleted++
		} else {
			kept = append(kept, e)
		}
	}

	if deleted == 0 {
		return 0, nil
	}

	// Rewrite the file with only the kept events.
	if err := s.file.Close(); err != nil {
		return 0, fmt.Errorf("audit file storage: close for rewrite: %w", err)
	}
	f, err := os.OpenFile(s.filePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("audit file storage: reopen for rewrite: %w", err)
	}
	for _, e := range kept {
		data, _ := json.Marshal(e)
		f.Write(append(data, '\n'))
	}
	s.file = f
	s.events = kept
	return deleted, nil
}

// Close flushes and closes the underlying file.
func (s *FileAuditStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.file.Close()
}

// NewPersistentLogger creates an audit Logger backed by file-based storage.
// The audit log is appended to path on every flush.
func NewPersistentLogger(path string, cfg *LoggerConfig) (Logger, error) {
	storage, err := NewFileAuditStorage(path)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &LoggerConfig{}
	}
	cfg.Storage = storage
	return NewLogger(cfg)
}

type EventBuilder struct {
	event *Event
}

func NewEventBuilder() *EventBuilder {
	return &EventBuilder{
		event: &Event{
			Timestamp: time.Now(),
		},
	}
}

func (b *EventBuilder) TenantID(id string) *EventBuilder {
	b.event.TenantID = id
	return b
}

func (b *EventBuilder) Type(t EventType) *EventBuilder {
	b.event.Type = t
	return b
}

func (b *EventBuilder) Actor(id, actorType string) *EventBuilder {
	b.event.ActorID = id
	b.event.ActorType = actorType
	return b
}

func (b *EventBuilder) Resource(resourceType, resourceID string) *EventBuilder {
	b.event.ResourceType = resourceType
	b.event.ResourceID = resourceID
	return b
}

func (b *EventBuilder) Action(action string) *EventBuilder {
	b.event.Action = action
	return b
}

func (b *EventBuilder) Status(status string) *EventBuilder {
	b.event.Status = status
	return b
}

func (b *EventBuilder) IPAddress(ip string) *EventBuilder {
	b.event.IPAddress = ip
	return b
}

func (b *EventBuilder) UserAgent(ua string) *EventBuilder {
	b.event.UserAgent = ua
	return b
}

func (b *EventBuilder) Metadata(meta map[string]interface{}) *EventBuilder {
	b.event.Metadata = meta
	return b
}

func (b *EventBuilder) Error(err string) *EventBuilder {
	b.event.Error = err
	return b
}

func (b *EventBuilder) Duration(ms int64) *EventBuilder {
	b.event.DurationMs = ms
	return b
}

func (b *EventBuilder) Build() *Event {
	return b.event
}
