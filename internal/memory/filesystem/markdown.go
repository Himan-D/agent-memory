package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MarkdownMemory implements Claude's MEMORY.md pattern.
// Stores memories as markdown files organized by topic.
//
// Directory structure:
//
//	/data/memories/{user_id}/
//	├── MEMORY.md           # Index (first N lines loaded per session)
//	├── preferences.md      # Topic file
//	├── decisions.md
//	├── patterns.md
//	└── daily/
//	    └── 2026-05-26.md   # Daily log
type MarkdownMemory struct {
	baseDir string // root directory for memory files
}

// NewMarkdownMemory creates a MarkdownMemory rooted at the given base directory.
func NewMarkdownMemory(baseDir string) *MarkdownMemory {
	return &MarkdownMemory{baseDir: baseDir}
}

// LoadIndex reads the first maxLines lines of MEMORY.md for a user.
// If maxLines <= 0, defaults to 200. Returns empty string if the file doesn't exist.
func (mm *MarkdownMemory) LoadIndex(userID string, maxLines int) (string, error) {
	if maxLines <= 0 {
		maxLines = 200
	}
	indexPath := filepath.Join(mm.userDir(userID), "MEMORY.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("filesystem: load index: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n"), nil
}

// LoadTopic reads a specific topic file on-demand.
// Returns empty string if the topic file doesn't exist.
func (mm *MarkdownMemory) LoadTopic(userID, topic string) (string, error) {
	topicPath := filepath.Join(mm.userDir(userID), mm.sanitizeFilename(topic)+".md")
	data, err := os.ReadFile(topicPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("filesystem: load topic %s: %w", topic, err)
	}
	return string(data), nil
}

// WriteFact appends a fact to the appropriate topic file and updates the MEMORY.md index.
func (mm *MarkdownMemory) WriteFact(userID, topic, fact string) error {
	if err := mm.EnsureUserDir(userID); err != nil {
		return err
	}

	topicFile := mm.sanitizeFilename(topic) + ".md"
	topicPath := filepath.Join(mm.userDir(userID), topicFile)

	// Create or append to topic file
	var content string
	existing, err := os.ReadFile(topicPath)
	if err == nil {
		content = string(existing)
	}

	timestamp := time.Now().Format("2006-01-02 15:04")
	entry := fmt.Sprintf("- [%s] %s\n", timestamp, fact)

	if content == "" {
		// Capitalize first letter of topic for heading
		heading := topic
		if len(heading) > 0 {
			heading = strings.ToUpper(heading[:1]) + heading[1:]
		}
		content = fmt.Sprintf("# %s\n\n%s", heading, entry)
	} else {
		content += entry
	}

	if err := os.WriteFile(topicPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("filesystem: write fact: %w", err)
	}

	// Update MEMORY.md index
	return mm.updateIndex(userID, topic, fact)
}

// WriteDaily appends an entry to today's daily log.
func (mm *MarkdownMemory) WriteDaily(userID, entry string) error {
	if err := mm.EnsureUserDir(userID); err != nil {
		return err
	}

	dailyDir := filepath.Join(mm.userDir(userID), "daily")
	if err := os.MkdirAll(dailyDir, 0755); err != nil {
		return fmt.Errorf("filesystem: write daily: mkdir: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	dailyPath := filepath.Join(dailyDir, today+".md")

	var content string
	existing, err := os.ReadFile(dailyPath)
	if err == nil {
		content = string(existing)
	}

	timestamp := time.Now().Format("15:04")
	logEntry := fmt.Sprintf("- [%s] %s\n", timestamp, entry)

	if content == "" {
		content = fmt.Sprintf("# Daily Log: %s\n\n%s", today, logEntry)
	} else {
		content += logEntry
	}

	if err := os.WriteFile(dailyPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("filesystem: write daily: %w", err)
	}
	return nil
}

// Consolidate merges duplicates and drops stale facts, then rebuilds the MEMORY.md index.
// This is a simplified consolidation that rewrites the index from all topic files.
func (mm *MarkdownMemory) Consolidate(userID string) error {
	topics, err := mm.ListTopics(userID)
	if err != nil {
		return fmt.Errorf("filesystem: consolidate: %w", err)
	}

	var indexLines []string
	indexLines = append(indexLines, "# Memory Index")
	indexLines = append(indexLines, "")
	indexLines = append(indexLines, fmt.Sprintf("Last consolidated: %s", time.Now().Format("2006-01-02 15:04")))
	indexLines = append(indexLines, "")

	for _, topic := range topics {
		content, err := mm.LoadTopic(userID, topic)
		if err != nil {
			continue
		}
		// Count entries (lines starting with "- ")
		lines := strings.Split(content, "\n")
		entryCount := 0
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "- ") {
				entryCount++
			}
		}
		indexLines = append(indexLines, fmt.Sprintf("## %s (%d entries)", topic, entryCount))

		// Include the most recent entries (up to 5) in the index
		var recentEntries []string
		for i := len(lines) - 1; i >= 0 && len(recentEntries) < 5; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "- ") {
				recentEntries = append([]string{trimmed}, recentEntries...)
			}
		}
		for _, e := range recentEntries {
			indexLines = append(indexLines, e)
		}
		indexLines = append(indexLines, "")
	}

	indexPath := filepath.Join(mm.userDir(userID), "MEMORY.md")
	return os.WriteFile(indexPath, []byte(strings.Join(indexLines, "\n")), 0644)
}

