package main

import "github.com/urfave/cli/v2"

func commands() []*cli.Command {
	return []*cli.Command{
		initCmd(),
		healthCmd(),
		agentsCmd(),
		memoriesCmd(),
		sessionsCmd(),
		searchCmd(),
		skillsCmd(),
		groupsCmd(),
		entitiesCmd(),
		backupCmd(),
		compressionCmd(),
		tierCmd(),
		authCmd(),
		webhooksCmd(),
		dashboardCmd(),
		docsCmd(),
		monitorCmd(),
		completionCmd(),
	}
}

func initCmd() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize CLI configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "url", Aliases: []string{"u"}, Usage: "API server URL", Value: "http://localhost:8080"},
			&cli.StringFlag{Name: "api-key", Aliases: []string{"k"}, Usage: "API key"},
		},
		Action: func(c *cli.Context) error {
			return handleInit(c.String("url"), c.String("api-key"))
		},
	}
}

func healthCmd() *cli.Command {
	return &cli.Command{
		Name:  "health",
		Usage: "Check API server health",
		Action: func(c *cli.Context) error {
			return handleHealth(c.String("url"))
		},
	}
}

func agentsCmd() *cli.Command {
	return &cli.Command{
		Name:  "agents",
		Usage: "Manage agents",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List all agents",
				Action: func(c *cli.Context) error {
					return handleListAgents(c.String("url"), c.String("api-key"), c.String("format"))
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
					return handleCreateAgent(c.String("url"), c.String("api-key"), c.String("format"), c.String("name"), c.String("config"))
				},
			},
			{
				Name:  "get",
				Usage: "Get agent by ID",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleGetAgent(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"))
				},
			},
			{
				Name:  "update",
				Usage: "Update an agent",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}},
					&cli.StringFlag{Name: "config", Aliases: []string{"c"}},
				},
				Action: func(c *cli.Context) error {
					return handleUpdateAgent(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"), c.String("name"), c.String("config"))
				},
			},
			{
				Name:  "delete",
				Usage: "Delete an agent",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleDeleteAgent(c.String("url"), c.String("api-key"), c.String("id"))
				},
			},
		},
	}
}

func memoriesCmd() *cli.Command {
	return &cli.Command{
		Name:  "memories",
		Usage: "Manage memories",
		Subcommands: []*cli.Command{
			{
				Name:  "add",
				Usage: "Add a memory",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
					&cli.StringFlag{Name: "session-id", Aliases: []string{"s"}},
					&cli.StringFlag{Name: "content", Aliases: []string{"c"}, Required: true},
					&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Value: "conversation"},
				},
				Action: func(c *cli.Context) error {
					return handleAddMemory(c.String("url"), c.String("api-key"), c.String("format"), c.String("agent-id"), c.String("session-id"), c.String("content"), c.String("type"))
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
					return handleListMemories(c.String("url"), c.String("api-key"), c.String("format"), c.String("agent-id"), c.Int("limit"))
				},
			},
			{
				Name:  "get",
				Usage: "Get a memory by ID",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleGetMemory(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"))
				},
			},
			{
				Name:  "update",
				Usage: "Update a memory",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
					&cli.StringFlag{Name: "content", Aliases: []string{"c"}},
					&cli.StringFlag{Name: "type", Aliases: []string{"t"}},
				},
				Action: func(c *cli.Context) error {
					return handleUpdateMemory(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"), c.String("content"), c.String("type"))
				},
			},
			{
				Name:  "delete",
				Usage: "Delete a memory",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleDeleteMemory(c.String("url"), c.String("api-key"), c.String("id"))
				},
			},
			{
				Name:  "stats",
				Usage: "Get memory statistics",
				Action: func(c *cli.Context) error {
					return handleMemoryStats(c.String("url"), c.String("api-key"), c.String("format"))
				},
			},
		},
	}
}

