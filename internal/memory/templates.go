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

type ExtractedRelation struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type MemoryProcessingResult struct {
	ProcessedContent string              `json:"processed_content"`
	Facts            []ExtractedFact     `json:"facts"`
	Entities         []ExtractedEntity   `json:"entities"`
	Relations        []ExtractedRelation `json:"relations"`
	Importance       string              `json:"importance"`
	ShouldStore      bool                `json:"should_store"`
	Reason           string              `json:"reason,omitempty"`
	Categories       []string            `json:"categories"`
}

type ShouldStoreResult struct {
	Store      bool     `json:"store"`
	Importance int      `json:"importance"`
	Reason     string   `json:"reason"`
	Categories []string `json:"categories"`
}

// systemPromptExtractFacts instructs the model to extract structured, long-term
// facts from conversational content. Uses chain-of-thought framing and strict
// JSON output constraints to improve extraction quality.
var systemPromptExtractFacts = `You are a precise memory extraction system. Your task is to identify and extract only the information that has long-term value from the input content.

THINK STEP-BY-STEP before producing output:
1. Read the content carefully and identify who is involved.
2. Ask: "What would a person want remembered about this in 6 months?"
3. Filter out transient, obvious, or generic statements.
4. Express each surviving fact as a single, self-contained sentence.

EXTRACTION RULES:
- Extract ONLY information worth remembering long-term: preferences, decisions, constraints, goals, skills, personal facts, professional requirements
- Each fact MUST be self-contained — understandable without the original conversation
- Maximum 30 words per fact; minimum 5 words
- Prefer specificity: "prefers dark roast coffee" over "likes coffee"
- DO NOT extract: greetings, filler phrases, obvious statements, or questions without answers

OUTPUT FORMAT — return ONLY a valid JSON array, no prose:
[
  {"fact": "<concise self-contained fact>", "category": "<category>", "importance": "<high|medium|low>"}
]

CATEGORIES: preference, fact, decision, requirement, goal, skill, constraint, personal, work, health, other

IMPORTANCE GUIDELINES:
- high: critical preferences, hard constraints, explicit decisions, health/safety info
- medium: useful preferences, goals, professional context, recurring patterns
- low: minor preferences, general interests, low-signal observations`

var userPromptExtractFacts = `Extract long-term memories from this content.

CONTENT:
---
{{.Content}}
---

Context:
- User ID: {{if .UserID}}{{.UserID}}{{else}}unknown{{end}}
- Memory type: {{.MemoryType}}

Step 1 — What is the topic and who is involved?
Step 2 — What information has lasting value beyond this conversation?
Step 3 — Express each lasting fact as a concise, self-contained sentence.

Return ONLY a JSON array of extracted facts. If nothing is worth storing, return [].`

// systemPromptShouldStore classifies whether content warrants long-term storage.
// Structured scoring rubric reduces model variance.
var systemPromptShouldStore = `You are a memory importance classifier. Decide whether the provided content contains information worth storing as a long-term memory.

THINK STEP-BY-STEP:
1. Identify the type of content (question, statement, preference, decision, fact, etc.)
2. Score each dimension below (0–10)
3. Compute the weighted average to get the final importance score

SCORING DIMENSIONS (weights):
- Specificity (0.3): Is the information concrete and specific vs. generic?
- Longevity (0.3): Will this still be relevant in 3–6 months?
- Uniqueness (0.2): Is this something not already obvious or common knowledge?
- Actionability (0.2): Can this info improve future responses or decisions?

STORE THRESHOLDS:
- importance 1–3 (low): Generic, obvious, or purely transient — DO NOT store
- importance 4–6 (medium): Moderately useful, store if specific to the user
- importance 7–10 (high): Critical preference, decision, or constraint — ALWAYS store

DO NOT STORE: greetings, meta-questions ("what can you do?"), single-word answers, timestamps without context

OUTPUT — return ONLY valid JSON:
{"store": <true|false>, "importance": <1-10>, "reason": "<one sentence>", "categories": [<list of applicable categories>]}`

