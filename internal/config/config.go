package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Neo4j      Neo4jConfig      `validate:"required"`
	Qdrant     QdrantConfig     `validate:"required"`
	OpenAI     OpenAIConfig     `validate:"required"`
	App        AppConfig        `validate:"required"`
	Auth       AuthConfig       `validate:"required"`
	LLM        LLMConfig        `validate:"required"`
	Memory     MemoryConfig     `validate:"required"`
	Compaction CompactionConfig `validate:"required"`
	Reranker   RerankerConfig   `validate:"required"`
}

type Neo4jConfig struct {
	URI          string `env:"NEO4J_URI" envDefault:"bolt://localhost:7687"`
	User         string `env:"NEO4J_USER" envDefault:"neo4j"`
	Password     string `env:"NEO4J_PASSWORD" envDefault:""`
	MaxConns     int    `env:"NEO4J_MAX_CONNS" envDefault:"50"`
	ConnTimeout  int    `env:"NEO4J_CONN_TIMEOUT" envDefault:"30"`
	QueryTimeout int    `env:"NEO4J_QUERY_TIMEOUT" envDefault:"60"`
}

type QdrantConfig struct {
	URL            string  `env:"QDRANT_URL" envDefault:"http://localhost:6333"`
	APIKey         string  `env:"QDRANT_API_KEY" envDefault:""`
	MaxConns       int     `env:"QDRANT_MAX_CONNS" envDefault:"100"`
	Collection     string  `env:"QDRANT_COLLECTION" envDefault:"agent_memory"`
	VectorSize     int     `env:"QDRANT_VECTOR_SIZE" envDefault:"1536"`
	ScoreThreshold float32 `env:"QDRANT_SCORE_THRESHOLD" envDefault:"0.7"`
}

type OpenAIConfig struct {
	APIKey   string `env:"OPENAI_API_KEY" envDefault:""`
	Model    string `env:"OPENAI_MODEL" envDefault:"text-embedding-3-small"`
	EmbedDim int    `env:"OPENAI_EMBED_DIM" envDefault:"1536"`
	OrgID    string `env:"OPENAI_ORG_ID" envDefault:""`
	BaseURL  string `env:"OPENAI_BASE_URL" envDefault:"https://api.openai.com/v1"`
}

type AppConfig struct {
	Host            string        `env:"APP_HOST" envDefault:"0.0.0.0"`
	HTTPPort        string        `env:"HTTP_PORT" envDefault:":8080"`
	GRPCPort        string        `env:"GRPC_PORT" envDefault:":50051"`
	Mode            string        `env:"SERVER_MODE" envDefault:"http"`
	Environment     string        `env:"ENVIRONMENT" envDefault:"development"`
	ReadTimeout     int           `env:"READ_TIMEOUT" envDefault:"30"`
	WriteTimeout    int           `env:"WRITE_TIMEOUT" envDefault:"30"`
	IdleTimeout     int           `env:"IDLE_TIMEOUT" envDefault:"120"`
	ShutdownTimeout int           `env:"SHUTDOWN_TIMEOUT" envDefault:"10"`
	SyncInterval    time.Duration `env:"SYNC_INTERVAL" envDefault:"1h"`
	ContextWindow   int           `env:"CONTEXT_WINDOW" envDefault:"50"`
	BatchSize       int           `env:"BATCH_SIZE" envDefault:"1000"`
	MessageBuffer   int           `env:"MESSAGE_BUFFER" envDefault:"100"`
	BufferTimeout   time.Duration `env:"BUFFER_TIMEOUT" envDefault:"5s"`
	SentryDSN       string        `env:"SENTRY_DSN" envDefault:""`
}

type AuthConfig struct {
	Enabled        bool     `env:"AUTH_ENABLED" envDefault:"false"`
	APIKeys        []string `env:"API_KEYS"`
	AdminAPIKeys   []string `env:"ADMIN_API_KEYS"`
	JWTSecret      string   `env:"JWT_SECRET" envDefault:""`
	TokenExpiry    int      `env:"TOKEN_EXPIRY" envDefault:"86400"`
	AllowedOrigins []string `env:"ALLOWED_ORIGINS"`
}

type LLMConfig struct {
	Provider     string  `env:"LLM_PROVIDER" envDefault:"openai"`
	APIKey       string  `env:"LLM_API_KEY" envDefault:""`
	BaseURL      string  `env:"LLM_BASE_URL" envDefault:""`
	OrgID        string  `env:"LLM_ORG_ID" envDefault:""`
	Model        string  `env:"LLM_MODEL" envDefault:"gpt-4o"`
	MaxTokens    int     `env:"LLM_MAX_TOKENS" envDefault:"4096"`
	Temperature  float64 `env:"LLM_TEMPERATURE" envDefault:"0.7"`
	RetryMax     int     `env:"LLM_RETRY_MAX" envDefault:"3"`
	RetryTimeout int     `env:"LLM_RETRY_TIMEOUT" envDefault:"30"`
}

