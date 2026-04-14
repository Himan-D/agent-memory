package cognee

import (
	"context"
	"fmt"
	"strings"

	"agent-memory/internal/embedding"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/cognify"
	"agent-memory/internal/memory/datapoint"
	"agent-memory/internal/memory/feedback"
	"agent-memory/internal/memory/loaders"
	"agent-memory/internal/memory/search"
	"agent-memory/internal/memory/session"
)

type Cognee struct {
	llm         llm.Provider
	embedding   *embedding.OpenAIEmbedding
	cognify     *cognify.Cognify
	search      *search.SearchService
	session     *session.Manager
	feedback    *feedback.FeedbackManager
	loader      *loaders.LoaderRegistry
	memify      *MemifyEngine
	graphStore  GraphStore
	vectorStore VectorStore
}

type GraphStore interface {
	CreateNode(label string, properties map[string]interface{}) error
	CreateRelationship(fromID, toID, relType string, properties map[string]interface{}) error
	DeleteNode(id string) error
	DeleteRelationship(id string) error
	FindNodes(label string, properties map[string]interface{}) ([]map[string]interface{}, error)
	FindEdges(fromID, toID string) ([]map[string]interface{}, error)
	ExecuteCypher(query string) ([]map[string]interface{}, error)
	GetSchema() (*GraphSchema, error)
}

type VectorStore interface {
	Upsert(collection string, vectors []VectorRecord) error
	Search(collection string, query []float32, limit int) ([]VectorRecord, error)
	Delete(collection string, ids []string) error
}

type VectorRecord struct {
	ID      string
	Vector  []float32
	Score   float64
	Payload map[string]interface{}
}

type GraphSchema struct {
	Labels        []string
	Relationships []string
}

type Config struct {
	LLMProvider     llm.Provider
	EmbeddingClient *embedding.OpenAIEmbedding
	GraphStore      GraphStore
	VectorStore     VectorStore
}

type CogneeResult struct {
	Success bool
	ID      string
	Error   error
}

type RecallResult struct {
	Results []*search.RetrievalResult
	Error   error
}

func New(cfg *Config) (*Cognee, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	c := &Cognee{
		llm:       cfg.LLMProvider,
		embedding: cfg.EmbeddingClient,
	}

	c.graphStore = cfg.GraphStore
	c.vectorStore = cfg.VectorStore

	c.cognify = cognify.New(
		cfg.LLMProvider,
		cfg.EmbeddingClient,
		&GraphStoreAdapter{cfg.GraphStore},
		&VectorStoreAdapter{cfg.VectorStore},
		cognify.Config{
			ChunkSize:      1000,
			ChunkOverlap:   200,
			ChunkStrategy:  "recursive",
			EntityTypes:    []string{"PERSON", "ORGANIZATION", "LOCATION", "CONCEPT", "EVENT", "OBJECT"},
			Model:          "gpt-4o-mini",
			EmbedModel:     "text-embedding-3-small",
			MaxConcurrency: 5,
		},
	)

	c.search = search.NewSearchService(
		cfg.LLMProvider,
		cfg.EmbeddingClient,
		&GraphStoreAdapter{cfg.GraphStore},
		&SearchVectorStoreAdapter{cfg.VectorStore},
	)

	c.session = session.NewManager()
	c.feedback = feedback.NewManager(nil)
	c.loader = loaders.NewDefaultRegistry()

	c.memify = NewMemifyEngine(cfg.LLMProvider, MemifyConfig{
		EnableTripletEmbeddings: true,
		EnableCodingRules:       true,
	})

	return c, nil
}

func (c *Cognee) Remember(ctx context.Context, text string, opts *RememberOptions) error {
	if opts == nil {
		opts = &RememberOptions{}
	}

	if opts.DatasetID == "" {
		opts.DatasetID = "default"
	}

	_, err := c.cognify.Cognify(ctx, &cognify.CognifyInput{
		Text:       text,
		Source:     opts.Source,
		SourceType: "text",
		Metadata:   map[string]interface{}{"dataset_id": opts.DatasetID},
	})

	if err != nil {
		return fmt.Errorf("cognify failed: %w", err)
	}

	if opts.SessionID != "" {
		_, err = c.session.AddMemory(ctx, opts.SessionID, text, "user")
		if err != nil {
			return fmt.Errorf("failed to add to session: %w", err)
		}
	}

	return nil
}

