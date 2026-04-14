package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/skills"
)

type AgentHarness struct {
	memSvc       *memory.Service
	llm          llm.Provider
	skills       *skills.Registry
	currentAgent string
	currentModel string
	sessionID    string
	history      []string
	workingDir   string
	subagents    map[string]*Subagent
}

type Subagent struct {
	ID       string
	Name     string
	Prompt   string
	Status   string
	ParentID string
}

func NewAgentHarness(memSvc *memory.Service, llmClient llm.Provider, skillRegistry *skills.Registry) *AgentHarness {
	h := &AgentHarness{
		memSvc:       memSvc,
		llm:          llmClient,
		skills:       skillRegistry,
		currentAgent: "default",
		currentModel: "gpt-4o",
		subagents:    make(map[string]*Subagent),
	}
	if dir, err := os.Getwd(); err == nil {
		h.workingDir = dir
	}
	return h
}

func (h *AgentHarness) ProcessInput(ctx context.Context, input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	if strings.HasPrefix(input, "/") {
		return h.handleCommand(ctx, input)
	}

	return h.handleMessage(ctx, input)
}

func (h *AgentHarness) handleMessage(ctx context.Context, input string) error {
	if h.llm == nil {
		return fmt.Errorf("LLM not configured - cannot process messages")
	}

	h.history = append(h.history, fmt.Sprintf("User: %s", input))

	response, err := h.sendToLLM(ctx, input)
	if err != nil {
		return fmt.Errorf("LLM error: %w", err)
	}

	h.history = append(h.history, fmt.Sprintf("Assistant: %s", response))
	fmt.Printf("🤖 %s\n", response)

	if h.memSvc != nil && h.sessionID != "" {
		msg := types.Message{
			SessionID: h.sessionID,
			Role:      "user",
			Content:   input,
		}
		h.memSvc.AddToContext(h.sessionID, msg)
	}

	return nil
}

func (h *AgentHarness) handleCommand(ctx context.Context, input string) error {
	parts := strings.SplitN(input, " ", 2)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch cmd {
	case "/help", "/?":
		return h.showHelp()
	case "/exit", "/quit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	case "/agents":
		return h.listAgents(ctx, args)
	case "/switch":
		return h.switchAgent(args)
	case "/init":
		return h.initMemory(ctx, args)
	case "/remember":
		return h.remember(ctx, args)
	case "/memory", "/context":
		return h.showMemory(ctx)
	case "/search":
		return h.search(ctx, args)
	case "/model":
		return h.switchModel(args)
	case "/compact":
		return h.compact(ctx)
	case "/skills":
		return h.listSkills(ctx)
	case "/skill":
		return h.useSkill(ctx, args)
	case "/subagents":
		return h.listSubagents(ctx)
	case "/mcp":
		return h.manageMCP(ctx, args)
	case "/server":
		return h.startServer(args)
	case "/remote":
		return h.connectRemote(args)
	case "/fork":
		return h.fork(ctx, args)
	case "/clear":
		return h.clearHistory()
	case "/history":
		return h.showHistory()
	case "/shell":
		return h.runShell(ctx, args)
	case "/read":
		return h.readFile(args)
	case "/write":
		return h.writeFile(ctx, args)
	case "/env":
		return h.showEnv(args)
	case "/cd":
		return h.changeDir(args)
	case "/ls":
		return h.listDir(args)
	case "/grep":
		return h.grep(args)
	case "/@":
		return h.processFileRef(ctx, args)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}

	return nil
}

