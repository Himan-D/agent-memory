package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

var (
	version = "0.1.0"
	baseURL = "http://localhost:8080"
	apiKey  = ""
)

type Config struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

func main() {
	godotenv.Load()

	if baseURLEnv := os.Getenv("AGENT_MEMORY_URL"); baseURLEnv != "" {
		baseURL = baseURLEnv
	}
	if apiKeyEnv := os.Getenv("AGENT_MEMORY_API_KEY"); apiKeyEnv != "" {
		apiKey = apiKeyEnv
	}

	app := &cli.App{
		Name:    "agent-memory",
		Version: version,
		Usage:   "CLI for Agent Memory - Persistent memory for AI agents",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "Base URL of the Agent Memory API",
				EnvVars: []string{"AGENT_MEMORY_URL"},
				Value:   baseURL,
			},
			&cli.StringFlag{
				Name:    "api-key",
				Aliases: []string{"k"},
				Usage:   "API key for authentication",
				EnvVars: []string{"AGENT_MEMORY_API_KEY"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Initialize Agent Memory CLI configuration",
				Action: func(c *cli.Context) error {
					return initConfig(c.String("url"), c.String("api-key"))
				},
			},
			{
				Name:  "health",
				Usage: "Check if the API is healthy",
				Action: func(c *cli.Context) error {
					return healthCheck(c.String("url"))
				},
			},
			{
				Name:  "agents",
				Usage: "Manage agents",
				Subcommands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List all agents",
						Action: func(c *cli.Context) error {
							return listAgents(c.String("url"), c.String("api-key"))
						},
					},
					{
						Name:  "create",
						Usage: "Create a new agent",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true},
							&cli.StringFlag{Name: "config", Aliases: []string{"c"}},
						},
						Action: func(c *cli.Context) error {
							return createAgent(c.String("url"), c.String("api-key"), c.String("name"), c.String("config"))
						},
					},
					{
						Name:  "get",
						Usage: "Get an agent by ID",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return getAgent(c.String("url"), c.String("api-key"), c.String("id"))
						},
					},
					{
						Name:  "delete",
						Usage: "Delete an agent by ID",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return deleteAgent(c.String("url"), c.String("api-key"), c.String("id"))
						},
					},
				},
			},
			{
				Name:  "memories",
				Usage: "Manage memories",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add a new memory",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
							&cli.StringFlag{Name: "session-id", Aliases: []string{"s"}},
							&cli.StringFlag{Name: "content", Aliases: []string{"c"}, Required: true},
							&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Value: "conversation"},
						},
						Action: func(c *cli.Context) error {
							return addMemory(c.String("url"), c.String("api-key"), c.String("agent-id"), c.String("session-id"), c.String("content"), c.String("type"))
						},
					},
					{
						Name:  "search",
						Usage: "Search memories semantically",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "query", Aliases: []string{"q"}, Required: true},
							&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 10},
						},
						Action: func(c *cli.Context) error {
							return searchMemories(c.String("url"), c.String("api-key"), c.String("query"), c.Int("limit"))
						},
					},
					{
						Name:  "list",
						Usage: "List memories for an agent",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
							&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 50},
						},
						Action: func(c *cli.Context) error {
							return listMemories(c.String("url"), c.String("api-key"), c.String("agent-id"), c.Int("limit"))
						},
					},
					{
						Name:  "delete",
						Usage: "Delete a memory",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return deleteMemory(c.String("url"), c.String("api-key"), c.String("id"))
						},
					},
				},
			},
			{
				Name:  "sessions",
				Usage: "Manage sessions",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create a new session",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return createSession(c.String("url"), c.String("api-key"), c.String("agent-id"))
						},
					},
					{
						Name:  "list",
						Usage: "List sessions for an agent",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return listSessions(c.String("url"), c.String("api-key"), c.String("agent-id"))
						},
					},
				},
			},
			{
				Name:  "skills",
				Usage: "Manage skills (procedural memory)",
				Subcommands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List all skills",
						Action: func(c *cli.Context) error {
							return listSkills(c.String("url"), c.String("api-key"))
						},
					},
					{
						Name:  "create",
						Usage: "Create a new skill",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true},
							&cli.StringFlag{Name: "description", Aliases: []string{"d"}},
							&cli.StringFlag{Name: "trigger", Aliases: []string{"t"}},
							&cli.StringFlag{Name: "action", Aliases: []string{"a"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return createSkill(c.String("url"), c.String("api-key"), c.String("name"), c.String("description"), c.String("trigger"), c.String("action"))
						},
					},
					{
						Name:  "extract",
						Usage: "Extract skills from agent interaction",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
							&cli.StringFlag{Name: "interaction", Aliases: []string{"i"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return extractSkills(c.String("url"), c.String("api-key"), c.String("agent-id"), c.String("interaction"))
						},
					},
					{
						Name:  "suggest",
						Usage: "Suggest skills for a task",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "task", Aliases: []string{"t"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return suggestSkills(c.String("url"), c.String("api-key"), c.String("task"))
						},
					},
				},
			},
			{
				Name:  "groups",
				Usage: "Manage agent groups (multi-agent)",
				Subcommands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List all groups",
						Action: func(c *cli.Context) error {
							return listGroups(c.String("url"), c.String("api-key"))
						},
					},
					{
						Name:  "create",
						Usage: "Create a new group",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return createGroup(c.String("url"), c.String("api-key"), c.String("name"))
						},
					},
					{
						Name:  "add-agent",
						Usage: "Add an agent to a group",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "group-id", Aliases: []string{"g"}, Required: true},
							&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
							&cli.StringFlag{Name: "role", Aliases: []string{"r"}, Value: "member"},
						},
						Action: func(c *cli.Context) error {
							return addAgentToGroup(c.String("url"), c.String("api-key"), c.String("group-id"), c.String("agent-id"), c.String("role"))
						},
					},
				},
			},
			{
				Name:  "entities",
				Usage: "Manage knowledge graph entities",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create an entity",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true},
							&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Required: true},
							&cli.StringFlag{Name: "properties", Aliases: []string{"p"}},
						},
						Action: func(c *cli.Context) error {
							return createEntity(c.String("url"), c.String("api-key"), c.String("name"), c.String("type"), c.String("properties"))
						},
					},
					{
						Name:  "get",
						Usage: "Get an entity",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return getEntity(c.String("url"), c.String("api-key"), c.String("name"))
						},
					},
					{
						Name:  "link",
						Usage: "Link two entities",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "from", Aliases: []string{"f"}, Required: true},
							&cli.StringFlag{Name: "to", Aliases: []string{"t"}, Required: true},
							&cli.StringFlag{Name: "relation", Aliases: []string{"r"}, Required: true},
						},
						Action: func(c *cli.Context) error {
							return linkEntities(c.String("url"), c.String("api-key"), c.String("from"), c.String("to"), c.String("relation"))
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initConfig(url, apiKeyVal string) error {
	if url == "" {
		url = "http://localhost:8080"
	}

	config := Config{
		BaseURL: url,
		APIKey:  apiKeyVal,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := homeDir + "/.agent-memory.json"
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Configuration saved to %s\n", configPath)
	fmt.Printf("Base URL: %s\n", url)
	return nil
}

func healthCheck(url string) error {
	resp, err := http.Get(url + "/health")
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	fmt.Println("Agent Memory is healthy!")
	return nil
}

func doRequest(method, url, apiKeyVal string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKeyVal != "" {
		req.Header.Set("X-API-Key", apiKeyVal)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func printJSON(data []byte) error {
	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "  "); err != nil {
		fmt.Println(string(data))
		return nil
	}
	fmt.Println(out.String())
	return nil
}

func listAgents(url, apiKeyVal string) error {
	resp, err := doRequest("GET", url+"/api/v1/agents", apiKeyVal, nil)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func createAgent(url, apiKeyVal, name, configJSON string) error {
	body := map[string]interface{}{
		"name": name,
	}
	if configJSON != "" {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			return fmt.Errorf("invalid JSON in config: %w", err)
		}
		body["config"] = config
	}

	resp, err := doRequest("POST", url+"/api/v1/agents", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func getAgent(url, apiKeyVal, id string) error {
	resp, err := doRequest("GET", url+"/api/v1/agents/"+id, apiKeyVal, nil)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func deleteAgent(url, apiKeyVal, id string) error {
	_, err := doRequest("DELETE", url+"/api/v1/agents/"+id, apiKeyVal, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Agent %s deleted successfully\n", id)
	return nil
}

func addMemory(url, apiKeyVal, agentID, sessionID, content, memType string) error {
	body := map[string]interface{}{
		"agent_id": agentID,
		"content":  content,
		"type":     memType,
	}
	if sessionID != "" {
		body["session_id"] = sessionID
	}

	resp, err := doRequest("POST", url+"/api/v1/memories", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func searchMemories(url, apiKeyVal, query string, limit int) error {
	body := map[string]interface{}{
		"query": query,
		"limit": limit,
	}

	resp, err := doRequest("POST", url+"/api/v1/memories/search", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func listMemories(url, apiKeyVal, agentID string, limit int) error {
	resp, err := doRequest("GET", fmt.Sprintf("%s/api/v1/agents/%s/memories?limit=%d", url, agentID, limit), apiKeyVal, nil)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func deleteMemory(url, apiKeyVal, id string) error {
	_, err := doRequest("DELETE", url+"/api/v1/memories/"+id, apiKeyVal, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Memory %s deleted successfully\n", id)
	return nil
}

func createSession(url, apiKeyVal, agentID string) error {
	body := map[string]interface{}{
		"agent_id": agentID,
	}

	resp, err := doRequest("POST", url+"/api/v1/sessions", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func listSessions(url, apiKeyVal, agentID string) error {
	resp, err := doRequest("GET", url+"/api/v1/agents/"+agentID+"/sessions", apiKeyVal, nil)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func listSkills(url, apiKeyVal string) error {
	resp, err := doRequest("GET", url+"/api/v1/skills", apiKeyVal, nil)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func createSkill(url, apiKeyVal, name, description, trigger, action string) error {
	body := map[string]interface{}{
		"name":   name,
		"action": action,
	}
	if description != "" {
		body["description"] = description
	}
	if trigger != "" {
		body["trigger"] = trigger
	}

	resp, err := doRequest("POST", url+"/api/v1/skills", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func extractSkills(url, apiKeyVal, agentID, interaction string) error {
	body := map[string]interface{}{
		"agent_id":    agentID,
		"interaction": interaction,
	}

	resp, err := doRequest("POST", url+"/api/v1/skills/extract", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func suggestSkills(url, apiKeyVal, task string) error {
	body := map[string]interface{}{
		"task": task,
	}

	resp, err := doRequest("POST", url+"/api/v1/skills/suggest", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func listGroups(url, apiKeyVal string) error {
	resp, err := doRequest("GET", url+"/api/v1/groups", apiKeyVal, nil)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func createGroup(url, apiKeyVal, name string) error {
	body := map[string]interface{}{
		"name": name,
	}

	resp, err := doRequest("POST", url+"/api/v1/groups", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func addAgentToGroup(url, apiKeyVal, groupID, agentID, role string) error {
	body := map[string]interface{}{
		"agent_id": agentID,
		"role":     role,
	}

	resp, err := doRequest("POST", url+"/api/v1/groups/"+groupID+"/members", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func createEntity(url, apiKeyVal, name, entityType, propertiesJSON string) error {
	body := map[string]interface{}{
		"name": name,
		"type": entityType,
	}
	if propertiesJSON != "" {
		var properties map[string]interface{}
		if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
			return fmt.Errorf("invalid JSON in properties: %w", err)
		}
		body["properties"] = properties
	}

	resp, err := doRequest("POST", url+"/api/v1/entities", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func getEntity(url, apiKeyVal, name string) error {
	resp, err := doRequest("GET", url+"/api/v1/entities/"+strings.ReplaceAll(name, "/", "%2F"), apiKeyVal, nil)
	if err != nil {
		return err
	}
	return printJSON(resp)
}

func linkEntities(url, apiKeyVal, from, to, relation string) error {
	body := map[string]interface{}{
		"from":     from,
		"to":       to,
		"relation": relation,
	}

	resp, err := doRequest("POST", url+"/api/v1/entities/link", apiKeyVal, body)
	if err != nil {
		return err
	}
	return printJSON(resp)
}
