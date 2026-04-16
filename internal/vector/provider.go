package vector

import (
	"context"

	"agent-memory/internal/memory/types"
)

type ProviderType string

const (
	ProviderQdrant      ProviderType = "qdrant"
	ProviderPinecone    ProviderType = "pinecone"
	ProviderWeaviate    ProviderType = "weaviate"
	ProviderChroma      ProviderType = "chroma"
	ProviderPgvector    ProviderType = "pgvector"
	ProviderMilvus      ProviderType = "milvus"
	ProviderElastic     ProviderType = "elasticsearch"
	ProviderVespa       ProviderType = "vespa"
	ProviderRedis       ProviderType = "redis"
	ProviderMongo       ProviderType = "mongodb"
	ProviderAzureSearch ProviderType = "azure"
	ProviderOpenSearch  ProviderType = "opensearch"
)

type VectorProvider interface {
	Name() ProviderType
	StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, metadata map[string]interface{}) (string, error)
	Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error)
	UpdateMemory(ctx context.Context, id string, text string, metadata map[string]interface{}) error
	DeleteMemory(ctx context.Context, id string) error
	UpdateVector(ctx context.Context, id string, embedding []float32) error
	DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error)
	Ping(ctx context.Context) error
	Close() error
}

type Config struct {
	Provider   ProviderType `env:"VECTOR_PROVIDER" envDefault:"qdrant"`
	VectorSize int          `env:"VECTOR_SIZE" envDefault:"1536"`

	Qdrant      QdrantConfig      `envPrefix:"QDRANT_"`
	Pinecone    PineconeConfig    `envPrefix:"PINECONE_"`
	Weaviate    WeaviateConfig    `envPrefix:"WEAVIATE_"`
	Chroma      ChromaConfig      `envPrefix:"CHROMA_"`
	Pgvector    PgvectorConfig    `envPrefix:"PGVECTOR_"`
	Milvus      MilvusConfig      `envPrefix:"MILVUS_"`
	Elastic     ElasticConfig     `envPrefix:"ELASTIC_"`
	Vespa       VespaConfig       `envPrefix:"VESPA_"`
	Redis       RedisConfig       `envPrefix:"REDIS_"`
	Mongo       MongoConfig       `envPrefix:"MONGO_"`
	AzureSearch AzureSearchConfig `envPrefix:"AZURE_SEARCH_"`
	OpenSearch  OpenSearchConfig  `envPrefix:"OPENSEARCH_"`
}

type QdrantConfig struct {
	URL        string `env:"URL" envDefault:"http://localhost:6333"`
	APIKey     string `env:"API_KEY" envDefault:""`
	Collection string `env:"COLLECTION" envDefault:"agent_memory"`
}

type PineconeConfig struct {
	APIKey      string `env:"API_KEY" envDefault:""`
	Environment string `env:"ENVIRONMENT" envDefault:"us-east-1"`
	Index       string `env:"INDEX" envDefault:"agent-memory"`
	Cloud       string `env:"CLOUD" envDefault:"aws"`
	Dimension   int    `env:"DIMENSION" envDefault:"1536"`
}

type WeaviateConfig struct {
	URL       string `env:"URL" envDefault:"http://localhost:8080"`
	APIKey    string `env:"API_KEY" envDefault:""`
	ClassName string `env:"CLASS_NAME" envDefault:"AgentMemory"`
	Dimension int    `env:"DIMENSION" envDefault:"1536"`
}

type ChromaConfig struct {
	URL        string `env:"URL" envDefault:"http://localhost:8000"`
	APIKey     string `env:"API_KEY" envDefault:""`
	Collection string `env:"COLLECTION" envDefault:"agent_memory"`
	Dimension  int    `env:"DIMENSION" envDefault:"1536"`
}