var userPromptShouldStore = `Classify this content for long-term memory storage.

CONTENT:
---
{{.Content}}
---

Step 1 — What type of content is this?
Step 2 — Score specificity, longevity, uniqueness, and actionability (0–10 each).
Step 3 — Compute weighted importance score.
Step 4 — Decide: store or discard?

Return ONLY valid JSON with store decision, importance score (1–10), one-sentence reason, and categories.`

// systemPromptExtractEntities extracts named entities with few-shot examples to
// anchor the model's output format and calibrate what counts as a named entity.
var systemPromptExtractEntities = `You are a named-entity extraction system. Extract specific named entities from text.

ENTITY TYPES:
- person: real named individuals ("Alice Johnson", "Elon Musk")
- place: named locations ("San Francisco", "CERN", "the Eiffel Tower")
- organization: companies, institutions, teams ("Anthropic", "NHS", "Arsenal FC")
- thing: named products, technologies, events ("iPhone 16", "GPT-4", "React 18")

RULES:
- Extract ONLY specific named entities, NOT generic references ("the user", "a company", "some place")
- Count how many times each entity is mentioned (mentions field)
- If the same entity appears under different names, pick the most complete form
- Return ONLY a valid JSON array — no prose, no markdown

FEW-SHOT EXAMPLES:

Input: "I work at OpenAI with Sam Altman and Greg Brockman. We're based in San Francisco."
Output: [
  {"name": "OpenAI", "type": "organization", "mentions": 1},
  {"name": "Sam Altman", "type": "person", "mentions": 1},
  {"name": "Greg Brockman", "type": "person", "mentions": 1},
  {"name": "San Francisco", "type": "place", "mentions": 1}
]

Input: "I prefer using VS Code over IntelliJ IDEA for Python projects."
Output: [
  {"name": "VS Code", "type": "thing", "mentions": 1},
  {"name": "IntelliJ IDEA", "type": "thing", "mentions": 1}
]

Input: "Let me know if you can help."
Output: []`

var userPromptExtractEntities = `Extract all named entities from this content.

CONTENT:
---
{{.Content}}
---

Return a JSON array of entities with their types and mention counts. If no named entities are present, return [].`

var systemPromptExtractRelations = `You are a relationship extraction system. Identify relationships between named entities.

RELATION TYPES:
- RELATED_TO: general relationship or association
- USES: one entity uses another (tools, technologies)
- DEPENDS_ON: one entity depends on another
- PART_OF: one entity is part of another
- WORKS_WITH: entities that work together
- KNOWS: one entity knows another (people)
- LIKES: one entity likes another (preferences)
- DISLIKES: one entity dislikes another
- OWNS: one entity owns another
- MEMBER_OF: one entity is a member of another
- CREATED_BY: one entity was created by another
- IMPROVES: one entity improves another
- CONFLICTS: entities that conflict or compete

RULES:
- Only identify relations between entities in the provided list
- Use the exact entity name from the list (case-sensitive)
- Only include relations explicitly supported by the content
- Each relation must be directional (from -> to)
- Return ONLY a valid JSON array — no prose, no markdown

FEW-SHOT EXAMPLE:

Entities: [{"name": "OpenAI", "type": "organization"}, {"name": "Sam Altman", "type": "person"}, {"name": "GPT-4", "type": "thing"}]
Content: "I work at OpenAI with Sam Altman developing GPT-4."
Output: [
  {"from": "Sam Altman", "to": "OpenAI", "type": "WORKS_WITH"},
  {"from": "GPT-4", "to": "OpenAI", "type": "CREATED_BY"},
  {"from": "Sam Altman", "to": "GPT-4", "type": "RELATED_TO"}
]`

var userPromptExtractRelations = `Identify relationships between the named entities based on the content.

CONTENT:
---
{{.Content}}
---

ENTITIES:
{{.Entities}}

Return a JSON array of relations. Each relation must have "from", "to", and "type" fields. If no relations exist, return [].`

