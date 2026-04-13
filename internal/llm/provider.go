package llm

import (
	"context"
)

type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderAzure     ProviderType = "azure"
	ProviderGoogle    ProviderType = "google"
	ProviderMistral   ProviderType = "mistral"
	ProviderCohere    ProviderType = "cohere"
	ProviderLocal     ProviderType = "local"
	ProviderAWS       ProviderType = "aws"
	ProviderGroq      ProviderType = "groq"
	ProviderDeepSeek  ProviderType = "deepseek"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type CompletionRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	Temperature      float64   `json:"temperature"`
	MaxTokens        int       `json:"max_tokens"`
	TopP             float64   `json:"top_p,omitempty"`
	FrequencyPenalty float64   `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64   `json:"presence_penalty,omitempty"`
	Stop             []string  `json:"stop,omitempty"`
}

type CompletionResponse struct {
	Content    string       `json:"content"`
	Model      string       `json:"model"`
	Provider   ProviderType `json:"provider"`
	Tokens     int          `json:"tokens,omitempty"`
	StopReason string       `json:"stop_reason,omitempty"`
}

type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbeddingResponse struct {
	Embedding []float32    `json:"embedding"`
	Model     string       `json:"model"`
	Provider  ProviderType `json:"provider"`
	Tokens    int          `json:"tokens,omitempty"`
}

type RerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopK      int      `json:"top_k"`
}

type RerankResponse struct {
	Results  []RerankResult `json:"results"`
	Model    string         `json:"model"`
	Provider ProviderType   `json:"provider"`
}

type RerankResult struct {
	Index    int     `json:"index"`
	Document string  `json:"document"`
	Score    float64 `json:"score"`
}

type Provider interface {
	Name() ProviderType
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
	Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
	Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error)
}

type Config struct {
	Provider     ProviderType `env:"LLM_PROVIDER" envDefault:"openai"`
	APIKey       string       `env:"LLM_API_KEY" envDefault:""`
	BaseURL      string       `env:"LLM_BASE_URL" envDefault:""`
	Organization string       `env:"LLM_ORG_ID" envDefault:""`

	OpenAI    OpenAIConfig    `envPrefix:"OPENAI_"`
	Anthropic AnthropicConfig `envPrefix:"ANTHROPIC_"`
	Azure     AzureConfig     `envPrefix:"AZURE_"`
	Google    GoogleConfig    `envPrefix:"GOOGLE_"`
	Mistral   MistralConfig   `envPrefix:"MISTRAL_"`
	Cohere    CohereConfig    `envPrefix:"COHERE_"`
	Local     LocalConfig     `envPrefix:"LOCAL_"`
	AWS       AWSConfig       `envPrefix:"AWS_"`
	Groq      GroqConfig      `envPrefix:"GROQ_"`
	DeepSeek  DeepSeekConfig  `envPrefix:"DEEPSEEK_"`
}

type OpenAIConfig struct {
	Model       string  `env:"MODEL" envDefault:"gpt-4o"`
	EmbedModel  string  `env:"EMBED_MODEL" envDefault:"text-embedding-3-small"`
	EmbedDim    int     `env:"EMBED_DIM" envDefault:"1536"`
	Temperature float64 `env:"TEMPERATURE" envDefault:"0.7"`
	MaxTokens   int     `env:"MAX_TOKENS" envDefault:"4096"`
}

type AnthropicConfig struct {
	Model       string  `env:"MODEL" envDefault:"claude-sonnet-4-20250514"`
	EmbedModel  string  `env:"EMBED_MODEL" envDefault:""`
	Temperature float64 `env:"TEMPERATURE" envDefault:"0.7"`
	MaxTokens   int     `env:"MAX_TOKENS" envDefault:"4096"`
}

type AzureConfig struct {
	Endpoint    string `env:"ENDPOINT" envDefault:""`
	Deployment  string `env:"DEPLOYMENT" envDefault:""`
	APIVersion  string `env:"API_VERSION" envDefault:"2024-02-01"`
	EmbedEngine string `env:"EMBED_ENGINE" envDefault:""`
}

type GoogleConfig struct {
	Model       string  `env:"MODEL" envDefault:"gemini-1.5-flash"`
	EmbedModel  string  `env:"EMBED_MODEL" envDefault:"text-embedding-004"`
	Temperature float64 `env:"TEMPERATURE" envDefault:"0.7"`
	MaxTokens   int     `env:"MAX_TOKENS" envDefault:"4096"`
}

type MistralConfig struct {
	Model       string  `env:"MODEL" envDefault:"mistral-large-latest"`
	EmbedModel  string  `env:"EMBED_MODEL" envDefault:"mistral-embed"`
	Temperature float64 `env:"TEMPERATURE" envDefault:"0.7"`
	MaxTokens   int     `env:"MAX_TOKENS" envDefault:"4096"`
}

type CohereConfig struct {
	Model       string  `env:"MODEL" envDefault:"command-r-plus"`
	EmbedModel  string  `env:"EMBED_MODEL" envDefault:"embed-english-v3.0"`
	RerankModel string  `env:"RERANK_MODEL" envDefault:"rerank-english-v3.0"`
	Temperature float64 `env:"TEMPERATURE" envDefault:"0.7"`
	MaxTokens   int     `env:"MAX_TOKENS" envDefault:"4096"`
}

type LocalConfig struct {
	URL         string  `env:"URL" envDefault:"http://localhost:11434"`
	Model       string  `env:"MODEL" envDefault:"llama3"`
	EmbedModel  string  `env:"EMBED_MODEL" envDefault:""`
	Temperature float64 `env:"TEMPERATURE" envDefault:"0.7"`
	MaxTokens   int     `env:"MAX_TOKENS" envDefault:"4096"`
}

type AWSConfig struct {
	Region          string  `env:"REGION" envDefault:"us-east-1"`
	Model           string  `env:"MODEL" envDefault:"anthropic.claude-3-5-sonnet-20241022-v2:0"`
	EmbedModel      string  `env:"EMBED_MODEL" envDefault:""`
	Temperature     float64 `env:"TEMPERATURE" envDefault:"0.7"`
	MaxTokens       int     `env:"MAX_TOKENS" envDefault:"4096"`
	Endpoint        string  `env:"ENDPOINT" envDefault:""`
	AccessKeyID     string  `env:"ACCESS_KEY_ID" envDefault:""`
	SecretAccessKey string  `env:"SECRET_ACCESS_KEY" envDefault:""`
}

type GroqConfig struct {
	Model       string  `env:"MODEL" envDefault:"llama-3.3-70b-versatile"`
	EmbedModel  string  `env:"EMBED_MODEL" envDefault:""`
	Temperature float64 `env:"TEMPERATURE" envDefault:"0.7"`
	MaxTokens   int     `env:"MAX_TOKENS" envDefault:"4096"`
}

type DeepSeekConfig struct {
	Model       string  `env:"MODEL" envDefault:"deepseek-chat"`
	EmbedModel  string  `env:"EMBED_MODEL" envDefault:""`
	Temperature float64 `env:"TEMPERATURE" envDefault:"0.7"`
	MaxTokens   int     `env:"MAX_TOKENS" envDefault:"4096"`
}

func NewProvider(cfg *Config) (Provider, error) {
	switch cfg.Provider {
	case ProviderOpenAI:
		return newOpenAIProvider(cfg), nil
	case ProviderAnthropic:
		return newAnthropicProvider(cfg), nil
	case ProviderAzure:
		return newAzureProvider(cfg), nil
	case ProviderGoogle:
		return newGoogleProvider(cfg), nil
	case ProviderMistral:
		return newMistralProvider(cfg), nil
	case ProviderCohere:
		return newCohereProvider(cfg), nil
	case ProviderLocal:
		return newLocalProvider(cfg), nil
	case ProviderAWS:
		return newAWSProvider(cfg), nil
	case ProviderGroq:
		return newGroqProvider(cfg), nil
	case ProviderDeepSeek:
		return newDeepSeekProvider(cfg), nil
	default:
		return newOpenAIProvider(cfg), nil
	}
}

func GetDefaultModel(provider ProviderType, cfg *Config) string {
	switch provider {
	case ProviderOpenAI:
		return cfg.OpenAI.Model
	case ProviderAnthropic:
		return cfg.Anthropic.Model
	case ProviderAzure:
		return cfg.Azure.Deployment
	case ProviderGoogle:
		return cfg.Google.Model
	case ProviderMistral:
		return cfg.Mistral.Model
	case ProviderCohere:
		return cfg.Cohere.Model
	case ProviderLocal:
		return cfg.Local.Model
	case ProviderAWS:
		return cfg.AWS.Model
	case ProviderGroq:
		return cfg.Groq.Model
	case ProviderDeepSeek:
		return cfg.DeepSeek.Model
	default:
		return "gpt-4o"
	}
}

func GetDefaultEmbedModel(provider ProviderType, cfg *Config) string {
	switch provider {
	case ProviderOpenAI:
		return cfg.OpenAI.EmbedModel
	case ProviderAnthropic:
		return cfg.Anthropic.EmbedModel
	case ProviderGoogle:
		return cfg.Google.EmbedModel
	case ProviderMistral:
		return cfg.Mistral.EmbedModel
	case ProviderCohere:
		return cfg.Cohere.EmbedModel
	case ProviderLocal:
		return cfg.Local.EmbedModel
	case ProviderAWS:
		return cfg.AWS.EmbedModel
	case ProviderGroq:
		return cfg.Groq.EmbedModel
	case ProviderDeepSeek:
		return cfg.DeepSeek.EmbedModel
	default:
		return "text-embedding-3-small"
	}
}

func GetEmbedDim(provider ProviderType, cfg *Config) int {
	switch provider {
	case ProviderOpenAI:
		return cfg.OpenAI.EmbedDim
	default:
		return 1536
	}
}
