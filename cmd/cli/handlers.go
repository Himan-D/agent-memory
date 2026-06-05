package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func handleInit(url, apiKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot get home directory: %w", err)
	}
	path := home + "/.agent-memory.json"
	cfg := CLIConfig{BaseURL: url, APIKey: apiKey}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	success("Configuration saved to %s", path)
	info("  API URL: %s", url)
	return nil
}

func handleHealth(url string) error {
	resp, err := doRequest("GET", apiURL(url, "/health"), "", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	_ = resp
	success("API server is healthy!")
	return nil
}

func handleListAgents(url, apiKey, format string) error {
	data, err := doRequest("GET", apiURL(url, "/agents"), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleCreateAgent(url, apiKey, format, name, configJSON string) error {
	body := map[string]interface{}{"name": name}
	if configJSON != "" {
		var cfg map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return fmt.Errorf("invalid config JSON: %w", err)
		}
		body["config"] = cfg
	}
	data, err := doRequest("POST", apiURL(url, "/agents"), apiKey, body)
	if err != nil {
		return err
	}
	success("Agent created!")
	return printResult(data, format)
}

func handleGetAgent(url, apiKey, format, id string) error {
	data, err := doRequest("GET", apiURL(url, "/agents/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleUpdateAgent(url, apiKey, format, id, name, configJSON string) error {
	body := map[string]interface{}{}
	if name != "" {
		body["name"] = name
	}
	if configJSON != "" {
		var cfg map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return fmt.Errorf("invalid config JSON: %w", err)
		}
		body["config"] = cfg
	}
	data, err := doRequest("PUT", apiURL(url, "/agents/"+id), apiKey, body)
	if err != nil {
		return err
	}
	success("Agent updated!")
	return printResult(data, format)
}

func handleDeleteAgent(url, apiKey, id string) error {
	_, err := doRequest("DELETE", apiURL(url, "/agents/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	success("Agent %s deleted", id)
	return nil
}

func handleAddMemory(url, apiKey, format, agentID, sessionID, content, memType string) error {
	body := map[string]interface{}{
		"agent_id": agentID,
		"content":  content,
		"type":     memType,
	}
	if sessionID != "" {
		body["session_id"] = sessionID
	}
	data, err := doRequest("POST", apiURL(url, "/memories"), apiKey, body)
	if err != nil {
		return err
	}
	success("Memory added!")
	return printResult(data, format)
}

func handleListMemories(url, apiKey, format, agentID string, limit int) error {
	path := fmt.Sprintf("/memories?agent_id=%s&limit=%d", agentID, limit)
	data, err := doRequest("GET", apiURL(url, path), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleGetMemory(url, apiKey, format, id string) error {
	data, err := doRequest("GET", apiURL(url, "/memories/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleUpdateMemory(url, apiKey, format, id, content, memType string) error {
	body := map[string]interface{}{}
	if content != "" {
		body["content"] = content
	}
	if memType != "" {
		body["type"] = memType
	}
	data, err := doRequest("PUT", apiURL(url, "/memories/"+id), apiKey, body)
	if err != nil {
		return err
	}
	success("Memory updated!")
	return printResult(data, format)
}

func handleDeleteMemory(url, apiKey, id string) error {
	_, err := doRequest("DELETE", apiURL(url, "/memories/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	success("Memory %s deleted", id)
	return nil
}

func handleMemoryStats(url, apiKey, format string) error {
	data, err := doRequest("GET", apiURL(url, "/memories/stats"), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleCreateSession(url, apiKey, format, agentID string) error {
	body := map[string]interface{}{"agent_id": agentID}
	data, err := doRequest("POST", apiURL(url, "/sessions"), apiKey, body)
	if err != nil {
		return err
	}
	success("Session created!")
	return printResult(data, format)
}

func handleListSessions(url, apiKey, format, agentID string) error {
	path := "/sessions?agent_id=" + agentID
	data, err := doRequest("GET", apiURL(url, path), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleGetSession(url, apiKey, format, id string) error {
	data, err := doRequest("GET", apiURL(url, "/sessions/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleDeleteSession(url, apiKey, id string) error {
	_, err := doRequest("DELETE", apiURL(url, "/sessions/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	success("Session %s deleted", id)
	return nil
}

func handleSearch(url, apiKey, format, query string, limit int) error {
	body := map[string]interface{}{"query": query, "limit": limit}
	data, err := doRequest("POST", apiURL(url, "/search"), apiKey, body)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleSearchAdvanced(url, apiKey, format, query, filterType, agentID string, limit int) error {
	body := map[string]interface{}{"query": query, "limit": limit}
	if filterType != "" {
		body["type"] = filterType
	}
	if agentID != "" {
		body["agent_id"] = agentID
	}
	data, err := doRequest("POST", apiURL(url, "/search/advanced"), apiKey, body)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleSearchEnhanced(url, apiKey, format, query string) error {
	data, err := doRequest("GET", apiURL(url, "/search/enhanced?query="+query), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleListSkills(url, apiKey, format string) error {
	data, err := doRequest("GET", apiURL(url, "/skills"), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleCreateSkill(url, apiKey, format, name, trigger, action string) error {
	body := map[string]interface{}{"name": name, "action": action}
	if trigger != "" {
		body["trigger"] = trigger
	}
	data, err := doRequest("POST", apiURL(url, "/skills"), apiKey, body)
	if err != nil {
		return err
	}
	success("Skill created!")
	return printResult(data, format)
}

func handleGetSkill(url, apiKey, format, id string) error {
	data, err := doRequest("GET", apiURL(url, "/skills/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleUpdateSkill(url, apiKey, format, id, name, trigger, action string) error {
	body := map[string]interface{}{}
	if name != "" {
		body["name"] = name
	}
	if trigger != "" {
		body["trigger"] = trigger
	}
	if action != "" {
		body["action"] = action
	}
	data, err := doRequest("PUT", apiURL(url, "/skills/"+id), apiKey, body)
	if err != nil {
		return err
	}
	success("Skill updated!")
	return printResult(data, format)
}

func handleDeleteSkill(url, apiKey, id string) error {
	_, err := doRequest("DELETE", apiURL(url, "/skills/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	success("Skill %s deleted", id)
	return nil
}

func handleExtractSkills(url, apiKey, format, content string) error {
	body := map[string]interface{}{"content": content}
	data, err := doRequest("POST", apiURL(url, "/skills/extract"), apiKey, body)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleSuggestSkills(url, apiKey, format, task string) error {
	body := map[string]interface{}{"task": task}
	data, err := doRequest("POST", apiURL(url, "/skills/suggest"), apiKey, body)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleExecuteSkill(url, apiKey, format, id, contextJSON string) error {
	body := map[string]interface{}{}
	if contextJSON != "" {
		var ctx map[string]interface{}
		if err := json.Unmarshal([]byte(contextJSON), &ctx); err != nil {
			return fmt.Errorf("invalid context JSON: %w", err)
		}
		body["context"] = ctx
	}
	data, err := doRequest("POST", apiURL(url, "/skills/"+id+"/execute"), apiKey, body)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleListGroups(url, apiKey, format string) error {
	data, err := doRequest("GET", apiURL(url, "/groups"), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleCreateGroup(url, apiKey, format, name string) error {
	body := map[string]interface{}{"name": name}
	data, err := doRequest("POST", apiURL(url, "/groups"), apiKey, body)
	if err != nil {
		return err
	}
	success("Group created!")
	return printResult(data, format)
}

func handleGetGroup(url, apiKey, format, id string) error {
	data, err := doRequest("GET", apiURL(url, "/groups/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleDeleteGroup(url, apiKey, id string) error {
	_, err := doRequest("DELETE", apiURL(url, "/groups/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	success("Group %s deleted", id)
	return nil
}

func handleAddAgentToGroup(url, apiKey, format, groupID, agentID, role string) error {
	body := map[string]interface{}{"agent_id": agentID, "role": role}
	data, err := doRequest("POST", apiURL(url, "/groups/"+groupID+"/members"), apiKey, body)
	if err != nil {
		return err
	}
	success("Agent added to group!")
	return printResult(data, format)
}

func handleRemoveAgentFromGroup(url, apiKey, groupID, agentID string) error {
	_, err := doRequest("DELETE", apiURL(url, "/groups/"+groupID+"/members/"+agentID), apiKey, nil)
	if err != nil {
		return err
	}
	success("Agent removed from group %s", groupID)
	return nil
}

func handleListEntities(url, apiKey, format string) error {
	data, err := doRequest("GET", apiURL(url, "/entities"), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleCreateEntity(url, apiKey, format, name, entityType, propertiesJSON string) error {
	body := map[string]interface{}{"name": name, "type": entityType}
	if propertiesJSON != "" {
		var props map[string]interface{}
		if err := json.Unmarshal([]byte(propertiesJSON), &props); err != nil {
			return fmt.Errorf("invalid properties JSON: %w", err)
		}
		body["properties"] = props
	}
	data, err := doRequest("POST", apiURL(url, "/entities"), apiKey, body)
	if err != nil {
		return err
	}
	success("Entity created!")
	return printResult(data, format)
}

func handleGetEntity(url, apiKey, format, id string) error {
	data, err := doRequest("GET", apiURL(url, "/entities/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleUpdateEntity(url, apiKey, format, id, name, entityType, propertiesJSON string) error {
	body := map[string]interface{}{}
	if name != "" {
		body["name"] = name
	}
	if entityType != "" {
		body["type"] = entityType
	}
	if propertiesJSON != "" {
		var props map[string]interface{}
		if err := json.Unmarshal([]byte(propertiesJSON), &props); err != nil {
			return fmt.Errorf("invalid properties JSON: %w", err)
		}
		body["properties"] = props
	}
	data, err := doRequest("PUT", apiURL(url, "/entities/"+id), apiKey, body)
	if err != nil {
		return err
	}
	success("Entity updated!")
	return printResult(data, format)
}

func handleDeleteEntity(url, apiKey, id string) error {
	_, err := doRequest("DELETE", apiURL(url, "/entities/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	success("Entity %s deleted", id)
	return nil
}

func handleLinkEntities(url, apiKey, format, from, to, relation string) error {
	body := map[string]interface{}{"from": from, "to": to, "relation": relation}
	data, err := doRequest("POST", apiURL(url, "/relations"), apiKey, body)
	if err != nil {
		return err
	}
	success("Entities linked!")
	return printResult(data, format)
}

func handleExportBackup(url, apiKey, agentID, outputPath string) error {
	path := fmt.Sprintf("/backup/export?agent_id=%s", agentID)
	data, err := doRequest("GET", apiURL(url, path), apiKey, nil)
	if err != nil {
		return err
	}
	if outputPath == "" {
		outputPath = fmt.Sprintf("backup-%s-%s.json", agentID, time.Now().Format("2006-01-02"))
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}
	success("Backup exported to %s", outputPath)
	return nil
}

func handleImportBackup(url, apiKey, agentID, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(data, &body); err != nil {
		return fmt.Errorf("invalid backup JSON: %w", err)
	}
	_, err = doRequest("POST", apiURL(url, "/backup/import"), apiKey, body)
	if err != nil {
		return err
	}
	success("Backup imported!")
	return nil
}

func handleCompressionStats(url, apiKey, format string) error {
	data, err := doRequest("GET", apiURL(url, "/compression/stats"), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleCompressionMode(url, apiKey, format, mode string) error {
	if mode == "" {
		data, err := doRequest("GET", apiURL(url, "/compression/mode"), apiKey, nil)
		if err != nil {
			return err
		}
		return printResult(data, format)
	}
	data, err := doRequest("PUT", apiURL(url, "/compression/mode"), apiKey, map[string]string{"mode": mode})
	if err != nil {
		return err
	}
	success("Compression mode set to %s", mode)
	return printResult(data, format)
}

func handleTierPolicy(url, apiKey, format, policy string) error {
	if policy == "" {
		data, err := doRequest("GET", apiURL(url, "/tier/policy"), apiKey, nil)
		if err != nil {
			return err
		}
		return printResult(data, format)
	}
	data, err := doRequest("PUT", apiURL(url, "/tier/policy"), apiKey, map[string]string{"policy": policy})
	if err != nil {
		return err
	}
	success("Tier policy set to %s", policy)
	return printResult(data, format)
}

func handleLogin(url, apiKey, format, email, password string) error {
	body := map[string]interface{}{"email": email, "password": password}
	data, err := doRequest("POST", apiURL(url, "/auth/login"), apiKey, body)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleLogout(url, apiKey string) error {
	_, err := doRequest("POST", apiURL(url, "/auth/logout"), apiKey, nil)
	if err != nil {
		return err
	}
	success("Logged out")
	return nil
}

func handleAuthStatus(url, apiKey, format string) error {
	data, err := doRequest("GET", apiURL(url, "/auth/me"), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleListWebhooks(url, apiKey, format string) error {
	data, err := doRequest("GET", apiURL(url, "/webhooks"), apiKey, nil)
	if err != nil {
		return err
	}
	return printResult(data, format)
}

func handleCreateWebhook(baseURL, apiKey, format, hookURL, events string) error {
	body := map[string]interface{}{
		"url":    hookURL,
		"events": strings.Split(events, ","),
	}
	data, err := doRequest("POST", apiURL(baseURL, "/webhooks"), apiKey, body)
	if err != nil {
		return err
	}
	success("Webhook created!")
	return printResult(data, format)
}

func handleDeleteWebhook(url, apiKey, id string) error {
	_, err := doRequest("DELETE", apiURL(url, "/webhooks/"+id), apiKey, nil)
	if err != nil {
		return err
	}
	success("Webhook %s deleted", id)
	return nil
}

func handleDashboard(url string) error {
	dashURL := strings.Replace(url, ":8080", ":3000", 1)
	dashURL = strings.Replace(dashURL, "http://", "https://", 1)
	info("Opening dashboard: %s", dashURL)
	return openBrowser(dashURL)
}

func handleDocs() error {
	info("Opening docs: https://hystersis.ai/docs")
	return openBrowser("https://hystersis.ai/docs")
}

func handleMonitor(url, apiKey, format string, interval int) error {
	info("Monitoring memory stats (interval: %ds)...", interval)
	info("Press Ctrl+C to stop\n")
	for {
		data, err := doRequest("GET", apiURL(url, "/memories/stats"), apiKey, nil)
		if err != nil {
			warn("Error: %v", err)
		} else {
			var stats map[string]interface{}
			if json.Unmarshal(data, &stats) == nil {
				fmt.Print("\033[2J\033[H")
				info("=== Memory Monitor ===")
				for k, v := range stats {
					fmt.Printf("  %s: %v\n", k, v)
				}
				fmt.Println()
			}
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func handleCompletionBash() error {
	fmt.Print(bashCompletion)
	return nil
}

func handleCompletionZsh() error {
	fmt.Print(zshCompletion)
	return nil
}

func openBrowser(url string) error {
	cmds := []string{"xdg-open", "open", "google-chrome", "firefox"}
	for _, cmd := range cmds {
		if _, err := os.Stat("/usr/bin/" + cmd); err == nil {
			// best effort
			go func() {}()

			fmt.Fprintf(os.Stderr, "Opening %s with %s...\n", url, cmd)
			return nil
		}
	}
	warn("No browser command found. Open manually: %s", url)
	return nil
}

const bashCompletion = `#!/bin/bash
_hystersis_completions()
{
  local cur prev words cword
  _init_completion || return
  commands="init health agents memories sessions search skills groups entities backup compression tier auth webhooks dashboard docs monitor completion"
  case $prev in
    hystersis) COMPREPLY=($(compgen -W "$commands" -- "$cur")) ;;
    agents|memories|sessions|search|skills|groups|entities|backup|compression|tier|auth|webhooks|completion)
      sub=$(case $prev in
        agents) echo "list create get update delete" ;;
        memories) echo "add list get update delete stats" ;;
        sessions) echo "create list get delete" ;;
        search) echo "advanced enhanced" ;;
        skills) echo "list create get update delete extract suggest execute" ;;
        groups) echo "list create get delete add-agent remove-agent" ;;
        entities) echo "list create get update delete link" ;;
        backup) echo "export import" ;;
        compression) echo "stats mode" ;;
        tier) echo "policy" ;;
        auth) echo "login logout status" ;;
        webhooks) echo "list create delete" ;;
        completion) echo "bash zsh" ;;
      esac)
      COMPREPLY=($(compgen -W "$sub" -- "$cur"))
      ;;
  esac
}
complete -F _hystersis_completions hystersis
`

const zshCompletion = `#compdef hystersis
_hystersis() {
  local -a commands
  commands=(
    'init:Initialize CLI configuration'
    'health:Check API server health'
    'agents:Manage agents'
    'memories:Manage memories'
    'sessions:Manage sessions'
    'search:Search memories'
    'skills:Manage skills'
    'groups:Manage agent groups'
    'entities:Manage knowledge graph entities'
    'backup:Backup and restore memories'
    'compression:Manage compression engine'
    'tier:Manage memory tier policy'
    'auth:Authentication commands'
    'webhooks:Manage webhooks'
    'dashboard:Open web dashboard'
    'docs:Open documentation'
    'monitor:Monitor memory events'
    'completion:Generate shell completion'
  )
  _describe 'command' commands
}
compdef _hystersis hystersis
`
