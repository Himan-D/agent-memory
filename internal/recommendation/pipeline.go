package recommendation

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

// MemoryCandidate represents a candidate memory in the recommendation pipeline.
type MemoryCandidate struct {
	ID        string
	Score     float64
	Memory    interface{} // *types.Memory (stored as interface to avoid import cycle)
	Metadata  map[string]interface{}
	Errors    []error
	StartTime time.Time
}

// NewMemoryCandidate creates a new candidate with the given ID.
func NewMemoryCandidate(id string) *MemoryCandidate {
	return &MemoryCandidate{
		ID:       id,
		Metadata: make(map[string]interface{}),
	}
}

// AddError records an error on this candidate without failing the pipeline.
func (c *MemoryCandidate) AddError(err error) {
	c.Errors = append(c.Errors, err)
}

// HasErrors returns true if any errors were recorded.
func (c *MemoryCandidate) HasErrors() bool {
	return len(c.Errors) > 0
}

// QueryContext holds the user/agent context for recommendation.
type QueryContext struct {
	UserID            string
	AgentID           string
	EngagementHistory []EngagementRecord
	FollowingIDs      []string
	BlockedIDs        []string
	MutedKeywords     []string
	TopicIDs          []string
	IPHash            string
	SeenIDs           []string
	ServedIDs         []string
	Metadata          map[string]interface{}
}

// EngagementRecord represents a past interaction with a memory.
type EngagementRecord struct {
	MemoryID    string
	Action      string // "view", "use", "reference", "derive", "skip", "block"
	Timestamp   time.Time
	DwellMillis int64
}

// PipelineConfig controls pipeline execution behavior.
type PipelineConfig struct {
	MaxCandidates  int
	MaxConcurrent  int
	Timeout        time.Duration
	ErrorPolicy    string // "continue", "fail-fast", "log"
	EnableParallel bool
	LogLevel       string
}

// DefaultPipelineConfig returns sensible defaults.
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		MaxCandidates:  150,
		MaxConcurrent:  8,
		Timeout:        5 * time.Second,
		ErrorPolicy:    "continue",
		EnableParallel: true,
		LogLevel:       "info",
	}
}

// PipelineStage is the common interface for all pipeline stages.
type PipelineStage interface {
	Name() string
	Execute(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error)
}

// Pipeline represents the full recommendation pipeline.
type Pipeline struct {
	config         *PipelineConfig
	queryHydrators []QueryHydrator
	sources        []Source
	hydrators      []Hydrator
	filters        []Filter
	scorers        []Scorer
	selector       Selector
	sideEffects    []SideEffect
}

// NewPipeline creates a new recommendation pipeline.
func NewPipeline(config *PipelineConfig) *Pipeline {
	if config == nil {
		config = DefaultPipelineConfig()
	}
	return &Pipeline{
		config: config,
	}
}

// WithQueryHydrators adds query hydration stages.
func (p *Pipeline) WithQueryHydrators(hydrators ...QueryHydrator) *Pipeline {
	p.queryHydrators = append(p.queryHydrators, hydrators...)
	return p
}

// WithSources adds candidate sources.
func (p *Pipeline) WithSources(sources ...Source) *Pipeline {
	p.sources = append(p.sources, sources...)
	return p
}

// WithHydrators adds candidate hydration stages.
func (p *Pipeline) WithHydrators(hydrators ...Hydrator) *Pipeline {
	p.hydrators = append(p.hydrators, hydrators...)
	return p
}

// WithFilters adds candidate filters.
func (p *Pipeline) WithFilters(filters ...Filter) *Pipeline {
	p.filters = append(p.filters, filters...)
	return p
}

// WithScorers adds scoring stages.
func (p *Pipeline) WithScorers(scorers ...Scorer) *Pipeline {
	p.scorers = append(p.scorers, scorers...)
	return p
}

// WithSelector sets the selection stage.
func (p *Pipeline) WithSelector(selector Selector) *Pipeline {
	p.selector = selector
	return p
}

// WithSideEffects adds side effect stages.
func (p *Pipeline) WithSideEffects(effects ...SideEffect) *Pipeline {
	p.sideEffects = append(p.sideEffects, effects...)
	return p
}

// Execute runs the full pipeline and returns ranked candidates.
func (p *Pipeline) Execute(ctx context.Context, query *QueryContext) ([]*MemoryCandidate, error) {
	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	start := time.Now()

	// Stage 1: Query Hydration
	if err := p.executeQueryHydrators(ctx, query); err != nil {
		return nil, fmt.Errorf("query hydration: %w", err)
	}

	// Stage 2: Candidate Sourcing (parallel)
	candidates, err := p.executeSources(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("candidate sourcing: %w", err)
	}

	// Stage 3: Candidate Hydration
	if err := p.executeHydrators(ctx, query, candidates); err != nil {
		return nil, fmt.Errorf("candidate hydration: %w", err)
	}

	// Stage 4: Pre-Scoring Filters
	candidates, err = p.executeFilters(ctx, query, candidates)
	if err != nil {
		return nil, fmt.Errorf("filtering: %w", err)
	}

	// Stage 5: Scoring
	if err := p.executeScorers(ctx, query, candidates); err != nil {
		return nil, fmt.Errorf("scoring: %w", err)
	}

	// Stage 6: Selection
	if p.selector != nil {
		candidates, err = p.selector.Select(ctx, query, candidates)
		if err != nil {
			return nil, fmt.Errorf("selection: %w", err)
		}
	}

	// Stage 7: Side Effects (async, non-blocking)
	p.executeSideEffectsAsync(ctx, query, candidates)

	log.Printf("[recommendation] Pipeline completed in %v: %d candidates", time.Since(start), len(candidates))
	return candidates, nil
}