type PgvectorConfig struct {
	Host      string `env:"HOST" envDefault:"localhost"`
	Port      int    `env:"PORT" envDefault:"5432"`
	User      string `env:"USER" envDefault:"postgres"`
	Password  string `env:"PASSWORD" envDefault:""`
	Database  string `env:"DATABASE" envDefault:"agentmemory"`
	SSLMode   string `env:"SSL_MODE" envDefault:"disable"`
	Dimension int    `env:"DIMENSION" envDefault:"1536"`
}

type MilvusConfig struct {
	URL        string `env:"URL" envDefault:"http://localhost:19530"`
	APIKey     string `env:"API_KEY" envDefault:""`
	Collection string `env:"COLLECTION" envDefault:"agent_memory"`
}

type ElasticConfig struct {
	Addresses []string `env:"ADDRESSES" envDefault:"http://localhost:9200"`
	APIKey    string   `env:"API_KEY" envDefault:""`
	Index     string   `env:"INDEX" envDefault:"agent-memory"`
}

type VespaConfig struct {
	URL         string `env:"URL" envDefault:"http://localhost:8080"`
	APIKey      string `env:"API_KEY" envDefault:""`
	Zone        string `env:"ZONE" envDefault:"default"`
	Application string `env:"APPLICATION" envDefault:"default"`
}

type RedisConfig struct {
	Addr      string `env:"ADDR" envDefault:"localhost:6379"`
	Password  string `env:"PASSWORD" envDefault:""`
	DB        int    `env:"DB" envDefault:"0"`
	Dimension int    `env:"DIMENSION" envDefault:"1536"`
}

type MongoConfig struct {
	URI        string `env:"URI" envDefault:"mongodb://localhost:27017"`
	APIKey     string `env:"API_KEY" envDefault:""`
	Database   string `env:"DATABASE" envDefault:"agentmemory"`
	Collection string `env:"COLLECTION" envDefault:"memories"`
	Dimension  int    `env:"DIMENSION" envDefault:"1536"`
}

type AzureSearchConfig struct {
	URL       string `env:"URL" envDefault:""`
	APIKey    string `env:"API_KEY" envDefault:""`
	IndexName string `env:"INDEX_NAME" envDefault:"agent-memory"`
	Dimension int    `env:"DIMENSION" envDefault:"1536"`
}

type OpenSearchConfig struct {
	URL            string `env:"URL" envDefault:"http://localhost:9200"`
	APIKey         string `env:"API_KEY" envDefault:""`
	Index          string `env:"INDEX" envDefault:"agent_memory"`
	Username       string `env:"USERNAME" envDefault:""`
	Password       string `env:"PASSWORD" envDefault:""`
	VectorField    string `env:"VECTOR_FIELD" envDefault:"vector"`
	TextField     string `env:"TEXT_FIELD" envDefault:"text"`
	IdField       string `env:"ID_FIELD" envDefault:"id"`
	Dimension     int    `env:"DIMENSION" envDefault:"1536"`
}

func NewVectorProvider(cfg *Config) (VectorProvider, error) {
	switch cfg.Provider {
	case ProviderQdrant:
		return newQdrantProvider(cfg), nil
	case ProviderPinecone:
		return newPineconeProvider(cfg)
	case ProviderWeaviate:
		return newWeaviateProvider(cfg)
	case ProviderChroma:
		return newChromaProvider(cfg)
	case ProviderPgvector:
		return newPgvectorProvider(cfg)
	case ProviderMilvus:
		return newMilvusProvider(cfg)
	case ProviderElastic:
		return newElasticProvider(cfg)
	case ProviderVespa:
		return newVespaProvider(cfg)
	case ProviderRedis:
		return newRedisProvider(cfg)
	case ProviderMongo:
		return newMongoProvider(cfg)
	case ProviderAzureSearch:
		return newAzureSearchProvider(cfg)
	case ProviderOpenSearch:
		return newOpenSearchProvider(cfg)
	default:
		return newQdrantProvider(cfg), nil
	}
}