// ListTopics returns available topic files for a user (filename without extension).
func (mm *MarkdownMemory) ListTopics(userID string) ([]string, error) {
	dir := mm.userDir(userID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("filesystem: list topics: %w", err)
	}

	var topics []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "MEMORY.md" {
			continue // skip the index file
		}
		if strings.HasSuffix(name, ".md") {
			topics = append(topics, strings.TrimSuffix(name, ".md"))
		}
	}
	if topics == nil {
		return []string{}, nil
	}
	return topics, nil
}

// ExportAll returns all memory files as a map[filename]content.
func (mm *MarkdownMemory) ExportAll(userID string) (map[string]string, error) {
	result := make(map[string]string)
	dir := mm.userDir(userID)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		result[relPath] = string(data)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("filesystem: export all: %w", err)
	}
	return result, nil
}

// EnsureUserDir creates the memory directory structure for a user if it doesn't exist.
func (mm *MarkdownMemory) EnsureUserDir(userID string) error {
	dir := mm.userDir(userID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("filesystem: ensure user dir: %w", err)
	}
	dailyDir := filepath.Join(dir, "daily")
	if err := os.MkdirAll(dailyDir, 0755); err != nil {
		return fmt.Errorf("filesystem: ensure daily dir: %w", err)
	}
	return nil
}

// userDir returns the base directory for a user's memory files.
func (mm *MarkdownMemory) userDir(userID string) string {
	return filepath.Join(mm.baseDir, mm.sanitizeFilename(userID))
}

// sanitizeFilename removes path separators and other unsafe characters from a filename.
func (mm *MarkdownMemory) sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ToLower(name)
	return name
}

// updateIndex appends a brief reference to the MEMORY.md index for a newly written fact.
func (mm *MarkdownMemory) updateIndex(userID, topic, fact string) error {
	indexPath := filepath.Join(mm.userDir(userID), "MEMORY.md")

	var content string
	existing, err := os.ReadFile(indexPath)
	if err == nil {
		content = string(existing)
	}

	if content == "" {
		content = "# Memory Index\n\n"
	}

	timestamp := time.Now().Format("2006-01-02")
	entry := fmt.Sprintf("- [%s] (%s) %s\n", timestamp, topic, truncate(fact, 80))
	content += entry

	return os.WriteFile(indexPath, []byte(content), 0644)
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
