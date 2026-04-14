package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Registry struct {
	mu     sync.RWMutex
	skills map[string]*Skill
	paths  []string
}

type Skill struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Triggers    []string `yaml:"triggers,omitempty"`
	Tools       []string `yaml:"tools,omitempty"`
	Model       string   `yaml:"model,omitempty"`
	MemoryType  string   `yaml:"memory_blocks,omitempty"`
	System      string   `yaml:"-"`
	SourceFile  string   `yaml:"-"`
}

type SkillFile struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Triggers    []string `yaml:"triggers"`
	Tools       []string `yaml:"tools"`
	Model       string   `yaml:"model"`
	MemoryType  string   `yaml:"memory_blocks"`
}

func NewRegistry(skillsPath string) (*Registry, error) {
	r := &Registry{
		skills: make(map[string]*Skill),
		paths:  []string{},
	}

	if skillsPath != "" {
		r.paths = append(r.paths, skillsPath)
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(homeDir, ".agent-memory", "skills")
		r.paths = append(r.paths, globalPath)
	}

	r.paths = append(r.paths, ".skills")

	for _, p := range r.paths {
		if err := r.loadFromPath(p); err != nil {
			continue
		}
	}

	if len(r.skills) == 0 {
		r.loadBuiltInSkills()
	}

	return r, nil
}

func (r *Registry) loadFromPath(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		if err := r.loadSkillFile(filePath); err != nil {
			continue
		}
	}

	return nil
}

func (r *Registry) loadSkillFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	skill, err := ParseSkillFile(content)
	if err != nil {
		return err
	}

	skill.SourceFile = filePath

	r.mu.Lock()
	defer r.mu.Unlock()

	if skill.Name == "" {
		name := strings.TrimSuffix(filepath.Base(filePath), ".md")
		skill.Name = name
	}

	r.skills[skill.Name] = skill
	return nil
}

func ParseSkillFile(content []byte) (*Skill, error) {
	var skill Skill
	var systemPart strings.Builder
	inFrontmatter := false
	frontmatterLines := []string{}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if i == 0 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}

		if inFrontmatter {
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = false
				continue
			}
			frontmatterLines = append(frontmatterLines, line)
			continue
		}

		if inFrontmatter && i == len(lines)-1 {
			inFrontmatter = false
			continue
		}

		if !inFrontmatter && len(frontmatterLines) > 0 {
			frontmatter := strings.Join(frontmatterLines, "\n")
			if err := yaml.Unmarshal([]byte(frontmatter), &skill); err != nil {
				return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
			}
		}

		systemPart.WriteString(line)
		systemPart.WriteString("\n")
	}

	skill.System = strings.TrimSpace(systemPart.String())

	return &skill, nil
}

