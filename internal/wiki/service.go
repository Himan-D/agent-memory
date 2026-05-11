package wiki

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type Service struct {
	mu       sync.RWMutex
	pages    map[string]*Page
	sources  map[string]*Source
	log      []LogEntry
	llm      llm.Provider
	llmModel string
	memSvc   MemorySearcher
}

type MemorySearcher interface {
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
	CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error)
	AddEntity(entity types.Entity) (*types.Entity, error)
}

func NewService(llmProvider llm.Provider, llmModel string, memSvc MemorySearcher) *Service {
	if llmModel == "" {
		llmModel = "gpt-4o-mini"
	}
	return &Service{
		pages:    make(map[string]*Page),
		sources:  make(map[string]*Source),
		log:      []LogEntry{},
		llm:      llmProvider,
		llmModel: llmModel,
		memSvc:   memSvc,
	}
}

func (s *Service) Ingest(ctx context.Context, req IngestRequest) (*IngestResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	s.sources[sourceID] = source

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
	s.log = append(s.log, entry)

	if len(s.log)%10 == 0 {
		s.updateIndex()
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
			Name        string `json:"name"`
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"entities"`
	}

	if err := json.Unmarshal([]byte(extractJSON(resp.Content)), &result); err != nil {
		result.Title = source.Title
		result.Content = resp.Content
		result.Links = []string{}
		result.Tags = []string{"auto-ingested"}
	}

	pageID := uuid.New().String()
	page := &Page{
		ID:        pageID,
		Title:     result.Title,
		Type:      PageTypeSummary,
		Content:   result.Content,
		Links:     result.Links,
		SourceIDs: []string{source.ID},
		Tags:      result.Tags,
		Metadata: map[string]interface{}{
			"source_title": source.Title,
			"source_id":    source.ID,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.pages[pageID] = page

	entitiesUpdated := 0
	pagesCreated := 1

	for _, entity := range result.Entities {
		existing := s.findPageByTitle(entity.Name)
		if existing != nil {
			existing.Content += fmt.Sprintf("\n\n## Update from %s\n%s", source.Title, entity.Description)
			existing.UpdatedAt = time.Now()
			existing.SourceIDs = append(existing.SourceIDs, source.ID)
			entitiesUpdated++
		} else {
			ePageID := uuid.New().String()
			ePage := &Page{
				ID:        ePageID,
				Title:     entity.Name,
				Type:      PageTypeEntity,
				Content:   fmt.Sprintf("# %s\n\n%s\n\n*Type: %s*", entity.Name, entity.Description, entity.Type),
				Links:     []string{page.Title},
				SourceIDs: []string{source.ID},
				Tags:      []string{entity.Type},
				Metadata: map[string]interface{}{
					"entity_type": entity.Type,
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			s.pages[ePageID] = ePage
			pagesCreated++

			ent := types.Entity{
				ID:         ePageID,
				Type:       entity.Type,
				Name:       entity.Name,
				Properties: map[string]interface{}{"description": entity.Description, "wiki_page_id": ePageID},
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			if _, err := s.memSvc.AddEntity(ent); err != nil {
				log.Printf("wiki: failed to create entity in graph: %v", err)
			}
		}
	}

	return page, entitiesUpdated, pagesCreated, nil
}

func (s *Service) Query(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	s.mu.RLock()
	relevantPages := s.findRelevantPages(req.Query, req.MaxContext)
	s.mu.RUnlock()

	if len(relevantPages) == 0 {
		memResults, err := s.memSvc.SearchMemories(ctx, &types.SearchRequest{
			Query: req.Query,
			Limit: 5,
		})
		if err == nil && len(memResults) > 0 {
			var contextParts []string
			for _, mr := range memResults {
				if mr.Text != "" {
					contextParts = append(contextParts, mr.Text)
				} else if mr.Metadata != nil {
					contextParts = append(contextParts, mr.Metadata.Content)
				}
			}
			relevantPages = []*Page{{
				ID:      "memory-search",
				Title:   "Memory Search Results",
				Type:    PageTypeAnalysis,
				Content: strings.Join(contextParts, "\n\n---\n\n"),
			}}
		}
	}

	var contextBuilder strings.Builder
	for i, p := range relevantPages {
		contextBuilder.WriteString(fmt.Sprintf("## %s (Type: %s)\n\n%s\n\n", p.Title, p.Type, p.Content))
		if i >= 4 {
			break
		}
	}

	answerPrompt := fmt.Sprintf(`You are a knowledgeable assistant answering based on a wiki knowledge base. 
Using the following wiki pages as context, answer the question thoroughly.

Context:
%s

Question: %s

Provide a comprehensive answer. If you reference specific facts, cite the source pages.
If information is incomplete, mention what's missing.`, contextBuilder.String(), req.Query)

	resp, err := s.llm.Complete(ctx, &llm.CompletionRequest{
		Model:       s.llmModel,
		Messages:    []llm.Message{{Role: "user", Content: answerPrompt}},
		Temperature: 0.3,
		MaxTokens:   4000,
	})
	if err != nil {
		return nil, fmt.Errorf("wiki: query: %w", err)
	}

	result := &QueryResult{
		Answer:  resp.Content,
		Sources: make([]string, 0, len(relevantPages)),
	}

	for _, p := range relevantPages {
		result.Sources = append(result.Sources, p.Title)
	}

	if req.SaveAsPage {
		pageID := uuid.New().String()
		page := &Page{
			ID:        pageID,
			Title:     fmt.Sprintf("Query: %s", truncate(req.Query, 80)),
			Type:      PageTypeAnalysis,
			Content:   fmt.Sprintf("# Query: %s\n\n## Answer\n\n%s", req.Query, resp.Content),
			Links:     result.Sources,
			SourceIDs: []string{},
			Tags:      []string{"query-result"},
			Metadata:  map[string]interface{}{"query": req.Query},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		s.mu.Lock()
		s.pages[pageID] = page
		s.log = append(s.log, LogEntry{
			Timestamp: time.Now(),
			Operation: "query",
			Target:    req.Query,
		})
		s.mu.Unlock()
		result.PageID = pageID
	}

	return result, nil
}

func (s *Service) Lint(ctx context.Context, req LintRequest) (*LintResult, error) {
	s.mu.RLock()

	checkTypes := req.CheckTypes
	if len(checkTypes) == 0 {
		checkTypes = []LintCheckType{LintContradictions, LintStaleClaims, LintOrphans, LintGaps}
	}

	result := &LintResult{}

	for _, ct := range checkTypes {
		switch ct {
		case LintContradictions:
			result.Contradictions = s.findContradictions()
		case LintStaleClaims:
			result.StaleClaims = s.findStaleClaims()
		case LintOrphans:
			result.OrphanPages = s.findOrphans()
		case LintGaps:
			result.Gaps = s.findGaps()
		}
	}

	result.Report = s.generateLintReport(result)

	s.mu.RUnlock()
	s.mu.Lock()
	s.log = append(s.log, LogEntry{
		Timestamp: time.Now(),
		Operation: "lint",
		Target:    "full-wiki",
		Details: fmt.Sprintf("Found %d contradictions, %d orphans, %d gaps",
			len(result.Contradictions), len(result.OrphanPages), len(result.Gaps)),
	})
	s.mu.Unlock()

	return result, nil
}

func (s *Service) GetPage(pageID string) (*Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.pages[pageID]
	if !ok {
		return nil, fmt.Errorf("wiki: page %s not found", pageID)
	}
	return p, nil
}

func (s *Service) ListPages(pageType PageType, limit, offset int) []*Page {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Page
	for _, p := range s.pages {
		if pageType == "" || p.Type == pageType {
			result = append(result, p)
		}
	}

	if limit <= 0 || limit > len(result) {
		limit = len(result)
	}
	if offset > len(result) {
		offset = 0
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end]
}

func (s *Service) UpdatePage(pageID string, updates map[string]interface{}) (*Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pages[pageID]
	if !ok {
		return nil, fmt.Errorf("wiki: page %s not found", pageID)
	}

	if title, ok := updates["title"].(string); ok {
		p.Title = title
	}
	if content, ok := updates["content"].(string); ok {
		p.Content = content
	}
	if links, ok := updates["links"].([]string); ok {
		p.Links = links
	}
	if tags, ok := updates["tags"].([]string); ok {
		p.Tags = tags
	}
	p.UpdatedAt = time.Now()

	return p, nil
}

func (s *Service) DeletePage(pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pages[pageID]; !ok {
		return fmt.Errorf("wiki: page %s not found", pageID)
	}
	delete(s.pages, pageID)
	return nil
}

func (s *Service) GetSource(sourceID string) (*Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	src, ok := s.sources[sourceID]
	if !ok {
		return nil, fmt.Errorf("wiki: source %s not found", sourceID)
	}
	return src, nil
}

func (s *Service) ListSources(limit, offset int) []*Source {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Source, 0, len(s.sources))
	for _, src := range s.sources {
		result = append(result, src)
	}
	return result
}

func (s *Service) GetStats() *WikiStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &WikiStats{
		TotalPages:   len(s.pages),
		TotalSources: len(s.sources),
		PagesByType:  make(map[PageType]int),
		RecentPages:  make([]*Page, 0),
	}

	for _, p := range s.pages {
		stats.PagesByType[p.Type]++
	}

	if len(s.log) > 0 {
		lastIngest := s.findLastLog("ingest")
		stats.LastIngest = lastIngest
		lastLint := s.findLastLog("lint")
		stats.LastLint = lastLint
	}

	return stats
}

