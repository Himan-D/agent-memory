package wiki

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

// MemorySearcher defines the interface for memory operations used by the wiki.
type MemorySearcher interface {
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
	CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error)
	AddEntity(entity types.Entity) (*types.Entity, error)
}

type Service struct {
	store    Store
	llm      llm.Provider
	llmModel string
	memSvc   MemorySearcher
}

func NewService(store Store, llmProvider llm.Provider, llmModel string, memSvc MemorySearcher) *Service {
	if llmModel == "" {
		llmModel = "gpt-4o-mini"
	}
	return &Service{
		store:    store,
		llm:      llmProvider,
		llmModel: llmModel,
		memSvc:   memSvc,
	}
}

func (s *Service) Ingest(ctx context.Context, req IngestRequest) (*IngestResult, error) {
	sourceID := uuid.New().String()
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.Content)))[:16]

	source := &Source{
		ID:          sourceID,
		Title:       req.Title,
		FilePath:    fmt.Sprintf("raw/%s.md", contentHash),
		ContentType: req.ContentType,
		Content:     req.Content,
		Metadata:    req.Metadata,
		AddedAt:     time.Now(),
	}
	if source.ContentType == "" {
		source.ContentType = "text/markdown"
	}
	if source.Title == "" {
		source.Title = fmt.Sprintf("Source %s", contentHash[:8])
	}

	if err := s.store.SaveSource(ctx, source); err != nil {
		return nil, fmt.Errorf("save source: %w", err)
	}

	summaryPage, entitiesCount, pagesCreated, err := s.ingestIntoWiki(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("wiki: ingest: %w", err)
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Operation: "ingest",
		Target:    source.Title,
		Details:   fmt.Sprintf("Created %d pages, updated %d entities", pagesCreated, entitiesCount),
	}
	if err := s.store.AddLog(ctx, entry); err != nil {
		log.Printf("wiki: failed to store log: %v", err)
	}

	mem := &types.Memory{
		ID:         uuid.New().String(),
		Content:    fmt.Sprintf("[Wiki Source] %s", source.Title),
		Type:       types.MemoryTypeUser,
		Importance: types.ImportanceHigh,
		Status:     types.MemoryStatusActive,
		Metadata: map[string]interface{}{
			"source_id":    sourceID,
			"content_hash": contentHash,
			"page_type":    "wiki_source",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if _, err := s.memSvc.CreateMemory(ctx, mem); err != nil {
		log.Printf("wiki: failed to store source memory: %v", err)
	}

	return &IngestResult{
		Source:          source,
		SummaryPage:     summaryPage,
		EntitiesUpdated: entitiesCount,
		PagesCreated:    pagesCreated,
	}, nil
}

func (s *Service) ingestIntoWiki(ctx context.Context, source *Source) (*Page, int, int, error) {
	summaryPrompt := fmt.Sprintf(`You are a knowledge base builder. Given the following source content, create a structured wiki summary.

Source: %s
Content:
%s

Create a comprehensive summary page. Include:
1. A clear title
2. Key points as bullet points
3. Entities mentioned (people, concepts, organizations)
4. Related topics that should link to other pages

Respond in JSON format:
{
  "title": "Short Descriptive Title",
  "content": "Full markdown content for the wiki page",
  "links": ["Related Topic 1", "Related Topic 2"],
  "tags": ["tag1", "tag2"],
  "entities": [{"name": "Entity Name", "type": "person|concept|org|place", "description": "Brief description"}]
}`, source.Title, truncate(source.Content, 8000))

	resp, err := s.llm.Complete(ctx, &llm.CompletionRequest{
		Model:       s.llmModel,
		Messages:    []llm.Message{{Role: "user", Content: summaryPrompt}},
		Temperature: 0.3,
		MaxTokens:   4000,
	})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("LLM summary generation: %w", err)
	}

	var result struct {
		Title    string   `json:"title"`
		Content  string   `json:"content"`
		Links    []string `json:"links"`
		Tags     []string `json:"tags"`
		Entities []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"entities"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, 0, 0, fmt.Errorf("parse LLM response: %w", err)
	}

	page := &Page{
		ID:        uuid.New().String(),
		Title:     result.Title,
		Type:      PageTypeSummary,
		Content:   result.Content,
		Links:     result.Links,
		Tags:      result.Tags,
		SourceIDs: []string{source.ID},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.SavePage(ctx, page); err != nil {
		return nil, 0, 0, fmt.Errorf("save summary page: %w", err)
	}

	// Update entities
	entitiesCount := 0
	for _, ent := range result.Entities {
		entity := types.Entity{
			Name: ent.Name,
			Type: ent.Type,
			Properties: map[string]interface{}{
				"description": fmt.Sprintf("Automatically extracted from source '%s'", source.Title),
				"source_id":   source.ID,
			},
			CreatedAt: time.Now(),
		}
		if _, err := s.memSvc.AddEntity(entity); err != nil {
			log.Printf("wiki: failed to add entity: %v", err)
		} else {
			entitiesCount++
		}
	}

	// Create additional pages for each link if they don't exist
	pagesCreated := 0
	for _, link := range result.Links {
		existing, err := s.store.GetPage(ctx, link)
		if err != nil && !strings.Contains(err.Error(), "page not found") {
			log.Printf("wiki: error checking link page %s: %v", link, err)
			continue
		}
		if existing != nil {
			continue
		}

		// Create a stub page for the link
		linkPage := &Page{
			ID:        link, // Use link title as ID (simplified)
			Title:     link,
			Type:      PageTypeConcept,
			Content:   fmt.Sprintf("This page is a placeholder for '%s'. Content will be added later.", link),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.store.SavePage(ctx, linkPage); err != nil {
			log.Printf("wiki: failed to create link page %s: %v", link, err)
		} else {
			pagesCreated++
		}
	}

	return page, entitiesCount, pagesCreated, nil
}

func (s *Service) Query(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	maxResults := 5
	if req.MaxContext > 0 {
		maxResults = req.MaxContext
	}
	if req.MaxContext == 0 {
		maxResults = 5
	}

	scoredPages := s.rankPagesByRelevance(req.Query, maxResults)
	result := make([]*Page, 0, maxResults)
	for i, sp := range scoredPages {
		if i >= maxResults {
			break
		}
		result = append(result, sp.page)
	}

	if len(result) == 0 && s.memSvc != nil {
		memories, err := s.memSvc.SearchMemories(ctx, &types.SearchRequest{
			Query:     req.Query,
			Limit:     maxResults,
			Threshold: 0.5,
		})
		if err == nil && len(memories) > 0 {
			var memPages []*Page
			for _, m := range memories {
				memPages = append(memPages, &Page{
					ID:        m.MemoryID,
					Title:     m.Text,
					Type:      PageTypeAnalysis,
					Content:   m.Text,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			}
			result = memPages
		}
	}

	if len(result) == 0 {
		return &QueryResult{Answer: "No relevant information found."}, nil
	}

	var contextStrs []string
	for i, p := range result {
		contextStrs = append(contextStrs, fmt.Sprintf("Context %d:\n%s\n", i+1, p.Content))
	}

	queryPrompt := fmt.Sprintf(`You are a helpful assistant. Given the following context, answer the question.

Question: %s

Context:
%s`, req.Query, strings.Join(contextStrs, "\n"))

	resp, err := s.llm.Complete(ctx, &llm.CompletionRequest{
		Model:       s.llmModel,
		Messages:    []llm.Message{{Role: "user", Content: queryPrompt}},
		Temperature: 0.3,
		MaxTokens:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM query: %w", err)
	}

	answer := strings.TrimSpace(resp.Content)
	if answer == "" {
		answer = "No answer could be generated."
	}

	queryResult := &QueryResult{
		Answer: answer,
	}

	if req.SaveAsPage && answer != "" {
		pageID := uuid.New().String()
		queryResult.PageID = pageID
		page := &Page{
			ID:        pageID,
			Title:     req.Query,
			Type:      PageTypeAnalysis,
			Content:   answer,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.store.SavePage(ctx, page); err != nil {
			log.Printf("wiki: failed to save answer page: %v", err)
		}
	}

	return queryResult, nil
}

func (s *Service) rankPagesByRelevance(query string, maxResults int) []*scoredPage {
	ql := strings.ToLower(query)
	var scoredPages []*scoredPage
	for _, p := range s.getAllPages() {
		score := 0.0
		if strings.Contains(strings.ToLower(p.Title), ql) {
			score += 3.0
		}
		if strings.Contains(strings.ToLower(p.Content), ql) {
			score += 1.0
		}
		if score > 0 {
			scoredPages = append(scoredPages, &scoredPage{page: p, score: score})
		}
	}

	// Simple sort by score descending (bubble sort)
	for i := 0; i < len(scoredPages); i++ {
		for j := i + 1; j < len(scoredPages); j++ {
			if scoredPages[i].score < scoredPages[j].score {
				scoredPages[i], scoredPages[j] = scoredPages[j], scoredPages[i]
			}
		}
	}

	if len(scoredPages) > maxResults {
		scoredPages = scoredPages[:maxResults]
	}
	return scoredPages
}

type scoredPage struct {
	page  *Page
	score float64
}

func (s *Service) getAllPages() []*Page {
	pages, _, _ := s.store.ListPages(context.Background(), 0, 0)
	return pages
}

func (s *Service) GetPage(ctx context.Context, id string) (*Page, error) {
	return s.store.GetPage(ctx, id)
}

func (s *Service) UpdatePage(ctx context.Context, page *Page) error {
	return s.store.SavePage(ctx, page)
}

func (s *Service) DeletePage(ctx context.Context, id string) error {
	return s.store.DeletePage(ctx, id)
}

func (s *Service) ListPages(ctx context.Context, limit, offset int) ([]*Page, int64, error) {
	return s.store.ListPages(ctx, limit, offset)
}

func (s *Service) SearchPages(ctx context.Context, query string, limit int) ([]*Page, error) {
	return s.store.SearchPages(ctx, query, limit)
}

func (s *Service) GetSource(ctx context.Context, id string) (*Source, error) {
	return s.store.GetSource(ctx, id)
}

func (s *Service) ListSources(ctx context.Context, limit, offset int) ([]*Source, int64, error) {
	return s.store.ListSources(ctx, limit, offset)
}

func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	index, err := s.store.GetIndex(ctx)
	if err != nil {
		return nil, err
	}
	return &Stats{
		PageCount:   index.PageCount,
		SourceCount: index.SourceCount,
	}, nil
}

func (s *Service) GetIndex(ctx context.Context) (*Index, error) {
	return s.store.GetIndex(ctx)
}

func (s *Service) GetLog(ctx context.Context, limit int) ([]LogEntry, error) {
	return s.store.GetLogs(ctx, limit)
}

type Stats struct {
	PageCount   int `json:"page_count"`
	SourceCount int `json:"source_count"`
}

func (s *Service) Lint(ctx context.Context, req LintRequest) (*LintResult, error) {
	// Implementation could be added later
	return &LintResult{}, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