type MemoryConfig struct {
	ProcessingEnabled   bool     `env:"MEMORY_PROCESSING_ENABLED" envDefault:"true"`
	AutoExtractFacts    bool     `env:"MEMORY_AUTO_EXTRACT_FACTS" envDefault:"true"`
	AutoExtractEntities bool     `env:"MEMORY_AUTO_EXTRACT_ENTITIES" envDefault:"true"`
	DefaultImportance   string   `env:"MEMORY_DEFAULT_IMPORTANCE" envDefault:"medium"`
	ConflictResolution  bool     `env:"MEMORY_CONFLICT_RESOLUTION" envDefault:"true"`
	MaxImportances      []string `env:"MEMORY_MAX_IMPORTANCES"`
	CacheEnabled        bool     `env:"MEMORY_CACHE_ENABLED" envDefault:"true"`
	CacheTTL            int      `env:"MEMORY_CACHE_TTL" envDefault:"3600"`
}

type CompactionConfig struct {
	Enabled             bool    `env:"COMPACTION_ENABLED" envDefault:"true"`
	Interval            string  `env:"COMPACTION_INTERVAL" envDefault:"24h"`
	MinMemories         int     `env:"COMPACTION_MIN_MEMORIES" envDefault:"100"`
	ImportanceThreshold string  `env:"COMPACTION_IMPORTANCE_THRESHOLD" envDefault:"low"`
	SummarizeThreshold  int     `env:"COMPACTION_SUMMARIZE_THRESHOLD" envDefault:"1000"`
	ArchiveOld          bool    `env:"COMPACTION_ARCHIVE_OLD" envDefault:"true"`
	ArchiveAfterDays    int     `env:"COMPACTION_ARCHIVE_AFTER_DAYS" envDefault:"30"`
	Deduplicate         bool    `env:"COMPACTION_DEDUPLICATE" envDefault:"true"`
	SimilarityThreshold float32 `env:"COMPACTION_SIMILARITY_THRESHOLD" envDefault:"0.92"`
}

type RerankerConfig struct {
	Provider string `env:"RERANKER_PROVIDER" envDefault:"disabled"`
	APIKey   string `env:"RERANKER_API_KEY" envDefault:""`
	BaseURL  string `env:"RERANKER_BASE_URL" envDefault:""`
	Model    string `env:"RERANKER_MODEL" envDefault:"cohere/rerank-english-v2.0"`
}

type ServerConfig struct {
	Mode    string
	HTTPURL string
	GRPCURL string
}

func (c *Config) Validate() []error {
	var errs []error

	if c.Neo4j.URI == "" {
		errs = append(errs, fmt.Errorf("NEO4J_URI is required"))
	}
	if c.Qdrant.URL == "" {
		errs = append(errs, fmt.Errorf("QDRANT_URL is required"))
	}
	if c.App.HTTPPort == "" {
		errs = append(errs, fmt.Errorf("HTTP_PORT is required"))
	}

	return errs
}

func (c *Config) ServerConfig() *ServerConfig {
	return &ServerConfig{
		Mode:    c.App.Mode,
		HTTPURL: fmt.Sprintf("http://%s%s", c.App.Host, c.App.HTTPPort),
		GRPCURL: fmt.Sprintf("%s%s", c.App.Host, c.App.GRPCPort),
	}
}