func (p *Pipeline) executeQueryHydrators(ctx context.Context, query *QueryContext) error {
	for _, h := range p.queryHydrators {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := h.Hydrate(ctx, query); err != nil {
			if p.config.ErrorPolicy == "fail-fast" {
				return fmt.Errorf("%s: %w", h.Name(), err)
			}
			log.Printf("[recommendation] Query hydrator %s error: %v", h.Name(), err)
		}
	}
	return nil
}

func (p *Pipeline) executeSources(ctx context.Context, query *QueryContext) ([]*MemoryCandidate, error) {
	var (
		mu     sync.Mutex
		allIDs []string
		idSet  = make(map[string]bool)
	)

	runSource := func(source Source) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[recommendation] Source %s panicked: %v\n%s", source.Name(), r, debug.Stack())
			}
		}()

		candidateIDs, err := source.Fetch(ctx, query)
		if err != nil {
			if p.config.ErrorPolicy == "fail-fast" {
				log.Printf("[recommendation] Source %s failed: %v", source.Name(), err)
				return
			}
			log.Printf("[recommendation] Source %s error: %v", source.Name(), err)
			return
		}

		mu.Lock()
		defer mu.Unlock()
		for _, id := range candidateIDs {
			if !idSet[id] && len(idSet) < p.config.MaxCandidates {
				idSet[id] = true
				allIDs = append(allIDs, id)
			}
		}
	}

	if p.config.EnableParallel {
		var wg sync.WaitGroup
		sem := make(chan struct{}, p.config.MaxConcurrent)
		for _, source := range p.sources {
			wg.Add(1)
			go func(s Source) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				runSource(s)
			}(source)
		}
		wg.Wait()
	} else {
		for _, source := range p.sources {
			runSource(source)
		}
	}

	candidates := make([]*MemoryCandidate, 0, len(allIDs))
	for _, id := range allIDs {
		candidates = append(candidates, NewMemoryCandidate(id))
	}

	return candidates, nil
}

func (p *Pipeline) executeHydrators(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) error {
	for _, h := range p.hydrators {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		runHydrator := func(c *MemoryCandidate) error {
			defer func() {
				if r := recover(); r != nil {
					c.AddError(fmt.Errorf("hydrator %s panicked: %v", h.Name(), r))
				}
			}()
			return h.Hydrate(ctx, query, c)
		}

		if p.config.EnableParallel {
			var wg sync.WaitGroup
			sem := make(chan struct{}, p.config.MaxConcurrent)
			for _, c := range candidates {
				wg.Add(1)
				go func(candidate *MemoryCandidate) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					if err := runHydrator(candidate); err != nil {
						if p.config.ErrorPolicy == "fail-fast" {
							log.Printf("[recommendation] Hydrator %s failed on %s: %v", h.Name(), candidate.ID, err)
						}
						candidate.AddError(err)
					}
				}(c)
			}
			wg.Wait()
		} else {
			for _, c := range candidates {
				if err := runHydrator(c); err != nil {
					c.AddError(err)
				}
			}
		}
	}
	return nil
}

func (p *Pipeline) executeFilters(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	remaining := candidates

	for _, f := range p.filters {
		select {
		case <-ctx.Done():
			return remaining, ctx.Err()
		default:
		}

		var err error
		remaining, err = f.Filter(ctx, query, remaining)
		if err != nil {
			if p.config.ErrorPolicy == "fail-fast" {
				return nil, fmt.Errorf("%s: %w", f.Name(), err)
			}
			log.Printf("[recommendation] Filter %s error: %v", f.Name(), err)
		}
	}

	return remaining, nil
}

func (p *Pipeline) executeScorers(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) error {
	for _, s := range p.scorers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		runScorer := func(c *MemoryCandidate) error {
			defer func() {
				if r := recover(); r != nil {
					c.AddError(fmt.Errorf("scorer %s panicked: %v", s.Name(), r))
				}
			}()
			return s.Score(ctx, query, c)
		}

		if p.config.EnableParallel {
			var wg sync.WaitGroup
			sem := make(chan struct{}, p.config.MaxConcurrent)
			for _, c := range candidates {
				wg.Add(1)
				go func(candidate *MemoryCandidate) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					if err := runScorer(candidate); err != nil {
						candidate.AddError(err)
					}
				}(c)
			}
			wg.Wait()
		} else {
			for _, c := range candidates {
				if err := runScorer(c); err != nil {
					c.AddError(err)
				}
			}
		}
	}
	return nil
}

func (p *Pipeline) executeSideEffectsAsync(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) {
	for _, effect := range p.sideEffects {
		go func(e SideEffect) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[recommendation] Side effect %s panicked: %v", e.Name(), r)
				}
			}()
			if err := e.Execute(ctx, query, candidates); err != nil {
				log.Printf("[recommendation] Side effect %s error: %v", e.Name(), err)
			}
		}(effect)
	}
}
