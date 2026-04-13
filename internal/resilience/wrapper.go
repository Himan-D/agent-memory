package resilience

import (
	"context"
	"log"
	"time"

	"agent-memory/internal/memory"
)

type ResilientStore struct {
	graph   GraphStore
	vector  VectorStore
	cbGraph *CircuitBreaker
	cbVec   *CircuitBreaker
}

type GraphStore interface {
	Close() error
	Ping(ctx context.Context) error
}

type VectorStore interface {
	Ping(ctx context.Context) error
	Close() error
}

func NewResilientStore(graph memory.GraphStore, vector memory.VectorStore) *ResilientStore {
	rs := &ResilientStore{
		graph:  graph,
		vector: vector,
		cbGraph: NewCircuitBreaker(CircuitBreakerConfig{
			Name:             "neo4j",
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
			OnStateChange: func(name string, from, to State) {
				log.Printf("[CIRCUIT-BREAKER] %s: %s -> %s", name, from, to)
			},
		}),
		cbVec: NewCircuitBreaker(CircuitBreakerConfig{
			Name:             "qdrant",
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
			OnStateChange: func(name string, from, to State) {
				log.Printf("[CIRCUIT-BREAKER] %s: %s -> %s", name, from, to)
			},
		}),
	}

	return rs
}

func (rs *ResilientStore) GraphHealth(ctx context.Context) error {
	return rs.cbGraph.Execute(ctx, func() error {
		return rs.graph.Ping(ctx)
	})
}

func (rs *ResilientStore) VectorHealth(ctx context.Context) error {
	return rs.cbVec.Execute(ctx, func() error {
		return rs.vector.Ping(ctx)
	})
}

func (rs *ResilientStore) IsHealthy(ctx context.Context) bool {
	graphErr := rs.GraphHealth(ctx)
	vectorErr := rs.VectorHealth(ctx)

	return graphErr == nil && vectorErr == nil
}

func (rs *ResilientStore) GetCircuitBreakerState() (graph, vector State) {
	return rs.cbGraph.State(), rs.cbVec.State()
}
