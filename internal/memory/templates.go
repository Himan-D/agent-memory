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
	if pr == nil || pr.templates == nil || pr.templates["extractFacts"] == nil {
		return content, nil
	}
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
	if pr == nil || pr.templates == nil || pr.templates["shouldStore"] == nil {
		return content, nil
	}
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
	if pr == nil || pr.templates == nil || pr.templates["extractEntities"] == nil {
		return content, nil
	}
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
	if pr == nil || pr.templates == nil || pr.templates["resolveConflict"] == nil {
		return newContent, nil
	}
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
	if pr == nil || pr.templates == nil || pr.templates["extractCategories"] == nil {
		return content, nil
	}
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

// ==================== Skill Extraction Templates ====================

type ExtractedSkill struct {
	Name       string   `json:"name"`
	Domain     string   `json:"domain"`
	Trigger    string   `json:"trigger"`
	Action     string   `json:"action"`
	Confidence float32  `json:"confidence"`
	Examples   []string `json:"examples,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

type SkillExtractionResult struct {
	Skills      []ExtractedSkill `json:"skills"`
	ShouldStore bool             `json:"should_store"`
	Reason      string           `json:"reason,omitempty"`
}

var systemPromptExtractSkills = `You are a procedural memory extraction system. Extract reusable skills, patterns, and procedures from the input content.

A SKILL is a trigger-action pair that can be learned and reused:
- TRIGGER: What situation/cue activates this skill (e.g., "when user asks about Python", "when error 500 occurs")
- ACTION: What response/action to take (e.g., "use this code pattern", "check logs first")

Rules:
- Extract ONLY actionable skills that can be learned and reused
- Each skill should have: name, domain, trigger, action
- Confidence score: 0.0-1.0 (how sure are you this is a reusable pattern)
- Domain: What field/category does this belong to (e.g., coding, debugging, cooking, finance)
- Examples: Optional real-world usage examples
- Tags: Optional labels for categorization
- Return ONLY a valid JSON object, nothing else
- Format: {"skills": [{"name": "...", "domain": "...", "trigger": "...", "action": "...", "confidence": 0.8, "examples": [...], "tags": [...]}], "should_store": true, "reason": "..."}`

var userPromptExtractSkills = `Extract reusable skills and procedures from this content:
---
{{.Content}}
---
User ID: {{if .UserID}}{{.UserID}}{{else}}unknown{{end}}
Agent ID: {{if .AgentID}}{{.AgentID}}{{else}}unknown{{end}}

Return a JSON object with extracted skills. Each skill should capture a reusable pattern with trigger-action structure.`

var systemPromptSynthesizeSkills = `You are a skill synthesis system. Merge similar skills into more general, reusable patterns.

Given multiple similar skills, identify the common pattern and create a generalized version.

Rules:
- Return ONLY valid JSON: {"synthesized_skill": {...}, "reason": "...", "merged_count": N}
- Keep the best parts of each skill
- Generalize the trigger to be more broadly applicable
- Increase confidence if multiple sources agree
- Return ONLY valid JSON.`

var userPromptSynthesizeSkills = `Synthesize these similar skills into a single, more general skill:

SKILLS TO MERGE:
{{.Skills}}

Return a synthesized skill that captures the common pattern across all inputs.`

var systemPromptSuggestProcedure = `You are a procedure suggestion system. Given a context/trigger, suggest relevant procedures from the skill library.

Rules:
- Return ONLY valid JSON: {"suggestions": [{"skill_id": "...", "relevance": 0.9, "confidence": 0.8, "reason": "..."}]}
- Score by: trigger match (0.5), historical success (0.3), recency (0.2)
- Only suggest verified or high-confidence skills
- Return ONLY valid JSON.`

var userPromptSuggestProcedure = `Find relevant procedures for this context:

TRIGGER: {{.Trigger}}
CONTEXT: {{.Context}}

Available skills: {{.Skills}}

Return JSON with relevant skill suggestions scored by relevance.`

var systemPromptInferProcedure = `You are a procedure inference system. Convert multi-step interactions into structured procedures.

Given a conversation or interaction sequence, identify if it represents a learnable procedure.

Rules:
- Return ONLY valid JSON: {"is_procedure": true/false, "steps": [...], "preconditions": [...], "postconditions": [...], "trigger": "...", "confidence": 0.8}
- A procedure has clear steps that can be followed repeatedly
- Preconditions: What must be true before starting
- Postconditions: What results after completion
- Return ONLY valid JSON.`

var userPromptInferProcedure = `Analyze this interaction for learnable procedure:

---
{{.Content}}
---

Return JSON describing if this is a procedure and its structure.`

func (pr *PromptRenderer) RenderExtractSkills(content, userID, agentID string) (string, error) {
	if pr == nil || pr.templates == nil || pr.templates["extractSkills"] == nil {
		return content, nil
	}
	var buf bytes.Buffer
	data := struct {
		Content string
		UserID  string
		AgentID string
	}{
		Content: content,
		UserID:  userID,
		AgentID: agentID,
	}
	if err := pr.templates["extractSkills"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) RenderSynthesizeSkills(skillsJSON string) (string, error) {
	if pr == nil || pr.templates == nil || pr.templates["synthesizeSkills"] == nil {
		return skillsJSON, nil
	}
	var buf bytes.Buffer
	data := struct {
		Skills string
	}{
		Skills: skillsJSON,
	}
	if err := pr.templates["synthesizeSkills"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) RenderSuggestProcedure(trigger, context string, skillsJSON string) (string, error) {
	if pr == nil || pr.templates == nil || pr.templates["suggestProcedure"] == nil {
		return trigger, nil
	}
	var buf bytes.Buffer
	data := struct {
		Trigger string
		Context string
		Skills  string
	}{
		Trigger: trigger,
		Context: context,
		Skills:  skillsJSON,
	}
	if err := pr.templates["suggestProcedure"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) RenderInferProcedure(content string) (string, error) {
	if pr == nil || pr.templates == nil || pr.templates["inferProcedure"] == nil {
		return content, nil
	}
	var buf bytes.Buffer
	data := struct {
		Content string
	}{
		Content: content,
	}
	if err := pr.templates["inferProcedure"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) GetSystemPromptExtractSkills() string {
	return systemPromptExtractSkills
}

func (pr *PromptRenderer) GetSystemPromptSynthesizeSkills() string {
	return systemPromptSynthesizeSkills
}

func (pr *PromptRenderer) GetSystemPromptSuggestProcedure() string {
	return systemPromptSuggestProcedure
}

func (pr *PromptRenderer) GetSystemPromptInferProcedure() string {
	return systemPromptInferProcedure
}

func NewSkillPromptRenderer() *PromptRenderer {
	pr := &PromptRenderer{
		templates: make(map[string]*template.Template),
	}

	pr.templates["extractFacts"] = template.Must(template.New("extractFacts").Parse(userPromptExtractFacts))
	pr.templates["shouldStore"] = template.Must(template.New("shouldStore").Parse(userPromptShouldStore))
	pr.templates["extractEntities"] = template.Must(template.New("extractEntities").Parse(userPromptExtractEntities))
	pr.templates["resolveConflict"] = template.Must(template.New("resolveConflict").Parse(userPromptResolveConflict))
	pr.templates["extractCategories"] = template.Must(template.New("extractCategories").Parse(extractCategoriesPrompt))
	pr.templates["extractSkills"] = template.Must(template.New("extractSkills").Parse(userPromptExtractSkills))
	pr.templates["synthesizeSkills"] = template.Must(template.New("synthesizeSkills").Parse(userPromptSynthesizeSkills))
	pr.templates["suggestProcedure"] = template.Must(template.New("suggestProcedure").Parse(userPromptSuggestProcedure))
	pr.templates["inferProcedure"] = template.Must(template.New("inferProcedure").Parse(userPromptInferProcedure))

	return pr
}