func sessionsCmd() *cli.Command {
	return &cli.Command{
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
					return handleCreateSession(c.String("url"), c.String("api-key"), c.String("format"), c.String("agent-id"))
				},
			},
			{
				Name:  "list",
				Usage: "List sessions for an agent",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleListSessions(c.String("url"), c.String("api-key"), c.String("format"), c.String("agent-id"))
				},
			},
			{
				Name:  "get",
				Usage: "Get a session by ID",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleGetSession(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"))
				},
			},
			{
				Name:  "delete",
				Usage: "Delete a session",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleDeleteSession(c.String("url"), c.String("api-key"), c.String("id"))
				},
			},
		},
	}
}

func searchCmd() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search memories",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "query", Aliases: []string{"q"}, Required: true},
			&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 10},
		},
		Subcommands: []*cli.Command{
			{
				Name:  "advanced",
				Usage: "Advanced search with filters",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "query", Aliases: []string{"q"}, Required: true},
					&cli.StringFlag{Name: "type", Aliases: []string{"t"}},
					&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}},
					&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 10},
				},
				Action: func(c *cli.Context) error {
					return handleSearchAdvanced(c.String("url"), c.String("api-key"), c.String("format"),
						c.String("query"), c.String("type"), c.String("agent-id"), c.Int("limit"))
				},
			},
			{
				Name:  "enhanced",
				Usage: "Enhanced search with spreading activation",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "query", Aliases: []string{"q"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleSearchEnhanced(c.String("url"), c.String("api-key"), c.String("format"), c.String("query"))
				},
			},
		},
		Action: func(c *cli.Context) error {
			return handleSearch(c.String("url"), c.String("api-key"), c.String("format"), c.String("query"), c.Int("limit"))
		},
	}
}

func skillsCmd() *cli.Command {
	return &cli.Command{
		Name:  "skills",
		Usage: "Manage skills (procedural memory)",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List all skills",
				Action: func(c *cli.Context) error {
					return handleListSkills(c.String("url"), c.String("api-key"), c.String("format"))
				},
			},
			{
				Name:  "create",
				Usage: "Create a skill",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true},
					&cli.StringFlag{Name: "trigger", Aliases: []string{"t"}},
					&cli.StringFlag{Name: "action", Aliases: []string{"a"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleCreateSkill(c.String("url"), c.String("api-key"), c.String("format"), c.String("name"), c.String("trigger"), c.String("action"))
				},
			},
			{
				Name:  "get",
				Usage: "Get skill by ID",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleGetSkill(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"))
				},
			},
			{
				Name:  "update",
				Usage: "Update a skill",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}},
					&cli.StringFlag{Name: "trigger", Aliases: []string{"t"}},
					&cli.StringFlag{Name: "action", Aliases: []string{"a"}},
				},
				Action: func(c *cli.Context) error {
					return handleUpdateSkill(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"), c.String("name"), c.String("trigger"), c.String("action"))
				},
			},
			{
				Name:  "delete",
				Usage: "Delete a skill",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleDeleteSkill(c.String("url"), c.String("api-key"), c.String("id"))
				},
			},
			{
				Name:  "extract",
				Usage: "Extract skills from interaction",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "content", Aliases: []string{"c"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleExtractSkills(c.String("url"), c.String("api-key"), c.String("format"), c.String("content"))
				},
			},
			{
				Name:  "suggest",
				Usage: "Suggest skills for a task",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "task", Aliases: []string{"t"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleSuggestSkills(c.String("url"), c.String("api-key"), c.String("format"), c.String("task"))
				},
			},
			{
				Name:  "execute",
				Usage: "Execute a skill by ID",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
					&cli.StringFlag{Name: "context", Aliases: []string{"c"}},
				},
				Action: func(c *cli.Context) error {
					return handleExecuteSkill(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"), c.String("context"))
				},
			},
		},
	}
}

