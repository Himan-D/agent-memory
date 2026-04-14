package search

import (
	"context"
	"fmt"
	"strings"

	"agent-memory/internal/embedding"
	"agent-memory/internal/llm"
)

type SearchType string

const (
	SearchTypeGraphCompletion           SearchType = "GRAPH_COMPLETION"
	SearchTypeRAGCompletion             SearchType = "RAG_COMPLETION"
	SearchTypeChunks                    SearchType = "CHUNKS"
	SearchTypeSummaries                 SearchType = "SUMMARIES"
	SearchTypeGraphSummaryCompletion    SearchType = "GRAPH_SUMMARY_COMPLETION"
	SearchTypeGraphCompletionCoT        SearchType = "GRAPH_COMPLETION_COT"
	SearchTypeGraphCompletionContextExt SearchType = "GRAPH_COMPLETION_CONTEXT_EXTENSION"
	SearchTypeTripletCompletion         SearchType = "TRIPLET_COMPLETION"
	SearchTypeChunksLexical             SearchType = "CHUNKS_LEXICAL"
	SearchTypeCodingRules               SearchType = "CODING_RULES"
	SearchTypeTemporal                  SearchType = "TEMPORAL"
	SearchTypeCypher                    SearchType = "CYPHER"
	SearchTypeNaturalLanguage           SearchType = "NATURAL_LANGUAGE"
)

type Retriever interface {
	Retrieve(ctx context.Context, req *RetrievalRequest) (*RetrievalResult, error)
}

type RetrievalRequest struct {
	Query       string
	QueryVector []float32
	DatasetID   string
	SessionID   string
	Limit       int
	Filters     map[string]interface{}
}

type RetrievalResult struct {
	Type      SearchType
	Content   string
	Chunks    []ChunkResult
	Entities  []EntityResult
	Relations []RelationResult
	GraphData *GraphData
	Score     float64
	Metadata  map[string]interface{}
	Error     error
}

type ChunkResult struct {
	ID       string
	Text     string
	Score    float64
	Source   string
	Metadata map[string]interface{}
}

type EntityResult struct {
	ID       string
	Name     string
	Type     string
	Score    float64
	Metadata map[string]interface{}
}

type RelationResult struct {
	ID       string
	Source   string
	Target   string
	Type     string
	Score    float64
	Metadata map[string]interface{}
}

type GraphData struct {
	Nodes []GraphNode
	Edges []GraphEdge
}

type GraphNode struct {
	ID       string
	Label    string
	Property string
	Score    float64
}

type GraphEdge struct {
	Source   string
	Target   string
	Label    string
	Property string
	Score    float64
}

type GraphStore interface {
	FindNodes(label string, properties map[string]interface{}) ([]map[string]interface{}, error)
	FindEdges(fromID, toID string) ([]map[string]interface{}, error)
	ExecuteCypher(query string) ([]map[string]interface{}, error)
	GetSchema() (*GraphSchema, error)
}

type VectorStore interface {
	Search(collection string, query []float32, limit int) ([]SearchResult, error)
	GetByID(collection string, ids []string) ([]SearchResult, error)
}

type SearchResult struct {
	ID      string
	Score   float64
	Payload map[string]interface{}
}

type GraphSchema struct {
	Labels        []string
	Relationships []string
}

type SearchService struct {
	llm         llm.Provider
	embedding   *embedding.OpenAIEmbedding
	retrievers  map[SearchType]Retriever
	graphStore  GraphStore
	vectorStore VectorStore
}

func NewSearchService(llmClient llm.Provider, embedClient *embedding.OpenAIEmbedding, graph GraphStore, vector VectorStore) *SearchService {
	s := &SearchService{
		llm:         llmClient,
		embedding:   embedClient,
		graphStore:  graph,
		vectorStore: vector,
		retrievers:  make(map[SearchType]Retriever),
	}

	s.registerDefaultRetrievers()
	return s
}