func (c *Cognee) RememberFile(ctx context.Context, filePath string, opts *RememberOptions) error {
	if opts == nil {
		opts = &RememberOptions{}
	}

	if opts.DatasetID == "" {
		opts.DatasetID = "default"
	}

	points, err := c.loader.Load(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to load file: %w", err)
	}

	for _, point := range points {
		_, err := c.cognify.Cognify(ctx, &cognify.CognifyInput{
			Text:       point.Content,
			Source:     point.Source.Name,
			SourceType: point.Source.Type,
			Metadata:   map[string]interface{}{"dataset_id": opts.DatasetID},
		})
		if err != nil {
			continue
		}
	}

	return nil
}

func (c *Cognee) Recall(ctx context.Context, query string, opts *RecallOptions) (*RecallResult, error) {
	if opts == nil {
		opts = &RecallOptions{}
	}

	if opts.SearchType == "" {
		opts.SearchType = search.SearchTypeGraphCompletion
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	if opts.SessionID != "" {
		sessionCtx, err := c.session.GetContext(ctx, opts.SessionID, 8000)
		if err == nil && len(sessionCtx) > 0 {
			sessionQuery := buildContextQuery(query, sessionCtx)
			query = sessionQuery
		}
	}

	result, err := c.search.Search(ctx, query, opts.SearchType, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &RecallResult{
		Results: []*search.RetrievalResult{result},
	}, nil
}

func (c *Cognee) Forget(ctx context.Context, datasetID string) error {
	if datasetID == "" {
		return fmt.Errorf("dataset_id is required")
	}

	return nil
}

func (c *Cognee) AddFeedback(ctx context.Context, queryID string, fbType feedback.FeedbackType, score float64, resultID, sessionID string) error {
	_, err := c.feedback.Add(queryID, fbType, score, resultID, sessionID)
	return err
}

func (c *Cognee) GetSession(ctx context.Context, sessionID string) (*session.Session, error) {
	return c.session.GetSession(ctx, sessionID)
}

func (c *Cognee) CreateSession(ctx context.Context, userID, name string) (*session.Session, error) {
	return c.session.CreateSession(ctx, userID, name)
}

func (c *Cognee) GetSearchService() *search.SearchService {
	return c.search
}

func (c *Cognee) GetCognify() *cognify.Cognify {
	return c.cognify
}

type RememberOptions struct {
	DatasetID string
	SessionID string
	Source    string
	Metadata  map[string]interface{}
}

type RecallOptions struct {
	SearchType search.SearchType
	Limit      int
	SessionID  string
	DatasetID  string
	Filters    map[string]interface{}
}

func buildContextQuery(query string, contextPoints []*datapoint.DataPoint) string {
	var ctxBuilder strings.Builder
	ctxBuilder.WriteString("Context from memory:\n")
	for _, p := range contextPoints {
		ctxBuilder.WriteString("- ")
		ctxBuilder.WriteString(p.Content)
		ctxBuilder.WriteString("\n")
	}
	ctxBuilder.WriteString("\nQuestion: ")
	ctxBuilder.WriteString(query)
	return ctxBuilder.String()
}

type GraphStoreAdapter struct {
	store GraphStore
}

func (a *GraphStoreAdapter) CreateNode(label string, properties map[string]interface{}) error {
	if a.store == nil {
		return nil
	}
	return a.store.CreateNode(label, properties)
}

func (a *GraphStoreAdapter) CreateRelationship(fromID, toID, relType string, properties map[string]interface{}) error {
	if a.store == nil {
		return nil
	}
	return a.store.CreateRelationship(fromID, toID, relType, properties)
}

func (a *GraphStoreAdapter) FindNodes(label string, properties map[string]interface{}) ([]map[string]interface{}, error) {
	if a.store == nil {
		return nil, nil
	}
	return a.store.FindNodes(label, properties)
}

func (a *GraphStoreAdapter) FindEdges(fromID, toID string) ([]map[string]interface{}, error) {
	if a.store == nil {
		return nil, nil
	}
	return a.store.FindEdges(fromID, toID)
}

func (a *GraphStoreAdapter) ExecuteCypher(query string) ([]map[string]interface{}, error) {
	if a.store == nil {
		return nil, nil
	}
	return a.store.ExecuteCypher(query)
}

func (a *GraphStoreAdapter) GetSchema() (*search.GraphSchema, error) {
	if a.store == nil {
		return nil, nil
	}
	schema, err := a.store.GetSchema()
	if err != nil {
		return nil, err
	}
	return &search.GraphSchema{
		Labels:        schema.Labels,
		Relationships: schema.Relationships,
	}, nil
}

type VectorStoreAdapter struct {
	store VectorStore
}

func (a *VectorStoreAdapter) Upsert(collection string, vectors []cognify.VectorRecord) error {
	if a.store == nil {
		return nil
	}
	records := make([]VectorRecord, len(vectors))
	for i, v := range vectors {
		records[i] = VectorRecord{
			ID:      v.ID,
			Vector:  v.Vector,
			Payload: v.Payload,
		}
	}
	return a.store.Upsert(collection, records)
}

func (a *VectorStoreAdapter) Search(collection string, query []float32, limit int) ([]cognify.SearchResult, error) {
	if a.store == nil {
		return nil, nil
	}
	results, err := a.store.Search(collection, query, limit)
	if err != nil {
		return nil, err
	}
	searchResults := make([]cognify.SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = cognify.SearchResult{
			ID:      r.ID,
			Score:   r.Score,
			Payload: r.Payload,
		}
	}
	return searchResults, nil
}

func (a *VectorStoreAdapter) Delete(collection string, ids []string) error {
	if a.store == nil {
		return nil
	}
	return a.store.Delete(collection, ids)
}

type SearchVectorStoreAdapter struct {
	store VectorStore
}

func (a *SearchVectorStoreAdapter) Upsert(collection string, vectors []cognify.VectorRecord) error {
	if a.store == nil {
		return nil
	}
	records := make([]VectorRecord, len(vectors))
	for i, v := range vectors {
		records[i] = VectorRecord{
			ID:      v.ID,
			Vector:  v.Vector,
			Payload: v.Payload,
		}
	}
	return a.store.Upsert(collection, records)
}

func (a *SearchVectorStoreAdapter) Search(collection string, query []float32, limit int) ([]search.SearchResult, error) {
	if a.store == nil {
		return nil, nil
	}
	results, err := a.store.Search(collection, query, limit)
	if err != nil {
		return nil, err
	}
	searchResults := make([]search.SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = search.SearchResult{
			ID:      r.ID,
			Score:   r.Score,
			Payload: r.Payload,
		}
	}
	return searchResults, nil
}

func (a *SearchVectorStoreAdapter) GetByID(collection string, ids []string) ([]search.SearchResult, error) {
	if a.store == nil {
		return nil, nil
	}
	return nil, nil
}

func (a *SearchVectorStoreAdapter) Delete(collection string, ids []string) error {
	if a.store == nil {
		return nil
	}
	return a.store.Delete(collection, ids)
}

type MemifyConfig struct {
	EnableTripletEmbeddings bool
	EnableCodingRules       bool
	EnableCustomRules       bool
	Rules                   []string
}

type MemifyEngine struct {
	llm    llm.Provider
	config MemifyConfig
}

func NewMemifyEngine(llmClient llm.Provider, config MemifyConfig) *MemifyEngine {
	return &MemifyEngine{
		llm:    llmClient,
		config: config,
	}
}

func (e *MemifyEngine) Enrich(ctx context.Context, data *datapoint.ExtractedData) (*datapoint.ExtractedData, error) {
	return data, nil
}
