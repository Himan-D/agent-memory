package wiki

import (
	"time"
)

type PageType string

const (
	PageTypeSummary    PageType = "summary"
	PageTypeEntity     PageType = "entity"
	PageTypeConcept    PageType = "concept"
	PageTypeComparison PageType = "comparison"
	PageTypeTimeline   PageType = "timeline"
	PageTypeAnalysis   PageType = "analysis"
	PageTypeSynthesis  PageType = "synthesis"
	PageTypeLog        PageType = "log"
	PageTypeIndex      PageType = "index"
)

type Source struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	FilePath    string                 `json:"file_path"`
	ContentType string                 `json:"content_type"`
	Content     string                 `json:"content,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	AddedAt     time.Time              `json:"added_at"`
}

type Page struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	Type      PageType               `json:"type"`
	Content   string                 `json:"content"`
	Links     []string               `json:"links,omitempty"`
	SourceIDs []string               `json:"source_ids,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"`
	Target    string    `json:"target"`
	Details   string    `json:"details,omitempty"`
}

type IngestRequest struct {
	Content     string                 `json:"content"`
	Title       string                 `json:"title,omitempty"`
	ContentType string                 `json:"content_type,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type IngestResult struct {
	Source          *Source `json:"source"`
	SummaryPage     *Page   `json:"summary_page"`
	EntitiesUpdated int     `json:"entities_updated"`
	PagesCreated    int     `json:"pages_created"`
}

type QueryRequest struct {
	Query      string `json:"query"`
	Format     string `json:"format,omitempty"`
	MaxContext int    `json:"max_context,omitempty"`
	SaveAsPage bool   `json:"save_as_page,omitempty"`
}

type QueryResult struct {
	Answer      string   `json:"answer"`
	Sources     []string `json:"sources,omitempty"`
	MissingInfo []string `json:"missing_info,omitempty"`
	PageID      string   `json:"page_id,omitempty"`
}

type LintCheckType string

const (
	LintContradictions LintCheckType = "contradictions"
	LintStaleClaims    LintCheckType = "stale_claims"
	LintOrphans        LintCheckType = "orphans"
	LintGaps           LintCheckType = "gaps"
)

type LintRequest struct {
	CheckTypes []LintCheckType `json:"check_types,omitempty"`
}

type LintResult struct {
	Contradictions []Contradiction `json:"contradictions,omitempty"`
	StaleClaims    []StaleClaim    `json:"stale_claims,omitempty"`
	OrphanPages    []OrphanPage    `json:"orphan_pages,omitempty"`
	Gaps           []InfoGap       `json:"gaps,omitempty"`
	Report         string          `json:"report,omitempty"`
}

type Contradiction struct {
	Page1ID    string `json:"page1_id"`
	Page1Title string `json:"page1_title"`
	Page2ID    string `json:"page2_id"`
	Page2Title string `json:"page2_title"`
	Details    string `json:"details"`
}

type StaleClaim struct {
	PageID    string `json:"page_id"`
	PageTitle string `json:"page_title"`
	Claim     string `json:"claim"`
	Reason    string `json:"reason"`
}

type OrphanPage struct {
	PageID        string   `json:"page_id"`
	PageTitle     string   `json:"page_title"`
	PageType      PageType `json:"page_type"`
	OutboundLinks int      `json:"outbound_links"`
}

type InfoGap struct {
	PageID    string   `json:"page_id"`
	PageTitle string   `json:"page_title"`
	Missing   []string `json:"missing"`
}

type WikiStats struct {
	TotalPages   int              `json:"total_pages"`
	TotalSources int              `json:"total_sources"`
	PagesByType  map[PageType]int `json:"pages_by_type"`
	RecentPages  []*Page          `json:"recent_pages"`
	LastIngest   *time.Time       `json:"last_ingest,omitempty"`
	LastLint     *time.Time       `json:"last_lint,omitempty"`
}