func (s *SearchService) registerDefaultRetrievers() {
	s.retrievers[SearchTypeGraphCompletion] = &GraphCompletionRetriever{
		llm: s.llm, graphStore: s.graphStore, vectorStore: s.vectorStore, embedding: s.embedding,
	}
	s.retrievers[SearchTypeRAGCompletion] = &RAGCompletionRetriever{
		llm: s.llm, vectorStore: s.vectorStore, embedding: s.embedding,
	}
	s.retrievers[SearchTypeChunks] = &ChunksRetriever{
		vectorStore: s.vectorStore, embedding: s.embedding,
	}
	s.retrievers[SearchTypeSummaries] = &SummariesRetriever{
		graphStore: s.graphStore,
	}
	s.retrievers[SearchTypeCypher] = &CypherRetriever{
		graphStore: s.graphStore,
	}
	s.retrievers[SearchTypeNaturalLanguage] = &NaturalLanguageRetriever{
		llm: s.llm, graphStore: s.graphStore,
	}
}

func (s *SearchService) RegisterRetriever(searchType SearchType, retriever Retriever) {
	s.retrievers[searchType] = retriever
}

func (s *SearchService) Search(ctx context.Context, query string, searchType SearchType, limit int) (*RetrievalResult, error) {
	if searchType == "" {
		searchType = SearchTypeGraphCompletion
	}
	if limit <= 0 {
		limit = 10
	}

	retriever, ok := s.retrievers[searchType]
	if !ok {
		return nil, fmt.Errorf("retriever not found for search type: %s", searchType)
	}

	var queryVector []float32
	if s.embedding != nil {
		vec, err := s.embedding.GenerateEmbeddingWithContext(ctx, query)
		if err == nil {
			queryVector = vec
		}
	}

	req := &RetrievalRequest{
		Query:       query,
		QueryVector: queryVector,
		Limit:       limit,
	}

	return retriever.Retrieve(ctx, req)
}

func (s *SearchService) SearchWithOptions(ctx context.Context, query string, opts *SearchOptions) (*RetrievalResult, error) {
	searchType := SearchTypeGraphCompletion
	limit := 10

	if opts != nil {
		if opts.SearchType != "" {
			searchType = opts.SearchType
		}
		if opts.Limit > 0 {
			limit = opts.Limit
		}
	}

	var queryVector []float32
	if s.embedding != nil {
		vec, err := s.embedding.GenerateEmbeddingWithContext(ctx, query)
		if err == nil {
			queryVector = vec
		}
	}

	req := &RetrievalRequest{
		Query:       query,
		QueryVector: queryVector,
		SessionID:   opts.SessionID,
		DatasetID:   opts.DatasetID,
		Limit:       limit,
		Filters:     opts.Filters,
	}

	retriever, ok := s.retrievers[searchType]
	if !ok {
		return nil, fmt.Errorf("retriever not found: %s", searchType)
	}

	return retriever.Retrieve(ctx, req)
}

type SearchOptions struct {
	SessionID  string
	DatasetID  string
	SearchType SearchType
	Limit      int
	Filters    map[string]interface{}
}

type GraphCompletionRetriever struct {
	llm         llm.Provider
	graphStore  GraphStore
	vectorStore VectorStore
	embedding   *embedding.OpenAIEmbedding
}

func (r *GraphCompletionRetriever) Retrieve(ctx context.Context, req *RetrievalRequest) (*RetrievalResult, error) {
	result := &RetrievalResult{
		Type:     SearchTypeGraphCompletion,
		Metadata: make(map[string]interface{}),
	}

	chunks, err := r.retrieveRelevantChunks(ctx, req)
	if err == nil {
		result.Chunks = chunks
	}

	entities, err := r.retrieveRelevantEntities(ctx, req)
	if err == nil {
		result.Entities = entities
	}

	if len(chunks) > 0 || len(entities) > 0 {
		answer, err := r.generateCompletion(ctx, req.Query, result)
		if err == nil {
			result.Content = answer
		}
	}

	return result, nil
}

