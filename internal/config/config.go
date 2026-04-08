package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	Neo4j  Neo4jConfig
	Qdrant QdrantConfig
	OpenAI OpenAIConfig
	App    AppConfig
	Auth   AuthConfig
}

type Neo4jConfig struct {
	URI      string
	User     string
	Password string
	MaxConns int
}

type QdrantConfig struct {
	URL      string
	APIKey   string
	MaxConns int
}

type OpenAIConfig struct {
	APIKey   string
	Model    string
	EmbedDim int
}

type AppConfig struct {
	SyncInterval  time.Duration
	ContextWindow int
	BatchSize     int
	MessageBuffer int
	BufferTimeout time.Duration
	HTTPPort      string
}

type AuthConfig struct {
	Enabled      bool
	APIKeys      []string
	AdminAPIKeys []string
	JWTSecret    string
	TokenExpiry  time.Duration
}

func Load() *Config {
	return &Config{
		Neo4j: Neo4jConfig{
			URI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
			User:     getEnv("NEO4J_USER", "neo4j"),
			Password: getEnv("NEO4J_PASSWORD", "password"),
			MaxConns: 50,
		},
		Qdrant: QdrantConfig{
			URL:      getEnv("QDRANT_URL", "localhost:6333"),
			APIKey:   getEnv("QDRANT_API_KEY", ""),
			MaxConns: 100,
		},
		OpenAI: OpenAIConfig{
			APIKey:   getEnv("OPENAI_API_KEY", ""),
			Model:    getEnv("OPENAI_MODEL", "text-embedding-3-small"),
			EmbedDim: 1536,
		},
		App: AppConfig{
			SyncInterval:  time.Hour,
			ContextWindow: 50,
			BatchSize:     1000,
			MessageBuffer: 50,
			BufferTimeout: 5 * time.Second,
			HTTPPort:      getEnv("HTTP_PORT", ":8080"),
		},
		Auth: AuthConfig{
			Enabled:      getEnv("AUTH_ENABLED", "false") == "true",
			APIKeys:      parseAPIKeys(getEnv("API_KEYS", "")),
			AdminAPIKeys: parseAPIKeys(getEnv("ADMIN_API_KEYS", "")),
			JWTSecret:    getEnv("JWT_SECRET", ""),
			TokenExpiry:  24 * time.Hour,
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseAPIKeys(keys string) []string {
	if keys == "" {
		return []string{}
	}
	var result []string
	for _, k := range strings.Split(keys, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			result = append(result, k)
		}
	}
	return result
}
