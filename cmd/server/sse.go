package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type sseEvent struct {
	ID    string      `json:"id"`
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	Time  time.Time   `json:"timestamp"`
}

type sseSubscriber struct {
	tenantID string
	ch       chan sseEvent
	done     chan struct{}
}

type sseHub struct {
	mu          sync.RWMutex
	subscribers map[string]*sseSubscriber
}

func newSSEHub() *sseHub {
	return &sseHub{
		subscribers: make(map[string]*sseSubscriber),
	}
}

func (h *sseHub) subscribe(tenantID string) (string, *sseSubscriber) {
	key := fmt.Sprintf("%s-%d", tenantID, time.Now().UnixNano())
	sub := &sseSubscriber{
		tenantID: tenantID,
		ch:       make(chan sseEvent, 64),
		done:     make(chan struct{}),
	}
	h.mu.Lock()
	h.subscribers[key] = sub
	h.mu.Unlock()
	return key, sub
}

func (h *sseHub) unsubscribe(key string) {
	h.mu.Lock()
	if sub, ok := h.subscribers[key]; ok {
		select {
		case <-sub.done:
		default:
			close(sub.done)
		}
		delete(h.subscribers, key)
	}
	h.mu.Unlock()
}

func (h *sseHub) broadcast(tenantID string, evt sseEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for k, sub := range h.subscribers {
		if sub.tenantID == tenantID || sub.tenantID == "*" {
			select {
			case sub.ch <- evt:
			default:
				go h.unsubscribe(k)
			}
		}
	}
}

func (s *APIServer) sseHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	tenantID := getTenantID(r)
	if token := r.URL.Query().Get("token"); token != "" {
		if session, valid := s.sessionStore.ValidateToken(token); valid {
			tenantID = session.UserID
		}
	}
	if tenantID == "" {
		tenantID = "default"
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	key, sub := s.sseHub.subscribe(tenantID)
	defer s.sseHub.unsubscribe(key)

	fmt.Fprintf(w, "event: ping\ndata: {\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-sub.done:
			return
		case evt := <-sub.ch:
			data, err := json.Marshal(evt.Data)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\nid: %s\n\n", evt.Event, data, evt.ID)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, "event: ping\ndata: {\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
			flusher.Flush()
		}
	}
}

func (s *APIServer) emitSSE(tenantID, eventType string, data interface{}) {
	if s.sseHub == nil {
		return
	}
	s.sseHub.broadcast(tenantID, sseEvent{
		ID:    fmt.Sprintf("%d", time.Now().UnixNano()),
		Event: eventType,
		Data:  data,
		Time:  time.Now(),
	})
}

func (s *APIServer) startSSECleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			s.sseHub.mu.Lock()
			for k, sub := range s.sseHub.subscribers {
				select {
				case <-sub.done:
					delete(s.sseHub.subscribers, k)
				default:
				}
			}
			s.sseHub.mu.Unlock()
		}
	}()
}