func groupsCmd() *cli.Command {
	return &cli.Command{
		Name:  "groups",
		Usage: "Manage agent groups (multi-agent)",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List all groups",
				Action: func(c *cli.Context) error {
					return handleListGroups(c.String("url"), c.String("api-key"), c.String("format"))
				},
			},
			{
				Name:  "create",
				Usage: "Create a group",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleCreateGroup(c.String("url"), c.String("api-key"), c.String("format"), c.String("name"))
				},
			},
			{
				Name:  "get",
				Usage: "Get group by ID",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleGetGroup(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"))
				},
			},
			{
				Name:  "delete",
				Usage: "Delete a group",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleDeleteGroup(c.String("url"), c.String("api-key"), c.String("id"))
				},
			},
			{
				Name:  "add-agent",
				Usage: "Add agent to group",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "group-id", Aliases: []string{"g"}, Required: true},
					&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
					&cli.StringFlag{Name: "role", Aliases: []string{"r"}, Value: "member"},
				},
				Action: func(c *cli.Context) error {
					return handleAddAgentToGroup(c.String("url"), c.String("api-key"), c.String("format"), c.String("group-id"), c.String("agent-id"), c.String("role"))
				},
			},
			{
				Name:  "remove-agent",
				Usage: "Remove agent from group",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "group-id", Aliases: []string{"g"}, Required: true},
					&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleRemoveAgentFromGroup(c.String("url"), c.String("api-key"), c.String("group-id"), c.String("agent-id"))
				},
			},
		},
	}
}

func entitiesCmd() *cli.Command {
	return &cli.Command{
		Name:  "entities",
		Usage: "Manage knowledge graph entities",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List all entities",
				Action: func(c *cli.Context) error {
					return handleListEntities(c.String("url"), c.String("api-key"), c.String("format"))
				},
			},
			{
				Name:  "create",
				Usage: "Create an entity",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true},
					&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Required: true},
					&cli.StringFlag{Name: "properties", Aliases: []string{"p"}},
				},
				Action: func(c *cli.Context) error {
					return handleCreateEntity(c.String("url"), c.String("api-key"), c.String("format"), c.String("name"), c.String("type"), c.String("properties"))
				},
			},
			{
				Name:  "get",
				Usage: "Get entity by name or ID",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleGetEntity(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"))
				},
			},
			{
				Name:  "update",
				Usage: "Update an entity",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}},
					&cli.StringFlag{Name: "type", Aliases: []string{"t"}},
					&cli.StringFlag{Name: "properties", Aliases: []string{"p"}},
				},
				Action: func(c *cli.Context) error {
					return handleUpdateEntity(c.String("url"), c.String("api-key"), c.String("format"), c.String("id"), c.String("name"), c.String("type"), c.String("properties"))
				},
			},
			{
				Name:  "delete",
				Usage: "Delete an entity",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleDeleteEntity(c.String("url"), c.String("api-key"), c.String("id"))
				},
			},
			{
				Name:  "link",
				Usage: "Link two entities with a relation",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "from", Aliases: []string{"f"}, Required: true},
					&cli.StringFlag{Name: "to", Aliases: []string{"t"}, Required: true},
					&cli.StringFlag{Name: "relation", Aliases: []string{"r"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleLinkEntities(c.String("url"), c.String("api-key"), c.String("format"), c.String("from"), c.String("to"), c.String("relation"))
				},
			},
		},
	}
}

func backupCmd() *cli.Command {
	return &cli.Command{
		Name:  "backup",
		Usage: "Backup and restore memories",
		Subcommands: []*cli.Command{
			{
				Name:  "export",
				Usage: "Export memories to JSON file",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}},
				},
				Action: func(c *cli.Context) error {
					return handleExportBackup(c.String("url"), c.String("api-key"), c.String("agent-id"), c.String("output"))
				},
			},
			{
				Name:  "import",
				Usage: "Import memories from JSON file",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "agent-id", Aliases: []string{"a"}, Required: true},
					&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleImportBackup(c.String("url"), c.String("api-key"), c.String("agent-id"), c.String("file"))
				},
			},
		},
	}
}