// systemPromptResolveConflict resolves contradictions between existing and new memory.
var systemPromptResolveConflict = `You are a memory conflict resolution system. When new information contradicts an existing memory, decide how to resolve the conflict.

THINK STEP-BY-STEP:
1. Identify what specifically conflicts between the two memories.
2. Evaluate recency: newer information is generally more reliable.
3. Evaluate specificity: more specific information supersedes generic statements.
4. Evaluate importance: higher-importance memory should be protected.
5. Choose the best resolution action.

RESOLUTION ACTIONS:
- "update": Replace old memory with new content (use when new info is clearly more current/accurate)
- "keep_both": Retain both memories (use when both have distinct, complementary value)
- "discard_new": Reject new information (use when existing memory is more reliable or authoritative)

OUTPUT — return ONLY valid JSON:
{"action": "<update|keep_both|discard_new>", "updated_content": "<merged or updated text if action=update, else empty>", "reason": "<one clear sentence explaining the decision>"}`

var userPromptResolveConflict = `Resolve this memory conflict.

EXISTING MEMORY:
Content: {{.ExistingContent}}
Importance: {{.ExistingImportance}}

NEW INFORMATION:
{{.NewContent}}

Step 1 — What specifically conflicts?
Step 2 — Which is more recent, specific, and reliable?
Step 3 — Choose: update, keep_both, or discard_new.

Return ONLY valid JSON with action, updated_content (if applicable), and reason.`

var extractCategoriesPrompt = `You are a memory categorization system. Assign one or more categories to the following content.

AVAILABLE CATEGORIES:
- preference: User likes, dislikes, habits, or recurring choices
- fact: Objective factual information about the user or world
- decision: Choices or commitments explicitly made by the user
- requirement: Stated needs, constraints, or non-negotiable conditions
- goal: User objectives, targets, or aspirations
- skill: User capabilities, expertise, or areas of knowledge
- personal: Personal biographical information
- work: Work, career, or professional context
- health: Health, medical, or wellness information
- other: Does not fit any category above

CONTENT:
---
{{.Content}}
---

Return ONLY a JSON array of the applicable categories. Use the minimum number of categories that accurately describe the content.
Example: ["preference", "work"]`

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
	pr.templates["extractRelations"] = template.Must(template.New("extractRelations").Parse(userPromptExtractRelations))
	pr.templates["extractSkills"] = template.Must(template.New("extractSkills").Parse(userPromptExtractSkills))
	pr.templates["synthesizeSkills"] = template.Must(template.New("synthesizeSkills").Parse(userPromptSynthesizeSkills))
	pr.templates["suggestProcedure"] = template.Must(template.New("suggestProcedure").Parse(userPromptSuggestProcedure))
	pr.templates["inferProcedure"] = template.Must(template.New("inferProcedure").Parse(userPromptInferProcedure))

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

func (pr *PromptRenderer) RenderExtractRelations(content string, entitiesJSON string) (string, error) {
	var buf bytes.Buffer
	data := struct {
		Content  string
		Entities string
	}{
		Content:  content,
		Entities: entitiesJSON,
	}
	if err := pr.templates["extractRelations"].Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pr *PromptRenderer) GetSystemPromptExtractRelations() string {
	return systemPromptExtractRelations
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
	Enabled              bool
	AutoExtractFacts     bool
	AutoExtractEntities  bool
	AutoExtractRelations bool
	DefaultImportance    string
}

func DefaultConfig() *Config {
	return &Config{
		Enabled:              true,
		AutoExtractFacts:     true,
		AutoExtractEntities:  true,
		AutoExtractRelations: true,
		DefaultImportance:    ImportanceMedium,
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

	pr.templates["extractSkills"] = template.Must(template.New("extractSkills").Parse(userPromptExtractSkills))
	pr.templates["synthesizeSkills"] = template.Must(template.New("synthesizeSkills").Parse(userPromptSynthesizeSkills))
	pr.templates["suggestProcedure"] = template.Must(template.New("suggestProcedure").Parse(userPromptSuggestProcedure))
	pr.templates["inferProcedure"] = template.Must(template.New("inferProcedure").Parse(userPromptInferProcedure))

	return pr
}
