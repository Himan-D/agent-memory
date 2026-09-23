package memory

import (
	"sync"
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

type mockNeo4j struct {
	messages      map[string][]types.Message
	addMsgCalls   int
	addMsgsCalls  int
	lastBatchSize int
	mu            sync.Mutex
}

func (m *mockNeo4j) AddMessage(sessionID string, msg types.Message) error {
	m.mu.Lock()
	m.addMsgCalls++
	m.mu.Unlock()
	return m.AddMessages(sessionID, []types.Message{msg})
}

func (m *mockNeo4j) AddMessages(sessionID string, msgs []types.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addMsgsCalls++
	m.lastBatchSize = len(msgs)
	m.messages[sessionID] = append(m.messages[sessionID], msgs...)
	return nil
}

func TestMessageBuffer_Add(t *testing.T) {
	mock := &mockNeo4j{messages: make(map[string][]types.Message)}
	buf := NewMessageBuffer(10, time.Hour, mock)

	msg := types.Message{
		ID:        "test-1",
		SessionID: "session-1",
		Role:      "user",
		Content:   "Hello",
	}

	if err := buf.Add(msg); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if buf.Len() != 1 {
		t.Errorf("expected buffer length 1, got %d", buf.Len())
	}
}

func TestMessageBuffer_FlushOnSize(t *testing.T) {
	mock := &mockNeo4j{messages: make(map[string][]types.Message)}
	buf := NewMessageBuffer(2, time.Hour, mock)

	buf.Add(types.Message{ID: "1", SessionID: "s1"})
	buf.Add(types.Message{ID: "2", SessionID: "s1"})

	time.Sleep(100 * time.Millisecond)

	if buf.Len() != 0 {
		t.Errorf("expected buffer to flush, got length %d", buf.Len())
	}

	if len(mock.messages["s1"]) != 2 {
		t.Errorf("expected 2 messages in mock, got %d", len(mock.messages["s1"]))
	}
}

func TestMessageBuffer_MultipleSessions(t *testing.T) {
	mock := &mockNeo4j{messages: make(map[string][]types.Message)}
	buf := NewMessageBuffer(10, time.Hour, mock)

	buf.Add(types.Message{SessionID: "s1", Content: "msg1"})
	buf.Add(types.Message{SessionID: "s2", Content: "msg2"})
	buf.Add(types.Message{SessionID: "s1", Content: "msg3"})

	if buf.Len() != 3 {
		t.Errorf("expected 3 messages, got %d", buf.Len())
	}

	buf.FlushAll()

	if len(mock.messages["s1"]) != 2 {
		t.Errorf("s1 should have 2 messages, got %d", len(mock.messages["s1"]))
	}
	if len(mock.messages["s2"]) != 1 {
		t.Errorf("s2 should have 1 message, got %d", len(mock.messages["s2"]))
	}
}

func TestMessageBuffer_Close(t *testing.T) {
	mock := &mockNeo4j{messages: make(map[string][]types.Message)}
	buf := NewMessageBuffer(10, time.Hour, mock)

	buf.Add(types.Message{SessionID: "s1", Content: "test"})
	buf.Add(types.Message{SessionID: "s1", Content: "test2"})

	if err := buf.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("buffer should be empty after close, got %d", buf.Len())
	}
}

func TestMessageBuffer_BatchFlush(t *testing.T) {
	mock := &mockNeo4j{messages: make(map[string][]types.Message)}
	// Set maxSize to 3
	buf := NewMessageBuffer(3, time.Hour, mock)

	buf.Add(types.Message{ID: "m1", SessionID: "s1", Content: "c1"})
	buf.Add(types.Message{ID: "m2", SessionID: "s1", Content: "c2"})
	buf.Add(types.Message{ID: "m3", SessionID: "s1", Content: "c3"})

	// Wait for async flush if needed, but buffer.Add calls flushSession synchronously when maxSize is reached.
	// Actually, in buffer.go:
	// if len(mb.messages[sessionID]) >= mb.maxSize {
	//     mb.flushSession(sessionID)
	// }

	if mock.addMsgsCalls != 1 {
		t.Errorf("expected 1 call to AddMessages, got %d", mock.addMsgsCalls)
	}

	if mock.lastBatchSize != 3 {
		t.Errorf("expected batch size 3, got %d", mock.lastBatchSize)
	}

	if len(mock.messages["s1"]) != 3 {
		t.Errorf("expected 3 messages in mock, got %d", len(mock.messages["s1"]))
	}
}

func BenchmarkMessageBuffer_Add(b *testing.B) {
	mock := &mockNeo4j{messages: make(map[string][]types.Message)}
	buf := NewMessageBuffer(10000, time.Hour, mock)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Add(types.Message{
			ID:        "msg-" + string(rune(i)),
			SessionID: "session-1",
			Content:   "test content",
		})
	}
}