func (h *AgentHarness) showHelp() error {
	fmt.Println(`Available commands:
  /help, /?          Show this help message
  /exit, /quit       Exit the agent
  
  Agent Management:
    /agents           List available agents
    /switch [name]    Switch to a different agent
    /init [prompt]   Initialize memory with context
  
  Memory:
    /remember [text]  Add something to remember
    /memory, /context View current memory state
    /search [query]  Search memories
    /compact          Summarize and compact context
    /@ [path]         Reference file and add to memory
  
  Model:
    /model [name]     Switch to a different model
  
  Skills:
    /skills           List available skills
    /skill [name]     Use a specific skill
  
  Subagents:
    /subagents        List active subagents
    /fork [prompt]    Fork conversation with subagent
  
  Server:
    /server           Start server mode
    /remote [host]    Connect to remote agent
  
  File Operations:
    /read [path]      Read file contents
    /write [path] [content]  Write content to file
    /ls [path]        List directory contents
    /cd [path]        Change directory
    /grep [pattern] [file]  Search in file
  
  System:
    /shell [cmd]      Run shell command
    /env [var]        Show environment variables
    /clear            Clear conversation history
    /history          Show conversation history`)
	return nil
}

func (h *AgentHarness) listAgents(ctx context.Context, args string) error {
	fmt.Println("\n📋 Available Agents:")
	fmt.Println("  • default (current)")
	fmt.Println("  • research")
	fmt.Println("  • coding")
	fmt.Println("  • planning")
	return nil
}

func (h *AgentHarness) switchAgent(name string) error {
	if name == "" {
		return fmt.Errorf("specify agent name: /switch [name]")
	}
	h.currentAgent = name
	fmt.Printf("✓ Switched to agent: %s\n", name)
	return nil
}