func (s *Service) GetLog(limit, offset int) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.log) {
		limit = len(s.log)
	}
	if offset > len(s.log) {
		offset = 0
	}
	end := offset + limit
	if end > len(s.log) {
		end = len(s.log)
	}
	return s.log[offset:end]
}

func (s *Service) GetIndex() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var b strings.Builder
	b.WriteString("# Wiki Index\n\n")
	b.WriteString(fmt.Sprintf("*Last updated: %s*\n\n", time.Now().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("## Summary\n- Total pages: %d\n- Total sources: %d\n\n", len(s.pages), len(s.sources)))

	for _, pt := range []PageType{PageTypeSummary, PageTypeEntity, PageTypeConcept, PageTypeComparison, PageTypeTimeline, PageTypeAnalysis, PageTypeSynthesis} {
		pages := s.pagesByType(pt)
		if len(pages) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("### %s Pages (%d)\n", strings.Title(string(pt)), len(pages)))
		for _, p := range pages {
			daysAgo := int(time.Since(p.UpdatedAt).Hours() / 24)
			freshness := ""
			switch {
			case daysAgo == 0:
				freshness = " (today)"
			case daysAgo == 1:
				freshness = " (yesterday)"
			default:
				freshness = fmt.Sprintf(" (%d days ago)", daysAgo)
			}
			b.WriteString(fmt.Sprintf("- [%s](/wiki/pages/%s)%s\n", p.Title, p.ID, freshness))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (s *Service) pagesByType(pt PageType) []*Page {
	var result []*Page
	for _, p := range s.pages {
		if p.Type == pt {
			result = append(result, p)
		}
	}
	return result
}

func (s *Service) findPageByTitle(title string) *Page {
	for _, p := range s.pages {
		if strings.EqualFold(p.Title, title) {
			return p
		}
	}
	return nil
}

func (s *Service) findRelevantPages(query string, maxResults int) []*Page {
	if maxResults <= 0 {
		maxResults = 5
	}

	queryLower := strings.ToLower(query)
	keywords := strings.Fields(queryLower)

	type scored struct {
		page  *Page
		score float64
	}

	var scoredPages []scored
	for _, p := range s.pages {
		contentLower := strings.ToLower(p.Content + " " + p.Title)
		var score float64
		for _, kw := range keywords {
			if strings.Contains(contentLower, kw) {
				score += 1.0
			}
		}
		if strings.Contains(contentLower, queryLower) {
			score += 3.0
		}
		for _, tag := range p.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				score += 2.0
			}
		}
		if score > 0 {
			scoredPages = append(scoredPages, scored{page: p, score: score})
		}
	}

	for i := 0; i < len(scoredPages)-1; i++ {
		for j := i + 1; j < len(scoredPages); j++ {
			if scoredPages[j].score > scoredPages[i].score {
				scoredPages[i], scoredPages[j] = scoredPages[j], scoredPages[i]
			}
		}
	}

	result := make([]*Page, 0, maxResults)
	for i, sp := range scoredPages {
		if i >= maxResults {
			break
		}
		result = append(result, sp.page)
	}
	return result
}

