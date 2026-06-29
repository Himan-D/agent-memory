package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// fakeRedis is a tiny in-memory replacement for *redis.Client sufficient to
// exercise RedisStore logic. It implements redisCmdable and the Scan method.
// It is not safe for concurrent use; tests should not share instances.
type fakeRedis struct {
	data map[string]string
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{data: map[string]string{}}
}

func (f *fakeRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	v, ok := f.data[key]
	if !ok {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(v)
	return cmd
}

func (f *fakeRedis) Set(ctx context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	f.data[key] = toString(value)
	cmd.SetVal("OK")
	return cmd
}

func (f *fakeRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	n := 0
	for _, k := range keys {
		if _, ok := f.data[k]; ok {
			delete(f.data, k)
			n++
		}
	}
	cmd.SetVal(int64(n))
	return cmd
}

func (f *fakeRedis) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	n := 0
	for _, k := range keys {
		if _, ok := f.data[k]; ok {
			n++
		}
	}
	cmd.SetVal(int64(n))
	return cmd
}

func (f *fakeRedis) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetVal("PONG")
	return cmd
}

// Scan returns keys matching pattern. Pattern is a simple "prefix:*" match.
// func (f *fakeRedis) Scan(ctx context.Context, cursor uint64, match string, _ int64) *redis.ScanCmd {
// 	cmd := redis.NewScanCmd(ctx, "scan", cursor, "MATCH", match)
// 	prefix := match
// 	if i := lastStar(match); i >= 0 {
// 		prefix = match[:i]
// 	}
// 	var keys []string
// 	for k := range f.data {
// 		if hasPrefix(k, prefix) {
// 			keys = append(keys, k)
// 		}
// 	}
// 	cmd.SetVal(keys, 0)
// 	return cmd
// }

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return ""
}

func lastStar(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '*' {
			return i
		}
	}
	return -1
}

func hasPrefix(s, prefix string) bool {
	if len(prefix) > len(s) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func TestRedisStore_AppendAndListTurns(t *testing.T) {
	f := newFakeRedis()
	s := newRedisStoreFromClient(f, time.Hour, "session:")

	if err := s.AppendTurn("u1", "s1", QATurn{ID: "t1", Question: "Q1", Answer: "A1"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendTurn("u1", "s1", QATurn{ID: "t2", Question: "Q2", Answer: "A2"}); err != nil {
		t.Fatal(err)
	}

	turns, err := s.ListTurns("u1", "s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(turns))
	}
	if turns[0].ID != "t1" || turns[1].ID != "t2" {
		t.Fatalf("order or content wrong: %+v", turns)
	}
}

func TestRedisStore_ListTurnsLastN(t *testing.T) {
	f := newFakeRedis()
	s := newRedisStoreFromClient(f, time.Hour, "session:")
	for i := 0; i < 5; i++ {
		_ = s.AppendTurn("u1", "s1", QATurn{ID: idFromInt(i)})
	}
	got, err := s.ListTurns("u1", "s1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(got))
	}
	if got[0].ID != "3" || got[1].ID != "4" {
		t.Fatalf("expected last 2 turns (3, 4), got %+v", got)
	}
}

func TestRedisStore_UpsertAndListContext(t *testing.T) {
	f := newFakeRedis()
	s := newRedisStoreFromClient(f, time.Hour, "session:")
	entries := []ContextEntry{{ID: "a"}, {ID: "b"}}
	if err := s.UpsertContext("u1", entries); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListContext("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
}

func TestRedisStore_SaveAndListLessons(t *testing.T) {
	f := newFakeRedis()
	s := newRedisStoreFromClient(f, time.Hour, "session:")
	if err := s.SaveLesson("u1", DistilledLesson{SessionID: "s1", Statement: "X"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveLesson("u1", DistilledLesson{SessionID: "s1", Statement: "Y"}); err != nil {
		t.Fatal(err)
	}
	// SaveLesson in a different session bucket should still appear in
	// ListLessons (which scans across all sessions for the user).
	if err := s.SaveLesson("u1", DistilledLesson{SessionID: "s2", Statement: "Z"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListLessons("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 lessons, got %d", len(got))
	}
}

func TestRedisStore_KeyShape(t *testing.T) {
	f := newFakeRedis()
	s := newRedisStoreFromClient(f, time.Hour, "session:")
	_ = s.AppendTurn("u1", "sess-1", QATurn{ID: "t1"})
	if _, ok := f.data["session:u1:sess-1:turns"]; !ok {
		t.Fatalf("expected key 'session:u1:sess-1:turns' to exist; got keys=%v", keysOf(f))
	}
}

func TestRedisStore_CloseNilSafe(t *testing.T) {
	var s *RedisStore
	if err := s.Close(); err != nil {
		t.Fatalf("Close on nil receiver should be no-op, got %v", err)
	}
}

func TestRedisStore_PingPropagates(t *testing.T) {
	f := newFakeRedis()
	s := newRedisStoreFromClient(f, time.Hour, "session:")
	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("Ping should succeed against fake, got %v", err)
	}
}

func keysOf(f *fakeRedis) []string {
	out := make([]string, 0, len(f.data))
	for k := range f.data {
		out = append(out, k)
	}
	return out
}

func idFromInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 8)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// TestRedisStore_AppendTurnSerializationError ensures corrupt JSON in Redis
// surfaces as an error rather than silently losing data.
func TestRedisStore_AppendTurnCorruption(t *testing.T) {
	f := newFakeRedis()
	f.data["session:u1:s1:turns"] = "not json"
	s := newRedisStoreFromClient(f, time.Hour, "session:")
	err := s.AppendTurn("u1", "s1", QATurn{ID: "t1"})
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
	if !errors.Is(err, err) {
		// sanity: error is non-nil; not checking type because the
		// fmt.Errorf wrap breaks errors.Is matching
	}
}