func Load() *Config {
	return &Config{
		Neo4j: Neo4jConfig{
			URI:          getEnv("NEO4J_URI", "bolt://localhost:7687"),
			User:         getEnv("NEO4J_USER", "neo4j"),
			Password:     getEnv("NEO4J_PASSWORD", ""),
			MaxConns:     getEnvInt("NEO4J_MAX_CONNS", 50),
			ConnTimeout:  getEnvInt("NEO4J_CONN_TIMEOUT", 30),
			QueryTimeout: getEnvInt("NEO4J_QUERY_TIMEOUT", 60),
		},
		Qdrant: QdrantConfig{
			URL:            getEnv("QDRANT_URL", "http://localhost:6333"),
			APIKey:         getEnv("QDRANT_API_KEY", ""),
			MaxConns:       getEnvInt("QDRANT_MAX_CONNS", 100),
			Collection:     getEnv("QDRANT_COLLECTION", "agent_memory"),
			VectorSize:     getEnvInt("QDRANT_VECTOR_SIZE", 1536),
			ScoreThreshold: getEnvFloat32("QDRANT_SCORE_THRESHOLD", 0.7),
		},
		OpenAI: OpenAIConfig{
			APIKey:   getEnv("OPENAI_API_KEY", ""),
			Model:    getEnv("OPENAI_MODEL", "text-embedding-3-small"),
			EmbedDim: getEnvInt("OPENAI_EMBED_DIM", 1536),
			OrgID:    getEnv("OPENAI_ORG_ID", ""),
			BaseURL:  getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		},
		App: AppConfig{
			Host:            getEnv("APP_HOST", "0.0.0.0"),
			HTTPPort:        getEnv("HTTP_PORT", ":8080"),
			GRPCPort:        getEnv("GRPC_PORT", ":50051"),
			Mode:            getEnv("SERVER_MODE", "http"),
			ReadTimeout:     getEnvInt("READ_TIMEOUT", 30),
			WriteTimeout:    getEnvInt("WRITE_TIMEOUT", 30),
			IdleTimeout:     getEnvInt("IDLE_TIMEOUT", 120),
			ShutdownTimeout: getEnvInt("SHUTDOWN_TIMEOUT", 10),
			SyncInterval:    getEnvDuration("SYNC_INTERVAL", time.Hour),
			ContextWindow:   getEnvInt("CONTEXT_WINDOW", 50),
			BatchSize:       getEnvInt("BATCH_SIZE", 1000),
			MessageBuffer:   getEnvInt("MESSAGE_BUFFER", 100),
			BufferTimeout:   getEnvDuration("BUFFER_TIMEOUT", 5*time.Second),
		},
		Auth: AuthConfig{
			Enabled:        getEnv("AUTH_ENABLED", "false") == "true",
			APIKeys:        parseAPIKeys(getEnv("API_KEYS", "")),
			AdminAPIKeys:   parseAPIKeys(getEnv("ADMIN_API_KEYS", "")),
			JWTSecret:      getEnv("JWT_SECRET", ""),
			TokenExpiry:    getEnvInt("TOKEN_EXPIRY", 86400),
			AllowedOrigins: parseOrigins(getEnv("ALLOWED_ORIGINS", "*")),
		},
		LLM: LLMConfig{
			Provider:     getEnv("LLM_PROVIDER", "openai"),
			APIKey:       getEnv("LLM_API_KEY", ""),
			BaseURL:      getEnv("LLM_BASE_URL", ""),
			OrgID:        getEnv("LLM_ORG_ID", ""),
			Model:        getEnv("LLM_MODEL", "gpt-4o"),
			MaxTokens:    getEnvInt("LLM_MAX_TOKENS", 4096),
			Temperature:  getEnvFloat64("LLM_TEMPERATURE", 0.7),
			RetryMax:     getEnvInt("LLM_RETRY_MAX", 3),
			RetryTimeout: getEnvInt("LLM_RETRY_TIMEOUT", 30),
		},
		Memory: MemoryConfig{
			ProcessingEnabled:   getEnv("MEMORY_PROCESSING_ENABLED", "true") == "true",
			AutoExtractFacts:    getEnv("MEMORY_AUTO_EXTRACT_FACTS", "true") == "true",
			AutoExtractEntities: getEnv("MEMORY_AUTO_EXTRACT_ENTITIES", "true") == "true",
			DefaultImportance:   getEnv("MEMORY_DEFAULT_IMPORTANCE", "medium"),
			ConflictResolution:  getEnv("MEMORY_CONFLICT_RESOLUTION", "true") == "true",
			CacheEnabled:        getEnv("MEMORY_CACHE_ENABLED", "true") == "true",
			CacheTTL:            getEnvInt("MEMORY_CACHE_TTL", 3600),
		},
		Compaction: CompactionConfig{
			Enabled:             getEnv("COMPACTION_ENABLED", "true") == "true",
			Interval:            getEnv("COMPACTION_INTERVAL", "24h"),
			MinMemories:         getEnvInt("COMPACTION_MIN_MEMORIES", 100),
			ImportanceThreshold: getEnv("COMPACTION_IMPORTANCE_THRESHOLD", "low"),
			SummarizeThreshold:  getEnvInt("COMPACTION_SUMMARIZE_THRESHOLD", 1000),
			ArchiveOld:          getEnv("COMPACTION_ARCHIVE_OLD", "true") == "true",
			ArchiveAfterDays:    getEnvInt("COMPACTION_ARCHIVE_AFTER_DAYS", 30),
			Deduplicate:         getEnv("COMPACTION_DEDUPLICATE", "true") == "true",
			SimilarityThreshold: getEnvFloat32("COMPACTION_SIMILARITY_THRESHOLD", 0.92),
		},
		Reranker: RerankerConfig{
			Provider: getEnv("RERANKER_PROVIDER", "disabled"),
			APIKey:   getEnv("RERANKER_API_KEY", ""),
			BaseURL:  getEnv("RERANKER_BASE_URL", "https://api.cohere.ai"),
			Model:    getEnv("RERANKER_MODEL", "cohere/rerank-english-v2.0"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat32(key string, fallback float32) float32 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			return float32(f)
		}
	}
	return fallback
}

func getEnvFloat64(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
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

func parseOrigins(origins string) []string {
	if origins == "" || origins == "*" {
		return []string{"*"}
	}
	var result []string
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			result = append(result, o)
		}
	}
	return result
}
