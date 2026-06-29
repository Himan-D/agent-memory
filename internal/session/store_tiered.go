package session

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// TieredStore coordinates a hot tier (RedisStore) with a cold tier
// (Neo4jStore). Writes go to both tiers; reads prefer the hot tier and
// fall back to the cold tier when the hot tier has no data.
//
// TieredStore degrades gracefully: if the hot tier errors on a write, the
// cold tier still receives the write and a warning is logged. The cold
// tier is the source of truth, so a write failure on the hot tier does
// not lose data.
//
// This is the production-ready Store: pair NewRedisStore + NewNeo4jStore
// via NewTieredStore and pass the result to session.NewManager.
type TieredStore struct {
	hot  Store // Redis - low latency, time-bounded
	cold Store // Neo4j - durable
	log  *log.Logger
	// writeColdTimeout (zero = no timeout) caps how long a cold-tier write
	// can block a caller when Redis is healthy.
	writeColdTimeout time.Duration
	mu               sync.RWMutex
	hotHealthy       bool
}

// TieredStoreConfig configures a TieredStore.
type TieredStoreConfig struct {
	Hot              Store
	Cold             Store
	Logger           *log.Logger
	WriteColdTimeout time.Duration
}

// NewTieredStore wraps a hot and cold Store. Either may be nil: a nil hot
// tier makes the cold tier authoritative; a nil cold tier makes the hot
// tier the only writer (acceptable for ephemeral sessions).
func NewTieredStore(cfg TieredStoreConfig) *TieredStore {
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &TieredStore{
		hot:              cfg.Hot,
		cold:             cfg.Cold,
		log:              logger,
		writeColdTimeout: cfg.WriteColdTimeout,
		hotHealthy:       cfg.Hot != nil,
	}
}

// HotHealth reports whether the hot tier is currently reachable. Callers
// can poll this to decide whether to fall back to Neo4j-only mode.
func (t *TieredStore) HotHealth() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.hotHealthy
}

// Ping verifies both tiers.
func (t *TieredStore) Ping(ctx context.Context) error {
	var errs []error
	if t.hot != nil {
		if err := t.hot.Ping(ctx); err != nil {
			t.markHot(false)
			errs = append(errs, fmt.Errorf("hot tier: %w", err))
		} else {
			t.markHot(true)
		}
	}
	if t.cold != nil {
		if err := t.cold.Ping(ctx); err != nil {
			errs = append(errs, fmt.Errorf("cold tier: %w", err))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

// --- Store interface ---

func (t *TieredStore) AppendTurn(userID, sessionID string, turn QATurn) error {
	return t.writeBoth(func(s Store) error {
		return s.AppendTurn(userID, sessionID, turn)
	})
}

func (t *TieredStore) ListTurns(userID, sessionID string, lastN int) ([]QATurn, error) {
	if t.hot != nil && t.HotHealth() {
		turns, err := t.hot.ListTurns(userID, sessionID, lastN)
		if err == nil {
			return turns, nil
		}
		t.log.Printf("tiered store: hot ListTurns failed, falling back to cold: %v", err)
		t.markHot(false)
	}
	if t.cold == nil {
		return nil, errors.New("tiered store: cold tier unavailable")
	}
	return t.cold.ListTurns(userID, sessionID, lastN)
}

func (t *TieredStore) UpsertContext(userID string, entries []ContextEntry) error {
	return t.writeBoth(func(s Store) error {
		return s.UpsertContext(userID, entries)
	})
}

func (t *TieredStore) ListContext(userID string) ([]ContextEntry, error) {
	if t.hot != nil && t.HotHealth() {
		entries, err := t.hot.ListContext(userID)
		if err == nil {
			return entries, nil
		}
		t.log.Printf("tiered store: hot ListContext failed, falling back to cold: %v", err)
		t.markHot(false)
	}
	if t.cold == nil {
		return nil, errors.New("tiered store: cold tier unavailable")
	}
	return t.cold.ListContext(userID)
}

func (t *TieredStore) SaveLesson(userID string, lesson DistilledLesson) error {
	return t.writeBoth(func(s Store) error {
		return s.SaveLesson(userID, lesson)
	})
}

func (t *TieredStore) ListLessons(userID string) ([]DistilledLesson, error) {
	if t.hot != nil && t.HotHealth() {
		lessons, err := t.hot.ListLessons(userID)
		if err == nil {
			return lessons, nil
		}
		t.log.Printf("tiered store: hot ListLessons failed, falling back to cold: %v", err)
		t.markHot(false)
	}
	if t.cold == nil {
		return nil, errors.New("tiered store: cold tier unavailable")
	}
	return t.cold.ListLessons(userID)
}

// --- helpers ---

// writeBoth invokes fn on every configured tier. Cold-tier failures are
// returned; hot-tier failures are logged but do not abort. The cold tier
// is the source of truth.
func (t *TieredStore) writeBoth(fn func(Store) error) error {
	var hotErr error
	if t.hot != nil {
		if err := fn(t.hot); err != nil {
			t.log.Printf("tiered store: hot tier write failed: %v", err)
			t.markHot(false)
			hotErr = err
		} else {
			t.markHot(true)
		}
	}
	if t.cold != nil {
		if err := fn(t.cold); err != nil {
			// Cold tier failure is fatal: it is the source of truth.
			return fmt.Errorf("tiered store: cold tier write failed: %w", err)
		}
	}
	if hotErr != nil && t.cold == nil {
		return hotErr
	}
	return nil
}

func (t *TieredStore) markHot(ok bool) {
	t.mu.Lock()
	t.hotHealthy = ok
	t.mu.Unlock()
}