func compressionCmd() *cli.Command {
	return &cli.Command{
		Name:  "compression",
		Usage: "Manage compression engine",
		Subcommands: []*cli.Command{
			{
				Name:  "stats",
				Usage: "Show compression statistics",
				Action: func(c *cli.Context) error {
					return handleCompressionStats(c.String("url"), c.String("api-key"), c.String("format"))
				},
			},
			{
				Name:  "mode",
				Usage: "Get or set compression mode",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "set", Aliases: []string{"s"}},
				},
				Action: func(c *cli.Context) error {
					return handleCompressionMode(c.String("url"), c.String("api-key"), c.String("format"), c.String("set"))
				},
			},
		},
	}
}

func tierCmd() *cli.Command {
	return &cli.Command{
		Name:  "tier",
		Usage: "Manage memory tier policy",
		Subcommands: []*cli.Command{
			{
				Name:  "policy",
				Usage: "Get or set tier policy",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "set", Aliases: []string{"s"}},
				},
				Action: func(c *cli.Context) error {
					return handleTierPolicy(c.String("url"), c.String("api-key"), c.String("format"), c.String("set"))
				},
			},
		},
	}
}

func authCmd() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Authentication commands",
		Subcommands: []*cli.Command{
			{
				Name:  "login",
				Usage: "Login to the API",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "email", Aliases: []string{"e"}, Required: true},
					&cli.StringFlag{Name: "password", Aliases: []string{"p"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleLogin(c.String("url"), c.String("api-key"), c.String("format"), c.String("email"), c.String("password"))
				},
			},
			{
				Name:  "logout",
				Usage: "Logout from the API",
				Action: func(c *cli.Context) error {
					return handleLogout(c.String("url"), c.String("api-key"))
				},
			},
			{
				Name:  "status",
				Usage: "Show current user info",
				Action: func(c *cli.Context) error {
					return handleAuthStatus(c.String("url"), c.String("api-key"), c.String("format"))
				},
			},
		},
	}
}

func webhooksCmd() *cli.Command {
	return &cli.Command{
		Name:  "webhooks",
		Usage: "Manage webhooks",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List all webhooks",
				Action: func(c *cli.Context) error {
					return handleListWebhooks(c.String("url"), c.String("api-key"), c.String("format"))
				},
			},
			{
				Name:  "create",
				Usage: "Create a webhook",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "hook-url", Aliases: []string{"u"}, Required: true},
					&cli.StringFlag{Name: "events", Aliases: []string{"e"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleCreateWebhook(c.String("url"), c.String("api-key"), c.String("format"), c.String("hook-url"), c.String("events"))
				},
			},
			{
				Name:  "delete",
				Usage: "Delete a webhook",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					return handleDeleteWebhook(c.String("url"), c.String("api-key"), c.String("id"))
				},
			},
		},
	}
}

func dashboardCmd() *cli.Command {
	return &cli.Command{
		Name:  "dashboard",
		Usage: "Open the web dashboard in your browser",
		Action: func(c *cli.Context) error {
			return handleDashboard(c.String("url"))
		},
	}
}

func docsCmd() *cli.Command {
	return &cli.Command{
		Name:  "docs",
		Usage: "Open documentation in your browser",
		Action: func(c *cli.Context) error {
			return handleDocs()
		},
	}
}

func monitorCmd() *cli.Command {
	return &cli.Command{
		Name:  "monitor",
		Usage: "Monitor memory events in real-time",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "interval", Aliases: []string{"i"}, Value: 5},
		},
		Action: func(c *cli.Context) error {
			return handleMonitor(c.String("url"), c.String("api-key"), c.String("format"), c.Int("interval"))
		},
	}
}

func completionCmd() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "Generate shell completion scripts",
		Subcommands: []*cli.Command{
			{
				Name:  "bash",
				Usage: "Generate bash completion script",
				Action: func(c *cli.Context) error {
					return handleCompletionBash()
				},
			},
			{
				Name:  "zsh",
				Usage: "Generate zsh completion script",
				Action: func(c *cli.Context) error {
					return handleCompletionZsh()
				},
			},
		},
	}
}