func (s *Service) findOrphans() []OrphanPage {
	inbound := make(map[string]int)
	for _, p := range s.pages {
		for _, link := range p.Links {
			inbound[link]++
		}
	}

	var orphans []OrphanPage
	for _, p := range s.pages {
		if p.Type == PageTypeIndex || p.Type == PageTypeLog {
			continue
		}
		if inbound[p.Title] == 0 {
			orphans = append(orphans, OrphanPage{
				PageID:        p.ID,
				PageTitle:     p.Title,
				PageType:      p.Type,
				OutboundLinks: len(p.Links),
			})
		}
	}
	return orphans
}

func (s *Service) findContradictions() []Contradiction {
	var contradictions []Contradiction

	pages := make([]*Page, 0, len(s.pages))
	for _, p := range s.pages {
		pages = append(pages, p)
	}

	limit := len(pages)
	if limit > 20 {
		limit = 20
	}

	for i := 0; i < limit; i++ {
		for j := i + 1; j < limit; j++ {
			content1 := strings.ToLower(pages[i].Content)
			content2 := strings.ToLower(pages[j].Content)

			conflictIndicators := []string{"however", "but", "contradicts", "disputes", "refutes", "corrects"}
			for _, indicator := range conflictIndicators {
				if strings.Contains(content1, indicator) && strings.Contains(content2, indicator) {
					contradictions = append(contradictions, Contradiction{
						Page1ID:    pages[i].ID,
						Page1Title: pages[i].Title,
						Page2ID:    pages[j].ID,
						Page2Title: pages[j].Title,
						Details:    fmt.Sprintf("Both pages contain conflict indicator '%s'", indicator),
					})
					break
				}
			}
		}
	}

	return contradictions
}

