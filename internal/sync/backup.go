package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"agent-memory/internal/memory"
)

type BackupScheduler struct {
	memoryService *memory.Service
	interval      time.Duration
	backupDir     string
	maxBackups    int
	done          chan struct{}
}

func NewBackupScheduler(m *memory.Service, interval time.Duration, backupDir string, maxBackups int) *BackupScheduler {
	return &BackupScheduler{
		memoryService: m,
		interval:      interval,
		backupDir:     backupDir,
		maxBackups:    maxBackups,
		done:          make(chan struct{}),
	}
}

func (s *BackupScheduler) Start(ctx context.Context) {
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		fmt.Printf("backup scheduler: failed to create backup dir: %v\n", err)
		return
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	fmt.Printf("backup scheduler: started with interval=%v dir=%s max=%d\n", s.interval, s.backupDir, s.maxBackups)

	s.runBackup(ctx)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("backup scheduler: stopped by context")
			return
		case <-s.done:
			fmt.Println("backup scheduler: stopped")
			return
		case <-ticker.C:
			if err := s.runBackup(ctx); err != nil {
				fmt.Printf("backup scheduler: error: %v\n", err)
			}
		}
	}
}

func (s *BackupScheduler) Stop() {
	close(s.done)
}

func (s *BackupScheduler) runBackup(ctx context.Context) error {
	fmt.Println("backup scheduler: running backup...")

	agents, err := s.listAgents(ctx)
	if err != nil {
		return fmt.Errorf("list agents: %w", err)
	}

	if len(agents) == 0 {
		fmt.Println("backup scheduler: no agents to backup")
		return nil
	}

	backupCount := 0
	for _, agent := range agents {
		export, err := s.memoryService.ExportMemories(ctx, agent.ID, agent.OrgID)
		if err != nil {
			fmt.Printf("backup scheduler: failed to export agent %s: %v\n", agent.ID, err)
			continue
		}

		filename := fmt.Sprintf("backup-%s-%s.json", agent.ID, time.Now().Format("2006-01-02-15-04-05"))
		filePath := filepath.Join(s.backupDir, filename)

		data, err := json.MarshalIndent(export, "", "  ")
		if err != nil {
			fmt.Printf("backup scheduler: failed to marshal backup for agent %s: %v\n", agent.ID, err)
			continue
		}

		if err := os.WriteFile(filePath, data, 0644); err != nil {
			fmt.Printf("backup scheduler: failed to write backup file: %v\n", err)
			continue
		}

		fmt.Printf("backup scheduler: backed up agent %s to %s\n", agent.ID, filename)
		backupCount++
	}

	if err := s.rotateBackups(); err != nil {
		fmt.Printf("backup scheduler: failed to rotate backups: %v\n", err)
	}

	fmt.Printf("backup scheduler: completed %d backups\n", backupCount)
	return nil
}

func (s *BackupScheduler) rotateBackups() error {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return fmt.Errorf("read backup dir: %w", err)
	}

	byAgent := make(map[string][]backupInfo)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		var info backupInfo
		info.path = filepath.Join(s.backupDir, entry.Name())

		t, err := time.Parse("backup-*-2006-01-02-15-04-05.json", entry.Name())
		if err != nil {
			continue
		}
		info.time = t

		parts := splitFilename(entry.Name())
		if len(parts) >= 2 {
			info.agentID = parts[1]
		}

		byAgent[info.agentID] = append(byAgent[info.agentID], info)
	}

	for _, backups := range byAgent {
		if len(backups) <= s.maxBackups {
			continue
		}

		sortBackupsByTime(backups)
		toDelete := backups[s.maxBackups:]
		for _, b := range toDelete {
			if err := os.Remove(b.path); err != nil {
				fmt.Printf("backup scheduler: failed to remove old backup %s: %v\n", b.path, err)
			} else {
				fmt.Printf("backup scheduler: removed old backup %s\n", b.path)
			}
		}
	}

	return nil
}

func (s *BackupScheduler) listAgents(ctx context.Context) ([]AgentInfo, error) {
	type agentResult struct {
		ID    string `json:"id"`
		OrgID string `json:"org_id"`
	}

	results, err := s.memoryService.QueryGraph(
		`MATCH (a:Agent) RETURN a.id AS id, a.org_id AS org_id LIMIT 100`,
		map[string]interface{}{},
	)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}

	agents := make([]AgentInfo, 0, len(results))
	for _, r := range results {
		agents = append(agents, AgentInfo{
			ID:    r["id"].(string),
			OrgID: r["org_id"].(string),
		})
	}

	return agents, nil
}

type AgentInfo struct {
	ID    string
	OrgID string
}

type backupInfo struct {
	path    string
	agentID string
	time    time.Time
}

type byTime []struct {
	path    string
	agentID string
	time    time.Time
}

func splitFilename(name string) []string {
	parts := make([]string, 0)
	current := ""
	for _, c := range name {
		if c == '-' || c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func sortBackupsByTime(backups []backupInfo) {
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[j].time.Before(backups[i].time) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}
}
