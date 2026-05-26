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
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v3"
)

type CLIConfig struct {
	BaseURL string `json:"base_url" yaml:"base_url"`
	APIKey  string `json:"api_key" yaml:"api_key"`
}

var configPaths = []string{
	".agent-memory.json",
	".hystersis/config.json",
}

func loadConfig() *CLIConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	for _, p := range configPaths {
		data, err := os.ReadFile(home + "/" + p)
		if err != nil {
			continue
		}
		var cfg CLIConfig
		if strings.HasSuffix(p, ".yaml") || strings.HasSuffix(p, ".yml") {
			if yaml.Unmarshal(data, &cfg) == nil {
				return &cfg
			}
		} else {
			if json.Unmarshal(data, &cfg) == nil {
				return &cfg
			}
		}
	}
	return nil
}

func doRequest(method, url, apiKey string, body interface{}) ([]byte, error) {
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
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
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

func apiURL(base, path string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

func printResult(data []byte, format string) error {
	switch format {
	case "table":
		return printTable(data)
	case "yaml":
		return printYAML(data)
	default:
		return printJSON(data)
	}
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

func printYAML(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Println(string(data))
		return nil
	}
	out, err := yaml.Marshal(v)
	if err != nil {
		fmt.Println(string(data))
		return nil
	}
	fmt.Print(string(out))
	return nil
}

func printTable(data []byte) error {
	var list []interface{}
	if err := json.Unmarshal(data, &list); err != nil {
		var obj map[string]interface{}
		if err2 := json.Unmarshal(data, &obj); err2 != nil {
			return printJSON(data)
		}
		list = []interface{}{obj}
	}

	if len(list) == 0 {
		fmt.Println("(no results)")
		return nil
	}

	keys := collectKeys(list)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	var headerParts []string
	for _, k := range keys {
		headerParts = append(headerParts, strings.ToUpper(k))
	}
	fmt.Fprintln(w, strings.Join(headerParts, "\t"))

	sepParts := make([]string, len(keys))
	for i := range sepParts {
		sepParts[i] = "---"
	}
	fmt.Fprintln(w, strings.Join(sepParts, "\t"))

	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		var row []string
		for _, k := range keys {
			row = append(row, fmt.Sprintf("%v", m[k]))
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
	return nil
}

func collectKeys(list []interface{}) []string {
	seen := map[string]bool{}
	var keys []string
	preferred := []string{"id", "_id", "name", "title", "type", "status", "created_at", "updated_at"}
	for _, p := range preferred {
		seen[p] = true
	}
	for _, p := range preferred {
		keys = append(keys, p)
	}
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		for k := range m {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	var filtered []string
	for _, k := range keys {
		if seen[k] {
			delete(seen, k)
			filtered = append(filtered, k)
		}
	}
	return filtered
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func success(format string, args ...interface{}) {
	fmt.Printf(colorGreen+format+colorReset+"\n", args...)
}

func info(format string, args ...interface{}) {
	fmt.Printf(colorBlue+format+colorReset+"\n", args...)
}

func warn(format string, args ...interface{}) {
	fmt.Printf(colorYellow+format+colorReset+"\n", args...)
}

func errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, colorRed+format+colorReset+"\n", args...)
}
