package memory

import (
	"bytes"
	"fmt"
	"text/template"
)

type MemoryType string

const (
	MemoryTypeConversation MemoryType = "conversation"
	MemoryTypeSession      MemoryType = "session"
	MemoryTypeUser         MemoryType = "user"
	MemoryTypeOrg          MemoryType = "org"
)

const (
	ImportanceHigh   = "high"
	ImportanceMedium = "medium"
	ImportanceLow    = "low"
)

const (
	EntityTypePerson = "person"
	EntityTypePlace  = "place"
	EntityTypeOrg    = "organization"
	EntityTypeThing  = "thing"
)

type ExtractedFact struct {
	Fact       string `json:"fact"`
	Category   string `json:"category"`
	Importance string `json:"importance"`
}

type ExtractedEntity struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Mentions int    `json:"mentions"`
}

type MemoryProcessingResult struct {
	ProcessedContent string            `json:"processed_content"`
	Facts            []ExtractedFact   `json:"facts"`
	Entities         []ExtractedEntity `json:"entities"`
	Importance       string            `json:"importance"`
	ShouldStore      bool              `json:"should_store"`
	Reason           string            `json:"reason,omitempty"`
	Categories       []string          `json:"categories"`
}

type ShouldStoreResult struct {
	Store      bool     `json:"store"`
	Importance int      `json:"importance"`
	Reason     string   `json:"reason"`
	Categories []string `json:"categories"`
}

var systemPromptExtractFacts = `You are a memory extraction system. Extract key facts, preferences, and important information from the input content.

Rules:
- Extract ONLY information worth remembering long-term (preferences, facts, decisions, requirements, goals, constraints)
- Each fact should be concise (under 30 words)
- Focus on: preferences, facts, decisions, requirements, likes, dislikes, skills, constraints
- IMPORTANT: Return ONLY a valid JSON array, nothing else
- Format: [{{"fact": "...", "category": "...", "importance": "high|medium|low"}}]
- Categories: preference, fact, decision, requirement, goal, skill, constraint, personal, work, health, other`

var userPromptExtractFacts = `Extract memories from this content:
---
{{.Content}}
---
User ID: {{if .UserID}}{{.UserID}}{{else}}unknown{{end}}
Memory type: {{.MemoryType}}

Return a JSON array of extracted facts. Each fact should capture the essential information.
Return ONLY the JSON array, no other text.`

var systemPromptShouldStore = `You are a memory importance classifier. Determine if content contains information worth storing as a long-term memory.

Rules:
- Return JSON with: {"store": true/false, "importance": 1-10, "reason": "...", "categories": [...]}
- Store if content has: preferences, decisions, facts about user, requirements, goals, constraints
- Don't store if: generic statements, questions, obvious things, duplicates
- importance 1-3: Low (generic, obvious)
- importance 4-6: Medium (useful but not critical)
- importance 7-10: High (critical preferences, decisions, constraints)
- Return ONLY valid JSON, nothing else.`

var userPromptShouldStore = `Analyze this content for memory importance:
---
{{.Content}}
---

Return JSON with store decision, importance score (1-10), reason, and categories.`

var systemPromptExtractEntities = `You are an entity extraction system. Extract named entities (people, places, organizations, things) from text.

Rules:
- Return ONLY a valid JSON array, nothing else
- Format: [{{"name": "...", "type": "person|place|organization|thing", "mentions": N}}]
- Only extract specific named entities, not generic references
- "mentions" should be the number of times this entity appears
- Return ONLY the JSON array.`

var userPromptExtractEntities = `Extract all named entities from this content:
---
{{.Content}}
---

Return a JSON array of entities with their types.`

var systemPromptResolveConflict = `You are a memory conflict resolution system. When new information contradicts existing memories, determine how to resolve it.

Rules:
- Return ONLY valid JSON: {"action": "update|keep_both|discard_new", "updated_content": "...", "reason": "..."}
- "update": Replace old memory with new, more relevant information
- "keep_both": Keep both memories as they may be contextually relevant
- "discard_new": New information is less reliable/important than existing
- Consider: recency, importance, specificity, source reliability
- Return ONLY valid JSON.`

