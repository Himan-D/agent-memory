package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chzyer/readline"

	"agent-memory/cmd/agent/commands"
	"agent-memory/cmd/agent/harness"
	"agent-memory/internal/config"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory"
	"agent-memory/internal/skills"
)

var (
	version = "0.1.0"
)

func main() {
	fmt.Printf("🤖 Agent Memory CLI v%s\n", version)
	fmt.Println("Type /help for available commands, /exit to quit")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := loadConfig()

	memSvc, err := memory.NewService(cfg)
	if err != nil {
		fmt.Printf("❌ Failed to connect to memory service: %v\n", err)
		os.Exit(1)
	}
	defer memSvc.Close()

	llmCfg := &llm.Config{
		Provider:     llm.ProviderType(cfg.LLM.Provider),
		APIKey:       cfg.LLM.APIKey,
		BaseURL:      cfg.LLM.BaseURL,
		Organization: cfg.LLM.OrgID,
	}
	llmClient, err := llm.NewProvider(llmCfg)
	if err != nil {
		fmt.Printf("⚠️  LLM not configured: %v\n", err)
		fmt.Println("   Some features may be limited")
	}

	skillRegistry, err := skills.NewRegistry("")
	if err != nil {
		fmt.Printf("⚠️  Failed to load skills: %v", err)
	}

	h := harness.NewAgentHarness(memSvc, llmClient, skillRegistry)
	commands.Register(h)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n👋 Goodbye!")
		cancel()
	}()

	lc, err := createReadline(h)
	if err != nil {
		fmt.Printf("⚠️  Readline init failed: %v\n", err)
		lc, _ = readline.New("")
	}
	defer lc.Close()

	for {
		line, err := lc.Readline()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			continue
		}

		input := line
		if input == "" {
			continue
		}

		if err := h.ProcessInput(ctx, input); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	}
}

func createReadline(h *harness.AgentHarness) (*readline.Instance, error) {
	historyFile := getHistoryFile()

	completer := readline.NewPrefixCompleter(
		readline.PcItem("help",
			readline.PcItem("?")),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("agents"),
		readline.PcItem("switch"),
		readline.PcItem("init"),
		readline.PcItem("remember"),
		readline.PcItem("memory"),
		readline.PcItem("context"),
		readline.PcItem("search"),
		readline.PcItem("model"),
		readline.PcItem("compact"),
		readline.PcItem("skills"),
		readline.PcItem("skill"),
		readline.PcItem("subagents"),
		readline.PcItem("mcp"),
		readline.PcItem("server"),
		readline.PcItem("remote"),
		readline.PcItem("fork"),
		readline.PcItem("clear"),
		readline.PcItem("history"),
		readline.PcItem("shell"),
		readline.PcItem("read"),
		readline.PcItem("write"),
		readline.PcItem("env"),
		readline.PcItem("cd"),
		readline.PcItem("ls"),
		readline.PcItem("grep"),
	)

	return readline.NewEx(&readline.Config{
		Prompt:            "agent> ",
		HistoryFile:       historyFile,
		AutoComplete:      completer,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
		UniqueEditLine:    true,
	})
}

func getHistoryFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	os.MkdirAll(home+"/.agent-memory", 0700)
	return home + "/.agent-memory/.agent_history"
}

func loadConfig() *config.Config {
	cfg := config.Load()

	if os.Getenv("NEO4J_URI") == "" {
		cfg.Neo4j.URI = "bolt://localhost:7687"
	}
	if os.Getenv("NEO4J_USER") == "" {
		cfg.Neo4j.User = "neo4j"
	}
	if os.Getenv("NEO4J_PASSWORD") == "" {
		cfg.Neo4j.Password = "password"
	}

	return cfg
}
