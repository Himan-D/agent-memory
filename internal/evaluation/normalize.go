package evaluation

import "strings"

// NormalizeDatasetScope aligns question.session_id with memory.user_id so
// user-scoped vector search can retrieve the right corpus.
//
// LoCoMo already uses sample_id for both. Packaged BEAM/LongMemEval fixtures
// often use chat session ids (s001) while memories live under a different
// user_id (demo-user / u001 / benchmark-user) — without this remap, Hit@k is 0.
func NormalizeDatasetScope(ds *BenchmarkDataset) {
	if ds == nil || len(ds.Memories) == 0 {
		return
	}

	memUser := make(map[string]string, len(ds.Memories))
	uniqueUsers := map[string]struct{}{}
	for _, m := range ds.Memories {
		uid := strings.TrimSpace(m.UserID)
		if uid == "" {
			uid = "benchmark-user"
		}
		memUser[m.ID] = uid
		uniqueUsers[uid] = struct{}{}
	}

	soleUser := ""
	if len(uniqueUsers) == 1 {
		for u := range uniqueUsers {
			soleUser = u
		}
	}

	for i := range ds.Questions {
		q := &ds.Questions[i]
		if mid := strings.TrimSpace(q.MemoryID); mid != "" {
			if uid, ok := memUser[mid]; ok && uid != "" {
				q.SessionID = uid
				continue
			}
		}
		if soleUser != "" {
			q.SessionID = soleUser
			continue
		}
		// Multi-user corpus without memory_id: keep session_id if it already
		// matches a known user; otherwise leave unchanged (LoCoMo path).
		if _, ok := uniqueUsers[q.SessionID]; ok {
			continue
		}
	}
}