func (s *Service) findStaleClaims() []StaleClaim {
	var stale []StaleClaim

	staleIndicators := []string{"currently", "as of", "recently", "latest", "now"}
	for _, p := range s.pages {
		daysSinceUpdate := time.Since(p.UpdatedAt).Hours() / 24
		contentLower := strings.ToLower(p.Content)
		for _, indicator := range staleIndicators {
			if strings.Contains(contentLower, indicator) && daysSinceUpdate > 30 {
				stale = append(stale, StaleClaim{
					PageID:    p.ID,
					PageTitle: p.Title,
					Claim:     fmt.Sprintf("Contains temporal claim '%s' not updated in %.0f days", indicator, daysSinceUpdate),
					Reason:    "Temporal claim may be outdated",
				})
				break
			}
		}
	}
	return stale
}

func (s *Service) findGaps() []InfoGap {
	var gaps []InfoGap
	gapIndicators := []string{"unknown", "unclear", "tbd", "todo", "missing", "further research", "needs investigation"}

	for _, p := range s.pages {
		var missing []string
		contentLower := strings.ToLower(p.Content)
		for _, indicator := range gapIndicators {
			if strings.Contains(contentLower, indicator) {
				missing = append(missing, indicator)
			}
		}
		if len(missing) > 0 {
			gaps = append(gaps, InfoGap{
				PageID:    p.ID,
				PageTitle: p.Title,
				Missing:   missing,
			})
		}
	}
	return gaps
}

func (s *Service) generateLintReport(result *LintResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Wiki Lint Report - %s\n\n", time.Now().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Contradictions found: %d\n", len(result.Contradictions)))
	b.WriteString(fmt.Sprintf("- Potentially stale claims: %d\n", len(result.StaleClaims)))
	b.WriteString(fmt.Sprintf("- Orphan pages: %d\n", len(result.OrphanPages)))
	b.WriteString(fmt.Sprintf("- Information gaps: %d\n\n", len(result.Gaps)))

	if len(result.Contradictions) > 0 {
		b.WriteString("## Contradictions\n\n")
		for _, c := range result.Contradictions {
			b.WriteString(fmt.Sprintf("- **%s** vs **%s**: %s\n", c.Page1Title, c.Page2Title, c.Details))
		}
		b.WriteString("\n")
	}

	if len(result.OrphanPages) > 0 {
		b.WriteString("## Orphan Pages\n\n")
		for _, o := range result.OrphanPages {
			b.WriteString(fmt.Sprintf("- **%s** (%s) - %d outbound links\n", o.PageTitle, o.PageType, o.OutboundLinks))
		}
		b.WriteString("\n")
	}

	if len(result.Gaps) > 0 {
		b.WriteString("## Information Gaps\n\n")
		for _, g := range result.Gaps {
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", g.PageTitle, strings.Join(g.Missing, ", ")))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (s *Service) updateIndex() {
	indexID := ""
	for id, p := range s.pages {
		if p.Type == PageTypeIndex {
			indexID = id
			break
		}
	}

	content := s.GetIndex()

	if indexID != "" {
		s.pages[indexID].Content = content
		s.pages[indexID].UpdatedAt = time.Now()
	} else {
		pageID := uuid.New().String()
		s.pages[pageID] = &Page{
			ID:        pageID,
			Title:     "Wiki Index",
			Type:      PageTypeIndex,
			Content:   content,
			Links:     []string{},
			Tags:      []string{"index"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}
}

func (s *Service) findLastLog(operation string) *time.Time {
	for i := len(s.log) - 1; i >= 0; i-- {
		if s.log[i].Operation == operation {
			return &s.log[i].Timestamp
		}
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return s
	}
	end := strings.LastIndex(s, "}")
	if end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}
