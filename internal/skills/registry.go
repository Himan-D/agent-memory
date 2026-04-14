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
