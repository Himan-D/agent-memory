package memory

import (
	"fmt"
	"sync"
	"time"

	"agent-memory/internal/memory/types"
)

type MessageBuffer struct {
	mu       sync.Mutex
	messages map[string][]types.Message
	maxSize  int
	timeout  time.Duration
	neo4j    interface {
		AddMessage(sessionID string, msg types.Message) error
		AddMessages(sessionID string, msgs []types.Message) error
	}
	closed chan struct{}
}

func NewMessageBuffer(maxSize int, timeout time.Duration, neo4j interface {
	AddMessage(sessionID string, msg types.Message) error
	AddMessages(sessionID string, msgs []types.Message) error
}) *MessageBuffer {
	mb := &MessageBuffer{
		messages: make(map[string][]types.Message),
		maxSize:  maxSize,
		timeout:  timeout,
		neo4j:    neo4j,
		closed:   make(chan struct{}),
	}
	go mb.flushLoop()
	return mb
}

func (mb *MessageBuffer) Close() error {
	close(mb.closed)
	return mb.FlushAll()
}

func (mb *MessageBuffer) Add(msg types.Message) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	sessionID := msg.SessionID
	mb.messages[sessionID] = append(mb.messages[sessionID], msg)

	if len(mb.messages[sessionID]) >= mb.maxSize {
		mb.flushSession(sessionID)
	}

	return nil
}

func (mb *MessageBuffer) flushLoop() {
	ticker := time.NewTicker(mb.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-mb.closed:
			mb.FlushAll()
			return
		case <-ticker.C:
			mb.FlushAll()
		}
	}
}

func (mb *MessageBuffer) FlushSession(sessionID string) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	return mb.flushSession(sessionID)
}

func (mb *MessageBuffer) flushSession(sessionID string) error {
	msgs, ok := mb.messages[sessionID]
	if !ok || len(msgs) == 0 {
		return nil
	}

	// Optimization: Batch flush messages to Neo4j in a single round-trip.
	// Expected impact: Reduces database latency from O(N) to O(1) per flush.
	if err := mb.neo4j.AddMessages(sessionID, msgs); err != nil {
		fmt.Printf("warn: buffer flush session %s failed: %v\n", sessionID, err)
		return err
	}

	mb.messages[sessionID] = nil
	delete(mb.messages, sessionID)
	return nil
}

func (mb *MessageBuffer) FlushAll() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	var lastErr error
	for sessionID := range mb.messages {
		if err := mb.flushSession(sessionID); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (mb *MessageBuffer) Len() int {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	count := 0
	for _, msgs := range mb.messages {
		count += len(msgs)
	}
	return count
}
