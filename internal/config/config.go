package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Neo4j       Neo4jConfig  `validate:"required"`
	Qdrant      QdrantConfig `validate:"required"`
	OpenSearch  OpenSearchConfig
	PGVector    PGVectorConfig
	OpenAI      OpenAIConfig      `validate:"required"`
	App         AppConfig         `validate:"required"`
	Auth        AuthConfig        `validate:"required"`
	LLM         LLMConfig         `validate:"required"`
	Memory      MemoryConfig      `validate:"required"`
	Compaction  CompactionConfig  `validate:"required"`
	Compression CompressionConfig `validate:"required"`
	Reranker    RerankerConfig    `validate:"required"`
	Email       EmailConfig       `validate:"required"`
	Webhook     WebhookConfig     `validate:"required"`
	Privacy     PrivacyConfig
	Hooks       HooksConfig
	GCP         GCPConfig
	AWS         AWSConfig
	Storage     StorageConfig
	Telemetry   TelemetryConfig
	SSO         SSOConfig
}

type SSOConfig struct {
	Google SSOProviderConfig
	GitHub SSOProviderConfig
}

type SSOProviderConfig struct {
	ClientID     string `env:"" envDefault:""`
	ClientSecret string `env:"" envDefault:""`
	CallbackURL  string `env:"" envDefault:""`
}

type TelemetryConfig struct {
	Enabled      bool    `env:"TELEMETRY_ENABLED" envDefault:"false"`
	OTLPEndpoint string  `env:"OTLP_ENDPOINT" envDefault:"localhost:4317"`
	ServiceName  string  `env:"OTEL_SERVICE_NAME" envDefault:"hystersis"`
	SampleRate   float64 `env:"OTEL_SAMPLE_RATE" envDefault:"1.0"`
}

type EmailConfig struct {
	SMTPHost     string `env:"SMTP_HOST" envDefault:""`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUsername string `env:"SMTP_USERNAME" envDefault:""`
	SMTPPassword string `env:"SMTP_PASSWORD" envDefault:""`
	FromAddress  string `env:"FROM_ADDRESS" envDefault:"noreply@hystersis.ai"`
	UseTLS       bool   `env:"SMTP_USE_TLS" envDefault:"true"`
}

type WebhookConfig struct {
	URL    string `env:"WEBHOOK_URL" envDefault:""`
	Secret string `env:"WEBHOOK_SECRET" envDefault:""`
}

type PrivacyConfig struct {
	Enabled      bool   `env:"PRIVACY_FILTER_ENABLED" envDefault:"true"`
	RedactMode   string `env:"PRIVACY_REDACT_MODE" envDefault:"redact"`
	LogRedaction bool   `env:"PRIVACY_LOG_REDACTION" envDefault:"false"`
}

