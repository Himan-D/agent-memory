package recommendation

import (
	"context"
)

// QueryHydrator enriches the query context with user/agent data before sourcing candidates.
type QueryHydrator interface {
	Name() string
	Hydrate(ctx context.Context, query *QueryContext) error
}

// Source fetches candidate memory IDs from a data source.
type Source interface {
	Name() string
	Fetch(ctx context.Context, query *QueryContext) ([]string, error)
}

// Hydrator enriches a single candidate with additional data.
type Hydrator interface {
	Name() string
	Hydrate(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error
}

// Filter removes candidates that shouldn't be recommended.
type Filter interface {
	Name() string
	Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error)
}

// Scorer computes or adjusts the score for a candidate.
type Scorer interface {
	Name() string
	Score(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error
}

// Selector sorts candidates and selects the top K.
type Selector interface {
	Name() string
	Select(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error)
}

// SideEffect runs async operations after the pipeline completes (caching, logging, etc.).
type SideEffect interface {
	Name() string
	Execute(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) error
}
