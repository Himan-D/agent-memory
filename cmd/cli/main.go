package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

var version = "0.2.0"

func main() {
	godotenv.Load()

	cfg := loadConfig()
	defaultURL := "http://localhost:8080"
	if cfg != nil && cfg.BaseURL != "" {
		defaultURL = cfg.BaseURL
	}

	app := &cli.App{
		Name:    "hystersis",
		Version: version,
		Usage:   "Persistent memory infrastructure for AI agents",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "API server URL",
				EnvVars: []string{"AGENT_MEMORY_URL", "HYSTERESIS_URL"},
				Value:   defaultURL,
			},
			&cli.StringFlag{
				Name:    "api-key",
				Aliases: []string{"k"},
				Usage:   "API key for authentication",
				EnvVars: []string{"AGENT_MEMORY_API_KEY", "HYSTERESIS_API_KEY"},
			},
			&cli.StringFlag{
				Name:  "format",
				Usage: "Output format: json, table, yaml",
				Value: "json",
			},
		},
		EnableBashCompletion: true,
		Commands:             commands(),
	}

	if err := app.Run(os.Args); err != nil {
		errorf("Error: %v", err)
		os.Exit(1)
	}
}