type HooksConfig struct {
	Enabled bool `env:"HOOKS_ENABLED" envDefault:"true"`
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

type PGVectorConfig struct {
	URL        string `env:"PGVECTOR_URL" envDefault:"postgres://localhost:5432/agent_memory?sslmode=disable"`
	TableName  string `env:"PGVECTOR_TABLE" envDefault:"memory_vectors"`
	VectorSize int    `env:"PGVECTOR_VECTOR_SIZE" envDefault:"1536"`
}

type OpenSearchConfig struct {
	URL            string  `env:"OPENSEARCH_URL" envDefault:"http://localhost:9200"`
	APIKey         string  `env:"OPENSEARCH_API_KEY" envDefault:""`
	Index          string  `env:"OPENSEARCH_INDEX" envDefault:"agent_memory"`
	VectorSize     int     `env:"OPENSEARCH_VECTOR_SIZE" envDefault:"1536"`
	ScoreThreshold float32 `env:"OPENSEARCH_SCORE_THRESHOLD" envDefault:"0.7"`
	Username       string  `env:"OPENSEARCH_USERNAME" envDefault:""`
	Password       string  `env:"OPENSEARCH_PASSWORD" envDefault:""`
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
	RedisURL        string        `env:"REDIS_URL" envDefault:""`
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
	ProcessingEnabled      bool     `env:"MEMORY_PROCESSING_ENABLED" envDefault:"true"`
	AutoExtractFacts       bool     `env:"MEMORY_AUTO_EXTRACT_FACTS" envDefault:"true"`
	AutoExtractEntities    bool     `env:"MEMORY_AUTO_EXTRACT_ENTITIES" envDefault:"true"`
	AutoExtractRelations   bool     `env:"MEMORY_AUTO_EXTRACT_RELATIONS" envDefault:"true"`
	DefaultImportance   string   `env:"MEMORY_DEFAULT_IMPORTANCE" envDefault:"medium"`
	ConflictResolution  bool     `env:"MEMORY_CONFLICT_RESOLUTION" envDefault:"true"`
	MaxImportances      []string `env:"MEMORY_MAX_IMPORTANCES"`
	CacheEnabled        bool     `env:"MEMORY_CACHE_ENABLED" envDefault:"true"`
	CacheTTL            int      `env:"MEMORY_CACHE_TTL" envDefault:"3600"`
	OntologyEnabled     bool     `env:"ONTOLOGY_ENABLED" envDefault:"true"`
	OntologySources     []string `env:"ONTOLOGY_SOURCES"`
	MultiSignalEnabled  bool     `env:"MULTI_SIGNAL_ENABLED" envDefault:"false"`
	ChunkingEnabled     bool     `env:"CHUNKING_ENABLED" envDefault:"false"`
	ChunkingMaxBytes    int      `env:"CHUNKING_MAX_BYTES" envDefault:"2048"`
	TemporalReasoning   bool     `env:"TEMPORAL_REASONING_ENABLED" envDefault:"true"`
	DecayEnabled        bool     `env:"MEMORY_DECAY_ENABLED" envDefault:"true"`
	SafetyEnabled       bool     `env:"MEMORY_SAFETY_ENABLED" envDefault:"true"`
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

type CompressionConfig struct {
	Enabled             bool    `env:"COMPRESSION_ENABLED" envDefault:"true"`
	Mode                string  `env:"COMPRESSION_MODE" envDefault:"extract"`
	TierPolicy          string  `env:"TIER_POLICY" envDefault:"balanced"`
	FastProvider        string  `env:"COMPRESSION_LLM_FAST_PROVIDER" envDefault:"openai"`
	FastModel           string  `env:"COMPRESSION_LLM_FAST_MODEL" envDefault:"gpt-4o-mini"`
	VerifyProvider      string  `env:"COMPRESSION_LLM_VERIFY_PROVIDER" envDefault:"anthropic"`
	VerifyModel         string  `env:"COMPRESSION_LLM_VERIFY_MODEL" envDefault:"claude-3-5-sonnet"`
	ComplexityThreshold float64 `env:"COMPRESSION_COMPLEXITY_THRESHOLD" envDefault:"0.6"`
	AsyncEnabled        bool    `env:"COMPRESSION_ASYNC_ENABLED" envDefault:"true"`
	WorkerCount         int     `env:"COMPRESSION_WORKER_COUNT" envDefault:"4"`
	// Spreading activation hyperparameters
	SpreadingInitialBudget float64 `env:"SPREADING_INITIAL_BUDGET" envDefault:"1.0"`
	SpreadingDecayFactor   float64 `env:"SPREADING_DECAY_FACTOR" envDefault:"0.85"`
	SpreadingThreshold     float64 `env:"SPREADING_THRESHOLD" envDefault:"0.1"`
	SpreadingMaxHops       int     `env:"SPREADING_MAX_HOPS" envDefault:"3"`
}

type RerankerConfig struct {
	Provider string `env:"RERANKER_PROVIDER" envDefault:"disabled"`
	APIKey   string `env:"RERANKER_API_KEY" envDefault:""`
	BaseURL  string `env:"RERANKER_BASE_URL" envDefault:""`
	Model    string `env:"RERANKER_MODEL" envDefault:"cohere/rerank-english-v2.0"`
}

type GCPConfig struct {
	ProjectID        string `env:"GCP_PROJECT_ID" envDefault:""`
	Region           string `env:"GCP_REGION" envDefault:"us-central1"`
	BucketName       string `env:"GCS_BUCKET" envDefault:""`
	PubSubTopic      string `env:"GCP_PUBSUB_TOPIC" envDefault:"hystersis-events"`
	UseSecretManager bool   `env:"GCP_USE_SECRET_MANAGER" envDefault:"false"`
}

type AWSConfig struct {
	Region            string `env:"AWS_REGION" envDefault:"us-east-1"`
	S3Bucket          string `env:"S3_BUCKET" envDefault:""`
	AccessKeyID       string `env:"AWS_ACCESS_KEY_ID" envDefault:""`
	SecretAccessKey   string `env:"AWS_SECRET_ACCESS_KEY" envDefault:""`
	UseSecretsManager bool   `env:"AWS_USE_SECRETS_MANAGER" envDefault:"false"`
}

type StorageConfig struct {
	Provider string `env:"STORAGE_PROVIDER" envDefault:"local"` // local, gcs, s3
	DataDir  string `env:"DATA_DIR" envDefault:"./data"`
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
		OpenSearch: OpenSearchConfig{
			URL:            getEnv("OPENSEARCH_URL", "http://localhost:9200"),
			APIKey:         getEnv("OPENSEARCH_API_KEY", ""),
			Index:          getEnv("OPENSEARCH_INDEX", "agent_memory"),
			VectorSize:     getEnvInt("OPENSEARCH_VECTOR_SIZE", 1536),
			ScoreThreshold: getEnvFloat32("OPENSEARCH_SCORE_THRESHOLD", 0.7),
			Username:       getEnv("OPENSEARCH_USERNAME", ""),
			Password:       getEnv("OPENSEARCH_PASSWORD", ""),
		},
		PGVector: PGVectorConfig{
			URL:        getEnv("PGVECTOR_URL", "postgres://localhost:5432/agent_memory?sslmode=disable"),
			TableName:  getEnv("PGVECTOR_TABLE", "memory_vectors"),
			VectorSize: getEnvInt("PGVECTOR_VECTOR_SIZE", 1536),
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
			MaxImportances:      parseOrigins(getEnv("MEMORY_MAX_IMPORTANCES", "")),
			CacheEnabled:        getEnv("MEMORY_CACHE_ENABLED", "true") == "true",
			CacheTTL:            getEnvInt("MEMORY_CACHE_TTL", 3600),
			OntologyEnabled:     getEnv("ONTOLOGY_ENABLED", "true") == "true",
			OntologySources:     parseOrigins(getEnv("ONTOLOGY_SOURCES", "")),
			MultiSignalEnabled:  getEnv("MULTI_SIGNAL_ENABLED", "true") == "true",
			ChunkingEnabled:     getEnv("CHUNKING_ENABLED", "false") == "true",
			ChunkingMaxBytes:    getEnvInt("CHUNKING_MAX_BYTES", 2048),
			TemporalReasoning:   getEnv("TEMPORAL_REASONING_ENABLED", "true") == "true",
			DecayEnabled:        getEnv("MEMORY_DECAY_ENABLED", "true") == "true",
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
		Compression: CompressionConfig{
			Enabled:                getEnv("COMPRESSION_ENABLED", "true") == "true",
			Mode:                   getEnv("COMPRESSION_MODE", "extract"),
			TierPolicy:             getEnv("TIER_POLICY", "balanced"),
			FastProvider:           getEnv("COMPRESSION_LLM_FAST_PROVIDER", "openai"),
			FastModel:              getEnv("COMPRESSION_LLM_FAST_MODEL", "gpt-4o-mini"),
			VerifyProvider:         getEnv("COMPRESSION_LLM_VERIFY_PROVIDER", "anthropic"),
			VerifyModel:            getEnv("COMPRESSION_LLM_VERIFY_MODEL", "claude-3-5-sonnet"),
			ComplexityThreshold:    getEnvFloat64("COMPRESSION_COMPLEXITY_THRESHOLD", 0.6),
			AsyncEnabled:           getEnv("COMPRESSION_ASYNC_ENABLED", "true") == "true",
			WorkerCount:            getEnvInt("COMPRESSION_WORKER_COUNT", 4),
			SpreadingInitialBudget: getEnvFloat64("SPREADING_INITIAL_BUDGET", 1.0),
			SpreadingDecayFactor:   getEnvFloat64("SPREADING_DECAY_FACTOR", 0.85),
			SpreadingThreshold:     getEnvFloat64("SPREADING_THRESHOLD", 0.1),
			SpreadingMaxHops:       getEnvInt("SPREADING_MAX_HOPS", 3),
		},
		Reranker: RerankerConfig{
			Provider: getEnv("RERANKER_PROVIDER", "disabled"),
			APIKey:   getEnv("RERANKER_API_KEY", ""),
			BaseURL:  getEnv("RERANKER_BASE_URL", "https://api.cohere.ai"),
			Model:    getEnv("RERANKER_MODEL", "cohere/rerank-english-v2.0"),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", ""),
			SMTPPort:     getEnvInt("SMTP_PORT", 587),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromAddress:  getEnv("FROM_ADDRESS", "noreply@hystersis.ai"),
			UseTLS:       getEnv("SMTP_USE_TLS", "true") == "true",
		},
		Webhook: WebhookConfig{
			URL:    getEnv("WEBHOOK_URL", ""),
			Secret: getEnv("WEBHOOK_SECRET", ""),
		},
		Privacy: PrivacyConfig{
			Enabled:      getEnv("PRIVACY_FILTER_ENABLED", "true") == "true",
			RedactMode:   getEnv("PRIVACY_REDACT_MODE", "redact"),
			LogRedaction: getEnv("PRIVACY_LOG_REDACTION", "false") == "true",
		},
		Hooks: HooksConfig{
			Enabled: getEnv("HOOKS_ENABLED", "true") == "true",
		},
		GCP: GCPConfig{
			ProjectID:        getEnv("GCP_PROJECT_ID", ""),
			Region:           getEnv("GCP_REGION", "us-central1"),
			BucketName:       getEnv("GCS_BUCKET", ""),
			PubSubTopic:      getEnv("GCP_PUBSUB_TOPIC", "hystersis-events"),
			UseSecretManager: getEnv("GCP_USE_SECRET_MANAGER", "false") == "true",
		},
		AWS: AWSConfig{
			Region:            getEnv("AWS_REGION", "us-east-1"),
			S3Bucket:          getEnv("S3_BUCKET", ""),
			AccessKeyID:       getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:   getEnv("AWS_SECRET_ACCESS_KEY", ""),
			UseSecretsManager: getEnv("AWS_USE_SECRETS_MANAGER", "false") == "true",
		},
		Storage: StorageConfig{
			Provider: getEnv("STORAGE_PROVIDER", "local"),
			DataDir:  getEnv("DATA_DIR", "./data"),
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
