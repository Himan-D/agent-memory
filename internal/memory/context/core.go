package context

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// CoreMemoryStore is the persistence backend for core memory sections.
type CoreMemoryStore interface {
	Load(ctx context.Context, userID string) (map[string]string, error)
	Save(ctx context.Context, userID string, sections map[string]string) error
}

// CoreMemory holds persistent user/agent facts that are always loaded into context.
// Modeled after MemGPT's "core memory" — a small, always-present block of critical
// information about the user and the agent's persona.
type CoreMemory struct {
	mu       sync.RWMutex
	sections map[string]string // e.g., "user_bio": "Alice is a data scientist..."
	store    CoreMemoryStore   // persistence backend (may be nil for in-memory only)
}

// NewCoreMemory creates a CoreMemory instance. The store parameter may be nil,
// in which case LoadForUser/SaveForUser will return errors.
func NewCoreMemory(store CoreMemoryStore) *CoreMemory {
	return &CoreMemory{
		sections: make(map[string]string),
		store:    store,
	}
}

// Get returns the content of a named section. Returns empty string if not found.
func (cm *CoreMemory) Get(section string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.sections[section]
}

// Set updates a section of core memory. This is the agent self-edit mechanism —
// the agent can modify its own persistent knowledge about the user.
func (cm *CoreMemory) Set(section, content string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if content == "" {
		delete(cm.sections, section)
	} else {
		cm.sections[section] = content
	}
}

// LoadForUser loads core memory sections from the persistent store.
func (cm *CoreMemory) LoadForUser(ctx context.Context, userID string) error {
	if cm.store == nil {
		return fmt.Errorf("context: core memory: no store configured")
	}
	loaded, err := cm.store.Load(ctx, userID)
	if err != nil {
		return fmt.Errorf("context: core memory: load: %w", err)
	}
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.sections = make(map[string]string, len(loaded))
	for k, v := range loaded {
		cm.sections[k] = v
	}
	return nil
}

// SaveForUser persists the current core memory state to the store.
func (cm *CoreMemory) SaveForUser(ctx context.Context, userID string) error {
	if cm.store == nil {
		return fmt.Errorf("context: core memory: no store configured")
	}
	cm.mu.RLock()
	snapshot := make(map[string]string, len(cm.sections))
	for k, v := range cm.sections {
		snapshot[k] = v
	}
	cm.mu.RUnlock()
	if err := cm.store.Save(ctx, userID, snapshot); err != nil {
		return fmt.Errorf("context: core memory: save: %w", err)
	}
	return nil
}

// FormatForPrompt renders all sections as a prompt-ready string.
// Sections are sorted by key for deterministic output.
func (cm *CoreMemory) FormatForPrompt() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	if len(cm.sections) == 0 {
		return ""
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(cm.sections))
	for k := range cm.sections {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("<core_memory>\n")
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("[%s]\n%s\n\n", k, cm.sections[k]))
	}
	sb.WriteString("</core_memory>")
	return sb.String()
}

// Sections returns a copy of all current sections.
func (cm *CoreMemory) Sections() map[string]string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	out := make(map[string]string, len(cm.sections))
	for k, v := range cm.sections {
		out[k] = v
	}
	return out
}