var userPromptResolveConflict = `Compare existing memory with new information:

EXISTING: {{.ExistingContent}}
IMPORTANCE: {{.ExistingImportance}}

NEW: {{.NewContent}}

Determine the best resolution action.`

var extractCategoriesPrompt = `You are a memory categorization system.

Categories available:
- preference: User likes/dislikes, habits
- fact: Factual information about user or world
- decision: Decisions made by user
- requirement: User needs or constraints
- goal: User objectives or targets
- skill: User capabilities or knowledge
- personal: Personal information
- work: Work-related information
- health: Health or medical information
- other: Doesn't fit other categories

Return a JSON array of categories that apply to this content:
---
{{.Content}}
---

Return ONLY JSON array like: ["preference", "personal"]`

type PromptRenderer struct {
	templates map[string]*template.Template
}

func NewPromptRenderer() *PromptRenderer {
	pr := &PromptRenderer{
		templates: make(map[string]*template.Template),
	}

	pr.templates["extractFacts"] = template.Must(template.New("extractFacts").Parse(userPromptExtractFacts))
	pr.templates["shouldStore"] = template.Must(template.New("shouldStore").Parse(userPromptShouldStore))
	pr.templates["extractEntities"] = template.Must(template.New("extractEntities").Parse(userPromptExtractEntities))
	pr.templates["resolveConflict"] = template.Must(template.New("resolveConflict").Parse(userPromptResolveConflict))
	pr.templates["extractCategories"] = template.Must(template.New("extractCategories").Parse(extractCategoriesPrompt))

	return pr
}

func (pr *PromptRenderer) RenderExtractFacts(content, userID, memType string) (string, error) {
	var buf bytes.Buffer
	data := struct {
		Content    string
		UserID     string
		MemoryType string
	}{
		Content:    content,
		UserID:     userID,
		MemoryType: memType,
	}
	if err := pr.templates["extractFacts"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) RenderShouldStore(content string) (string, error) {
	var buf bytes.Buffer
	data := struct {
		Content string
	}{
		Content: content,
	}
	if err := pr.templates["shouldStore"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) RenderExtractEntities(content string) (string, error) {
	var buf bytes.Buffer
	data := struct {
		Content string
	}{
		Content: content,
	}
	if err := pr.templates["extractEntities"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) RenderResolveConflict(existingContent, existingImportance, newContent string) (string, error) {
	var buf bytes.Buffer
	data := struct {
		ExistingContent    string
		ExistingImportance string
		NewContent         string
	}{
		ExistingContent:    existingContent,
		ExistingImportance: existingImportance,
		NewContent:         newContent,
	}
	if err := pr.templates["resolveConflict"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) RenderExtractCategories(content string) (string, error) {
	var buf bytes.Buffer
	data := struct {
		Content string
	}{
		Content: content,
	}
	if err := pr.templates["extractCategories"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) GetSystemPromptExtractFacts() string {
	return systemPromptExtractFacts
}

func (pr *PromptRenderer) GetSystemPromptShouldStore() string {
	return systemPromptShouldStore
}

func (pr *PromptRenderer) GetSystemPromptExtractEntities() string {
	return systemPromptExtractEntities
}

func (pr *PromptRenderer) GetSystemPromptResolveConflict() string {
	return systemPromptResolveConflict
}

func (pr *PromptRenderer) GetSystemPromptExtractCategories() string {
	return `You are a memory categorization system.`
}

type Config struct {
	Enabled             bool
	AutoExtractFacts    bool
	AutoExtractEntities bool
	DefaultImportance   string
}

func DefaultConfig() *Config {
	return &Config{
		Enabled:             true,
		AutoExtractFacts:    true,
		AutoExtractEntities: true,
		DefaultImportance:   ImportanceMedium,
	}
}

type ConflictResolutionAction string

const (
	ConflictActionUpdate     ConflictResolutionAction = "update"
	ConflictActionKeepBoth   ConflictResolutionAction = "keep_both"
	ConflictActionDiscardNew ConflictResolutionAction = "discard_new"
)

type ConflictResolutionResult struct {
	Action         ConflictResolutionAction `json:"action"`
	UpdatedContent string                   `json:"updated_content,omitempty"`
	Reason         string                   `json:"reason"`
}

func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("memory processing error: %v", err)
}