func (r *GraphCompletionRetriever) retrieveRelevantChunks(ctx context.Context, req *RetrievalRequest) ([]ChunkResult, error) {
	if r.vectorStore == nil || len(req.QueryVector) == 0 {
		return nil, nil
	}

	results, err := r.vectorStore.Search("chunks", req.QueryVector, req.Limit)
	if err != nil {
		return nil, err
	}

	chunks := make([]ChunkResult, 0, len(results))
	for _, res := range results {
		content := ""
		if c, ok := res.Payload["content"].(string); ok {
			content = c
		}
		source := ""
		if s, ok := res.Payload["source"].(string); ok {
			source = s
		}
		chunks = append(chunks, ChunkResult{
			ID:       res.ID,
			Text:     content,
			Score:    res.Score,
			Source:   source,
			Metadata: res.Payload,
		})
	}

	return chunks, nil
}

func (r *GraphCompletionRetriever) retrieveRelevantEntities(ctx context.Context, req *RetrievalRequest) ([]EntityResult, error) {
	if r.graphStore == nil {
		return nil, nil
	}

	entities, err := r.graphStore.FindNodes("Entity", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	results := make([]EntityResult, 0, len(entities))
	for _, e := range entities {
		results = append(results, EntityResult{
			ID:       fmt.Sprintf("%v", e["id"]),
			Name:     fmt.Sprintf("%v", e["name"]),
			Type:     fmt.Sprintf("%v", e["type"]),
			Score:    1.0,
			Metadata: e,
		})
	}

	return results, nil
}

func (r *GraphCompletionRetriever) generateCompletion(ctx context.Context, query string, result *RetrievalResult) (string, error) {
	if r.llm == nil {
		return r.fallbackGenerate(query, result)
	}

	var contextBuilder strings.Builder
	contextBuilder.WriteString("Context:\n\n")

	for i, chunk := range result.Chunks {
		contextBuilder.WriteString(fmt.Sprintf("[%d] %s (source: %s)\n\n", i+1, chunk.Text, chunk.Source))
	}

	if len(result.Entities) > 0 {
		contextBuilder.WriteString("Entities found:\n")
		for _, entity := range result.Entities {
			contextBuilder.WriteString(fmt.Sprintf("- %s (%s)\n", entity.Name, entity.Type))
		}
		contextBuilder.WriteString("\n")
	}

	prompt := fmt.Sprintf(`Based on the following context, answer the question concisely and accurately.

%s

Question: %s

Answer:`, contextBuilder.String(), query)

	resp, err := r.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       "gpt-4o",
		MaxTokens:   1000,
		Temperature: 0.3,
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

func (r *GraphCompletionRetriever) fallbackGenerate(query string, result *RetrievalResult) (string, error) {
	if len(result.Chunks) > 0 {
		return result.Chunks[0].Text, nil
	}
	return "No relevant information found.", nil
}

type RAGCompletionRetriever struct {
	llm         llm.Provider
	vectorStore VectorStore
	embedding   *embedding.OpenAIEmbedding
}

func (r *RAGCompletionRetriever) Retrieve(ctx context.Context, req *RetrievalRequest) (*RetrievalResult, error) {
	result := &RetrievalResult{
		Type:     SearchTypeRAGCompletion,
		Metadata: make(map[string]interface{}),
	}

	if r.vectorStore == nil || len(req.QueryVector) == 0 {
		return result, nil
	}

	results, err := r.vectorStore.Search("chunks", req.QueryVector, req.Limit)
	if err != nil {
		return nil, err
	}

	chunks := make([]ChunkResult, 0, len(results))
	for _, res := range results {
		content := ""
		if c, ok := res.Payload["content"].(string); ok {
			content = c
		}
		chunks = append(chunks, ChunkResult{
			ID:       res.ID,
			Text:     content,
			Score:    res.Score,
			Metadata: res.Payload,
		})
	}
	result.Chunks = chunks

	if r.llm != nil && len(chunks) > 0 {
		var contextBuilder strings.Builder
		for _, chunk := range chunks {
			contextBuilder.WriteString(chunk.Text)
			contextBuilder.WriteString("\n\n")
		}

		prompt := fmt.Sprintf(`Based on the following context, answer the question.

Context:
%s

Question: %s

Answer:`, contextBuilder.String(), req.Query)

		resp, err := r.llm.Complete(ctx, &llm.CompletionRequest{
			Messages: []llm.Message{
				{Role: "user", Content: prompt},
			},
			Model:       "gpt-4o",
			MaxTokens:   1000,
			Temperature: 0.3,
		})
		if err == nil {
			result.Content = resp.Content
		}
	}

	return result, nil
}

type ChunksRetriever struct {
	vectorStore VectorStore
	embedding   *embedding.OpenAIEmbedding
}

func (r *ChunksRetriever) Retrieve(ctx context.Context, req *RetrievalRequest) (*RetrievalResult, error) {
	result := &RetrievalResult{
		Type:     SearchTypeChunks,
		Metadata: make(map[string]interface{}),
	}

	if r.vectorStore == nil || len(req.QueryVector) == 0 {
		return result, nil
	}

	results, err := r.vectorStore.Search("chunks", req.QueryVector, req.Limit)
	if err != nil {
		return nil, err
	}

	chunks := make([]ChunkResult, 0, len(results))
	for _, res := range results {
		content := ""
		if c, ok := res.Payload["content"].(string); ok {
			content = c
		}
		chunks = append(chunks, ChunkResult{
			ID:       res.ID,
			Text:     content,
			Score:    res.Score,
			Metadata: res.Payload,
		})
	}
	result.Chunks = chunks

	return result, nil
}

type SummariesRetriever struct {
	graphStore GraphStore
}

func (r *SummariesRetriever) Retrieve(ctx context.Context, req *RetrievalRequest) (*RetrievalResult, error) {
	result := &RetrievalResult{
		Type:     SearchTypeSummaries,
		Metadata: make(map[string]interface{}),
	}

	return result, nil
}

type CypherRetriever struct {
	graphStore GraphStore
}

func (r *CypherRetriever) Retrieve(ctx context.Context, req *RetrievalRequest) (*RetrievalResult, error) {
	result := &RetrievalResult{
		Type:     SearchTypeCypher,
		Metadata: make(map[string]interface{}),
	}

	if r.graphStore == nil {
		return result, nil
	}

	nodes, err := r.graphStore.FindNodes("Entity", nil)
	if err != nil {
		return nil, err
	}

	entities := make([]EntityResult, 0, len(nodes))
	for _, n := range nodes {
		entities = append(entities, EntityResult{
			ID:       fmt.Sprintf("%v", n["id"]),
			Name:     fmt.Sprintf("%v", n["name"]),
			Type:     fmt.Sprintf("%v", n["type"]),
			Metadata: n,
		})
	}
	result.Entities = entities

	return result, nil
}

type NaturalLanguageRetriever struct {
	llm        llm.Provider
	graphStore GraphStore
}

func (r *NaturalLanguageRetriever) Retrieve(ctx context.Context, req *RetrievalRequest) (*RetrievalResult, error) {
	result := &RetrievalResult{
		Type:     SearchTypeNaturalLanguage,
		Metadata: make(map[string]interface{}),
	}

	if r.llm == nil || r.graphStore == nil {
		return result, nil
	}

	schema, err := r.graphStore.GetSchema()
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Convert this question to a Cypher query based on the graph schema.

Schema:
Labels: %v
Relationships: %v

Question: %s

Return only the Cypher query without any explanation.`, schema.Labels, schema.Relationships, req.Query)

	resp, err := r.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       "gpt-4o",
		MaxTokens:   500,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, err
	}

	results, err := r.graphStore.ExecuteCypher(resp.Content)
	if err != nil {
		return nil, err
	}

	nodes := make([]EntityResult, 0, len(results))
	for _, n := range results {
		nodes = append(nodes, EntityResult{
			ID:       fmt.Sprintf("%v", n["id"]),
			Name:     fmt.Sprintf("%v", n["name"]),
			Type:     fmt.Sprintf("%v", n["type"]),
			Metadata: n,
		})
	}
	result.Entities = nodes
	result.Content = resp.Content

	return result, nil
}