func (h *AgentHarness) initMemory(ctx context.Context, prompt string) error {
	if prompt == "" {
		prompt = "Initialize memory for this project"
	}

	fmt.Printf("🔄 Initializing memory: %s\n", prompt)

	session, err := h.memSvc.CreateSession(h.currentAgent, map[string]interface{}{
		"type":   "initialization",
		"prompt": prompt,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	h.sessionID = session.ID
	fmt.Printf("✓ Memory initialized (session: %s)\n", session.ID[:min(8, len(session.ID))])
	return nil
}

func (h *AgentHarness) remember(ctx context.Context, text string) error {
	if text == "" {
		return fmt.Errorf("specify what to remember: /remember [text]")
	}

	mem := &types.Memory{
		Content:    text,
		Type:       types.MemoryTypeUser,
		Importance: types.ImportanceMedium,
		Status:     types.MemoryStatusActive,
	}

	created, err := h.memSvc.CreateMemory(ctx, mem)
	if err != nil {
		return fmt.Errorf("failed to save memory: %w", err)
	}

	display := text
	if len(display) > 50 {
		display = display[:50] + "..."
	}
	fmt.Printf("✓ Remembered: %s (id: %s)\n", display, created.ID[:8])
	return nil
}

func (h *AgentHarness) showMemory(ctx context.Context) error {
	if h.sessionID == "" {
		fmt.Println("No active session. Use /init first.")
		return nil
	}

	messages, err := h.memSvc.GetContext(h.sessionID, 50)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	fmt.Println("\n📝 Current Memory/Context:")
	if len(messages) == 0 {
		fmt.Println("  (empty)")
	}
	for _, msg := range messages {
		role := strings.ToUpper(msg.Role[:1])
		content := msg.Content
		if len(content) > 80 {
			content = content[:80] + "..."
		}
		fmt.Printf("  [%s] %s\n", role, content)
	}
	return nil
}

func (h *AgentHarness) search(ctx context.Context, query string) error {
	if query == "" {
		return fmt.Errorf("specify search query: /search [query]")
	}

	fmt.Printf("🔍 Searching for: %s\n", query)

	results, err := h.memSvc.SearchMemories(ctx, &types.SearchRequest{
		Query: query,
		Limit: 10,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	fmt.Printf("\n📊 Found %d results:\n", len(results))
	for i, r := range results {
		content := r.Text
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		fmt.Printf("  %d. [%.2f] %s\n", i+1, r.Score, content)
	}
	return nil
}

func (h *AgentHarness) switchModel(name string) error {
	if name == "" {
		fmt.Printf("Current model: %s\n", h.currentModel)
		fmt.Println("Available models: gpt-4o, gpt-4o-mini, claude-3-5-sonnet, etc.")
		return nil
	}

	h.currentModel = name
	fmt.Printf("✓ Switched to model: %s\n", name)
	return nil
}

func (h *AgentHarness) compact(ctx context.Context) error {
	fmt.Println("🔄 Compacting context...")

	result, err := h.memSvc.RunCompaction(ctx, h.currentAgent, "")
	if err != nil {
		return fmt.Errorf("compaction failed: %w", err)
	}

	fmt.Printf("✓ Compacted: %d deleted, %d summarized, %d archived\n",
		result.DeletedCount, result.SummarizedCount, result.ArchivedCount)
	return nil
}

func (h *AgentHarness) listSkills(ctx context.Context) error {
	if h.skills == nil {
		fmt.Println("Skills system not initialized")
		return nil
	}

	allSkills := h.skills.ListSkills()
	fmt.Printf("\n🛠️  Available Skills (%d):\n", len(allSkills))
	for _, skill := range allSkills {
		fmt.Printf("  • %s: %s\n", skill.Name, skill.Description)
	}
	return nil
}

func (h *AgentHarness) useSkill(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("specify skill name: /skill [name]")
	}

	skill := h.skills.GetSkill(name)
	if skill == nil {
		return fmt.Errorf("skill not found: %s", name)
	}

	fmt.Printf("🔧 Using skill: %s\n", skill.Name)
	fmt.Printf("   %s\n", skill.Description)
	return nil
}

func (h *AgentHarness) listSubagents(ctx context.Context) error {
	fmt.Println("\n🤖 Active Subagents:")
	fmt.Println("  (no active subagents)")
	return nil
}

func (h *AgentHarness) manageMCP(ctx context.Context, args string) error {
	if args == "" {
		fmt.Println("\n🔌 MCP Server Management:")
		fmt.Println("  /mcp list        - List configured MCP servers")
		fmt.Println("  /mcp add [name]  - Add MCP server")
		fmt.Println("  /mcp remove [n]  - Remove MCP server")
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	action := parts[0]

	switch action {
	case "list":
		fmt.Println("  Configured servers: (none)")
	case "add":
		fmt.Println("  Use /server to start MCP server mode")
	default:
		return fmt.Errorf("unknown MCP action: %s", action)
	}
	return nil
}

func (h *AgentHarness) startServer(args string) error {
	fmt.Println("🚀 Starting MCP server mode...")
	fmt.Println("   Server will listen on stdin/stdout for MCP protocol")
	fmt.Println("   Use /remote to connect to a remote agent")
	return nil
}

func (h *AgentHarness) connectRemote(host string) error {
	if host == "" {
		return fmt.Errorf("specify host: /remote [host:port]")
	}
	fmt.Printf("🔗 Connecting to remote agent at %s...\n", host)
	return nil
}

func (h *AgentHarness) fork(ctx context.Context, prompt string) error {
	if prompt == "" {
		return fmt.Errorf("specify fork prompt: /fork [prompt]")
	}

	fmt.Printf("🍴 Forking conversation: %s\n", prompt)
	fmt.Println("   (Subagent would be spawned here)")
	return nil
}

func (h *AgentHarness) clearHistory() error {
	h.history = []string{}
	fmt.Println("✓ History cleared")
	return nil
}

func (h *AgentHarness) showHistory() error {
	fmt.Println("\n📜 Conversation History:")
	if len(h.history) == 0 {
		fmt.Println("  (empty)")
	}
	for i, entry := range h.history {
		fmt.Printf("  %d. %s\n", i+1, entry)
	}
	return nil
}

func (h *AgentHarness) runShell(ctx context.Context, args string) error {
	if args == "" {
		return fmt.Errorf("specify command: /shell [command]")
	}

	fmt.Printf("🔧 Running shell: %s\n", args)

	cmd := exec.Command("sh", "-c", args)
	cmd.Dir = h.workingDir
	output, err := cmd.CombinedOutput()

	if len(output) > 0 {
		fmt.Print(string(output))
	}
	if err != nil {
		return fmt.Errorf("shell error: %w", err)
	}
	return nil
}

func (h *AgentHarness) readFile(args string) error {
	if args == "" {
		return fmt.Errorf("specify file: /read [path]")
	}

	content, err := os.ReadFile(args)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	fmt.Print(string(content))
	return nil
}

func (h *AgentHarness) writeFile(ctx context.Context, args string) error {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("specify file and content: /write [path] [content]")
	}

	path := parts[0]
	content := parts[1]

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	fmt.Printf("✓ Written to %s\n", path)
	return nil
}

func (h *AgentHarness) showEnv(args string) error {
	if args == "" {
		for _, env := range os.Environ() {
			fmt.Println(env)
		}
		return nil
	}

	value := os.Getenv(args)
	if value == "" {
		fmt.Printf("%s: (not set)\n", args)
	} else {
		fmt.Printf("%s=%s\n", args, value)
	}
	return nil
}

func (h *AgentHarness) changeDir(args string) error {
	if args == "" {
		home, _ := os.UserHomeDir()
		args = home
	}

	if err := os.Chdir(args); err != nil {
		return fmt.Errorf("cd error: %w", err)
	}

	h.workingDir, _ = os.Getwd()
	fmt.Printf("✓ Changed directory to: %s\n", h.workingDir)
	return nil
}

func (h *AgentHarness) listDir(args string) error {
	path := args
	if path == "" {
		path = h.workingDir
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("ls error: %w", err)
	}

	for _, entry := range entries {
		info, _ := entry.Info()
		size := ""
		if info != nil && !entry.IsDir() {
			size = fmt.Sprintf(" %d", info.Size())
		}
		if entry.IsDir() {
			fmt.Printf("  📁 %s/\n", entry.Name())
		} else {
			fmt.Printf("  📄 %s%s\n", entry.Name(), size)
		}
	}
	return nil
}

func (h *AgentHarness) grep(args string) error {
	if args == "" {
		return fmt.Errorf("specify pattern: /grep [pattern] [file]")
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("specify pattern and file: /grep [pattern] [file]")
	}

	pattern := parts[0]
	filePath := parts[1]

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("grep error: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	found := false
	for i, line := range lines {
		if re.MatchString(line) {
			fmt.Printf("  %d: %s\n", i+1, line)
			found = true
		}
	}

	if !found {
		fmt.Println("  (no matches)")
	}
	return nil
}

func (h *AgentHarness) processFileRef(ctx context.Context, args string) error {
	if args == "" {
		return fmt.Errorf("specify file: /@ [path]")
	}

	content, err := os.ReadFile(args)
	if err != nil {
		return fmt.Errorf("error reading %s: %w", args, err)
	}

	fmt.Printf("📎 File: %s\n---\n%s\n---\n", args, string(content))

	if h.sessionID != "" && h.memSvc != nil {
		mem := &types.Memory{
			Content:    fmt.Sprintf("Referenced file %s:\n%s", args, string(content)),
			Type:       types.MemoryTypeUser,
			Importance: types.ImportanceMedium,
			Status:     types.MemoryStatusActive,
		}
		h.memSvc.CreateMemory(ctx, mem)
	}

	return nil
}

func (h *AgentHarness) sendToLLM(ctx context.Context, input string) (string, error) {
	if h.llm == nil {
		return "", fmt.Errorf("LLM not configured")
	}

	resp, err := h.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a helpful AI assistant with memory capabilities. Be concise and helpful."},
			{Role: "user", Content: input},
		},
		Model:       h.currentModel,
		MaxTokens:   2000,
		Temperature: 0.7,
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

func (h *AgentHarness) GetSessionID() string {
	return h.sessionID
}

func (h *AgentHarness) SetSessionID(id string) {
	h.sessionID = id
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
