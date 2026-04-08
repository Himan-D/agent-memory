package memory

import (
	"sync"
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

type mockNeo4j struct {
	messages map[string][]types.Message
	mu       sync.Mutex
}

func (m *mockNeo4j) AddMessage(sessionID string, msg types.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[sessionID] = append(m.messages[sessionID], msg)
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