func (r *Registry) loadBuiltInSkills() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.skills["git-expert"] = &Skill{
		Name:        "git-expert",
		Description: "Expert git workflows, branching strategies, merge conflict resolution",
		Triggers:    []string{"git", "version control", "branching", "merge", "rebase"},
		Tools:       []string{"bash"},
		Model:       "auto",
		System: `You are a git expert. Help with:
- Branching strategies (GitFlow, trunk-based)
- Merge and rebase operations
- Conflict resolution
- Troubleshooting git issues
- Commit message best practices`,
	}

	r.skills["code-review"] = &Skill{
		Name:        "code-review",
		Description: "Review code for bugs, security issues, code quality",
		Triggers:    []string{"review code", "check bugs", "analyze code", "security"},
		Tools:       []string{"Read", "Glob", "Grep"},
		Model:       "auto",
		System: `You are an expert code reviewer. When reviewing code:
1. Check for common bugs and security vulnerabilities
2. Review code style consistency
3. Verify error handling
4. Assess test coverage
5. Provide actionable feedback`,
	}

	r.skills["debugger"] = &Skill{
		Name:        "debugger",
		Description: "Debug code issues, trace bugs, fix errors",
		Triggers:    []string{"debug", "fix", "error", "bug", "issue"},
		Tools:       []string{"Read", "Grep", "bash"},
		Model:       "auto",
		System: `You are a debugging expert. To debug issues:
1. Gather error messages and stack traces
2. Search for relevant code
3. Trace execution flow
4. Identify root cause
5. Propose and implement fixes`,
	}

	r.skills["planner"] = &Skill{
		Name:        "planner",
		Description: "Break down tasks, create implementation plans",
		Triggers:    []string{"plan", "breakdown", "implement", "task"},
		Tools:       []string{"Read", "Glob"},
		Model:       "auto",
		System: `You are a planning expert. When creating plans:
1. Understand the goal and constraints
2. Break into smaller tasks
3. Identify dependencies
4. Estimate complexity
5. Create actionable steps`,
	}

	r.skills["memory-manager"] = &Skill{
		Name:        "memory-manager",
		Description: "Manage different memory types - working, session, semantic, episodic, procedural, business rules",
		Triggers:    []string{"memory", "store", "recall", "remember", "context", "forget", "consolidate"},
		Tools:       []string{"Read", "Grep", "bash", "VectorStore"},
		Model:       "auto",
		System: `You are a memory management expert for AI agents. Manage these memory types:
- Working Memory: Current context window, immediate reasoning state
- Session Memory: Conversation history, rolling summaries
- Semantic Memory: Facts, preferences, knowledge (vector storage)
- Episodic Memory: Past experiences, patterns over time
- Procedural Memory: How-to playbooks, workflows, reusable skills
- Business Rules: Policies, SLAs, constraints

Operations:
- store_memory(type, content, metadata)
- retrieve_memory(query, type, limit)
- consolidate_memory() - move session to long-term
- forget_memory(criteria) - intelligent forgetting
- import_memory(source) - migration support
- export_memory(format)`,
	}

	r.skills["graph-expert"] = &Skill{
		Name:        "graph-expert",
		Description: "Knowledge graph operations - entities, relationships, entity resolution",
		Triggers:    []string{"graph", "entities", "relationships", "knowledge", "entity", "link"},
		Tools:       []string{"Read", "Grep", "bash", "GraphStore"},
		Model:       "auto",
		System: `You are a knowledge graph expert. Operations include:
- Entity Extraction: Identify entities from text
- Entity Creation: CREATE (n:Entity {name, type, properties})
- Relationship Mapping: CREATE (a)-[r:RELATIONSHIP]->(b)
- Entity Resolution: Link duplicate entities
- Graph Traversal: MATCH paths, shortest path
- Query Patterns: Find connections, communities

Graph types: Person, Organization, Location, Event, Concept, Tool, Memory
Relationship types: KNOWS, USES, CREATED, BELONGS_TO, RELATED_TO`,
	}

	r.skills["search-expert"] = &Skill{
		Name:        "search-expert",
		Description: "Semantic search, hybrid search, vector similarity, reranking",
		Triggers:    []string{"search", "find", "query", "semantic", "vector", "similarity", "hybrid"},
		Tools:       []string{"Grep", "Read", "bash", "VectorStore"},
		Model:       "auto",
		System: `You are a search expert. Master these search types:
- Semantic Search: Meaning-based vector similarity search
- Hybrid Search: Combine keyword + vector for best results
- Reranking: Reorder results by relevance
- Multi-modal Search: Text, image, audio search
- Filters: Apply metadata filters (date, type, source)
- Pagination: Efficient large result handling

Best practices:
- Use hybrid search for production
- Add metadata filters for precision
- Implement result diversification
- Cache frequent queries
- Monitor query latency (<100ms target)`,
	}

	r.skills["multi-agent"] = &Skill{
		Name:        "multi-agent",
		Description: "Coordinate multiple agents, shared memory pools, pub/sub messaging",
		Triggers:    []string{"sync", "collaborate", "share", "team", "agent", "coordinate", "delegation"},
		Tools:       []string{"bash", "Read", "Grep"},
		Model:       "auto",
		System: `You are a multi-agent coordination expert. Handle:
- Shared Memory Pools: Agents share context and memories
- Task Delegation: Route tasks to appropriate agents
- Conflict Resolution: Handle competing agent requests
- Pub/Sub Messaging: Real-time agent communication
- Agent Discovery: Find available agents by capability
- State Synchronization: Keep agents aligned

Patterns:
- Leader-Follower: One agent coordinates
- Peer-to-Peer: Agents collaborate equally
- Hierarchical: Nested agent teams
- Debate: Agents propose, then converge`,
	}

	r.skills["integration-expert"] = &Skill{
		Name:        "integration-expert",
		Description: "SDK integrations - LangChain, LangGraph, CrewAI, LlamaIndex, Agno",
		Triggers:    []string{"langchain", "langgraph", "crewai", "llamaindex", "agno", "integration", "framework"},
		Tools:       []string{"Read", "Grep", "bash"},
		Model:       "auto",
		System: `You are an AI framework integration expert. Expert in:

LangChain:
- LCEL (LangChain Expression Language)
- Chains, Agents, Tools
- Memory integration
- Retrievers

LangGraph:
- Stateful agents with graphs
- Checkpointing
- Human-in-the-loop

CrewAI:
- Crews, Agents, Tasks
- Sequential/parallel processes
- Memory sharing

LlamaIndex:
- Data indexing
- Query engines
- Node parsers

Agno:
- Vector memory
- Agent teams
- Tool execution

Provide code examples and best practices for each.`,
	}

	r.skills["analytics-pro"] = &Skill{
		Name:        "analytics-pro",
		Description: "Usage analytics, metrics, dashboard, insights",
		Triggers:    []string{"analytics", "metrics", "stats", "usage", "dashboard", "insights", "monitoring"},
		Tools:       []string{"Read", "Grep", "bash"},
		Model:       "auto",
		System: `You are an analytics expert. Track and analyze:
- Query Analytics: Most common searches, response times
- Memory Analytics: Storage usage, recall rates, consolidation stats
- User Analytics: Active users, session lengths, feature usage
- Cost Analytics: Token usage, API calls, compute costs
- Quality Metrics: Recall, relevance, latency
- Custom Dashboards: Build views for each metric type

Provide actionable insights and recommendations.`,
	}

	r.skills["security-audit"] = &Skill{
		Name:        "security-audit",
		Description: "Audit logs, access control, compliance - SOC2, HIPAA, GDPR",
		Triggers:    []string{"audit", "compliance", "security", "access", "HIPAA", "SOC2", "GDPR", "privacy"},
		Tools:       []string{"Read", "Grep", "bash"},
		Model:       "auto",
		System: `You are a security and compliance expert. Handle:
- Audit Logs: Track all API access, data modifications
- Access Control: RBAC, API key scopes (read/write/admin)
- Compliance: SOC2, HIPAA, GDPR requirements
- Data Encryption: At rest and in transit
- Privacy: PII handling, data retention policies
- Threat Detection: Unusual access patterns

Report generation for compliance audits.`,
	}

	r.skills["migration-pro"] = &Skill{
		Name:        "migration-pro",
		Description: "Import/export data, backup/restore, migration between storage systems",
		Triggers:    []string{"import", "export", "migrate", "backup", "restore", "migration", "bulk"},
		Tools:       []string{"Read", "Grep", "bash"},
		Model:       "auto",
		System: `You are a data migration expert. Handle:
- Data Import: From CSV, JSON, SQL, other databases
- Data Export: Backup, migrate to new systems
- Format Conversion: Between storage backends
- Bulk Operations: Efficient large dataset handling
- Incremental Sync: Delta updates
- Rollback Support: Revert failed migrations
- Validation: Verify data integrity after migration`,
	}

	r.skills["skill-manager"] = &Skill{
		Name:        "skill-manager",
		Description: "Dynamic skill registration, discovery, and management - add new skills at runtime",
		Triggers:    []string{"skill", "register", "new skill", "capability", "extend", "plugin"},
		Tools:       []string{"Read", "Glob", "bash"},
		Model:       "auto",
		System: `You are a skill management expert. Enable dynamic skill system:
- Register Skill: Add new skill with triggers and tools
- Discover Skills: Find skills by trigger keywords
- Skill Dependencies: Manage tool requirements
- Hot Reload: Add skills without restart
- Skill Versioning: Track skill changes
- Compose Skills: Combine skills for complex tasks

Skills are defined in YAML frontmatter + markdown.`,
	}
}

func (r *Registry) ListSkills() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Skill, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, s)
	}
	return result
}

func (r *Registry) GetSkill(name string) *Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if skill, ok := r.skills[name]; ok {
		return skill
	}

	for _, skill := range r.skills {
		if strings.EqualFold(skill.Name, name) {
			return skill
		}
	}

	return nil
}

func (r *Registry) FindByTrigger(trigger string) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Skill
	trigger = strings.ToLower(trigger)

	for _, skill := range r.skills {
		for _, t := range skill.Triggers {
			if strings.Contains(strings.ToLower(t), trigger) {
				result = append(result, skill)
				break
			}
		}
	}

	return result
}

func (r *Registry) Reload() error {
	r.mu.Lock()
	r.skills = make(map[string]*Skill)
	r.mu.Unlock()

	for _, p := range r.paths {
		if err := r.loadFromPath(p); err != nil {
			continue
		}
	}

	if len(r.skills) == 0 {
		r.loadBuiltInSkills()
	}

	return nil
}

func (r *Registry) AddSkill(skill *Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if skill.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	r.skills[skill.Name] = skill
	return nil
}

func (r *Registry) RemoveSkill(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.skills, name)
	return nil
}
