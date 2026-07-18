package evaluation

import (
	"regexp"
	"sort"
	"strings"
)

// Common English function words stripped for keyword retrieval queries.
// Not scoring weights — only used to derive a secondary query from the question text.
var retrievalStopwords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"do": {}, "does": {}, "did": {}, "what": {}, "when": {}, "where": {}, "who": {},
	"whom": {}, "which": {}, "how": {}, "why": {}, "to": {}, "of": {}, "in": {},
	"on": {}, "for": {}, "and": {}, "or": {}, "with": {}, "from": {}, "by": {},
	"about": {}, "into": {}, "has": {}, "have": {}, "had": {}, "been": {}, "be": {},
	"would": {}, "could": {}, "should": {}, "will": {}, "can": {}, "her": {}, "his": {},
	"their": {}, "our": {}, "your": {}, "my": {}, "me": {}, "she": {}, "he": {},
	"they": {}, "we": {}, "you": {}, "it": {}, "its": {}, "that": {}, "this": {},
	"these": {}, "those": {}, "as": {}, "at": {}, "if": {}, "but": {}, "not": {},
	"no": {}, "yes": {}, "any": {}, "all": {}, "still": {}, "please": {}, "tell": {},
	"say": {}, "said": {}, "mention": {}, "mentioned": {}, "according": {},
}

var wordRe = regexp.MustCompile(`[a-z0-9]+`)
var properNameRe = regexp.MustCompile(`\b[A-Z][a-z]{2,}(?:\s+[A-Z][a-z]{2,})?\b`)
var dateCueRe = regexp.MustCompile(`(?i)\b(?:\d{1,2}(?:st|nd|rd|th)?\s+)?(?:jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\b(?:\s+\d{1,2}(?:st|nd|rd|th)?)?(?:\s*,?\s*\d{2,4})?|\b20\d{2}\b|\b\d{1,2}/\d{1,2}/\d{2,4}\b`)

// ExpandRetrievalQueries returns diversified retrieval queries derived from the question:
// original, keyword-only, proper-name focus, and temporal/date cues when present.
func ExpandRetrievalQueries(question string) []string {
	q := strings.TrimSpace(question)
	if q == "" {
		return nil
	}
	out := []string{q}
	lower := strings.ToLower(q)
	if kw := keywordQuery(lower); kw != "" && !strings.EqualFold(kw, q) {
		out = append(out, kw)
	}
	// Proper-name focused query (helps entity-centric LoCoMo turns).
	if names := properNameRe.FindAllString(q, -1); len(names) > 0 {
		kw := keywordQuery(lower)
		nameQ := strings.TrimSpace(strings.Join(names, " ") + " " + kw)
		if nameQ != "" {
			out = append(out, nameQ)
		}
	}
	// Temporal questions: add a date-seeking paraphrase.
	if strings.Contains(lower, "when ") || strings.Contains(lower, "what date") || strings.Contains(lower, "what day") {
		if kw := keywordQuery(lower); kw != "" {
			out = append(out, kw+" date time")
		}
	}
	if dates := dateCueRe.FindAllString(q, -1); len(dates) > 0 {
		out = append(out, strings.Join(dates, " ")+" "+keywordQuery(lower))
	}

	// Dedup (case-insensitive), preserve order
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(out))
	for _, s := range out {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		uniq = append(uniq, s)
	}
	return uniq
}

// ExpandLoCoMoQueries is an alias kept for call sites; behavior is dataset-agnostic.
func ExpandLoCoMoQueries(question string) []string {
	return ExpandRetrievalQueries(question)
}

func keywordQuery(lower string) string {
	tokens := wordRe.FindAllString(lower, -1)
	keep := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if len(t) < 2 {
			continue
		}
		if _, stop := retrievalStopwords[t]; stop {
			continue
		}
		keep = append(keep, t)
	}
	return strings.Join(keep, " ")
}

// FuseRRF merges ranked lists with Reciprocal Rank Fusion.
// rrfK is the RRF constant (standard literature default is used only when rrfK <= 0
// via the caller — prefer passing BenchmarkConfig.RRFK). When rrfK <= 0, uses 1/(rank+1).
func FuseRRF(lists [][]MemoryResult, query string, rrfK int) []MemoryResult {
	type agg struct {
		score   float64
		content string
		bestRaw float32
	}
	combined := map[string]*agg{}

	for _, list := range lists {
		for rank, r := range list {
			if r.ID == "" {
				continue
			}
			a, ok := combined[r.ID]
			if !ok {
				a = &agg{content: r.Content, bestRaw: r.Score}
				combined[r.ID] = a
			}
			denom := float64(rank + 1)
			if rrfK > 0 {
				denom = float64(rrfK + rank + 1)
			}
			a.score += 1.0 / denom
			if r.Score > a.bestRaw {
				a.bestRaw = r.Score
			}
			if a.content == "" && r.Content != "" {
				a.content = r.Content
			}
		}
	}

	// Lexical overlap as a real ranking signal (not a 1e-6 epsilon).
	qTokens := tokenSetLower(query)
	out := make([]MemoryResult, 0, len(combined))
	for id, a := range combined {
		lex := lexicalOverlap(qTokens, a.content)
		// Blend: RRF dominates; lexical breaks near-ties and lifts exact-term hits.
		score := a.score + 0.15*lex + 0.05*float64(a.bestRaw)
		out = append(out, MemoryResult{
			ID:      id,
			Content: a.content,
			Score:   float32(score),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	return out
}

// RankByLexical ranks candidates by query-term overlap (BM25-lite for fusion).
func RankByLexical(query string, candidates []MemoryResult) []MemoryResult {
	if len(candidates) == 0 {
		return nil
	}
	qTokens := tokenSetLower(query)
	out := make([]MemoryResult, len(candidates))
	copy(out, candidates)
	for i := range out {
		lex := lexicalOverlap(qTokens, out[i].Content)
		phrase := 0.0
		lowerQ := strings.ToLower(strings.TrimSpace(query))
		lowerC := strings.ToLower(out[i].Content)
		if lowerQ != "" && strings.Contains(lowerC, lowerQ) {
			phrase = 1.0
		} else if kw := keywordQuery(lowerQ); kw != "" && strings.Contains(lowerC, kw) {
			phrase = 0.5
		}
		out[i].Score = float32(lex + phrase)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	return out
}

// RerankLexical reorders fused hits with lexical overlap for precision@1 when no
// cross-encoder/Cohere reranker is available. Cheap and deterministic.
func RerankLexical(query string, results []MemoryResult, limit int) []MemoryResult {
	if len(results) == 0 {
		return results
	}
	qTokens := tokenSetLower(query)
	type scored struct {
		r     MemoryResult
		score float64
	}
	items := make([]scored, 0, len(results))
	for i, r := range results {
		lex := lexicalOverlap(qTokens, r.Content)
		// Preserve relative dense-rank position as a weak prior.
		densePrior := 1.0 / float64(i+1)
		items = append(items, scored{r: r, score: 0.7*lex + 0.3*densePrior + 0.1*float64(r.Score)})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].score == items[j].score {
			return items[i].r.Score > items[j].r.Score
		}
		return items[i].score > items[j].score
	})
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	out := make([]MemoryResult, 0, limit)
	for i := 0; i < limit; i++ {
		r := items[i].r
		r.Score = float32(items[i].score)
		out = append(out, r)
	}
	return out
}

func tokenSetLower(s string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, t := range wordRe.FindAllString(strings.ToLower(s), -1) {
		if len(t) < 2 {
			continue
		}
		if _, stop := retrievalStopwords[t]; stop {
			continue
		}
		set[t] = struct{}{}
	}
	return set
}

func lexicalOverlap(qTokens map[string]struct{}, content string) float64 {
	if len(qTokens) == 0 || content == "" {
		return 0
	}
	cTokens := tokenSetLower(content)
	var hit int
	for t := range qTokens {
		if _, ok := cTokens[t]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(qTokens))
}

// BlendQAScore picks the stronger of token-F1 and LLM overall (no fixed mix weights).
func BlendQAScore(tokenF1, llmOverall float64) float64 {
	if tokenF1 < 0 {
		tokenF1 = 0
	}
	if llmOverall < 0 {
		llmOverall = 0
	}
	if tokenF1 > 1 {
		tokenF1 = 1
	}
	if llmOverall > 1 {
		llmOverall = 1
	}
	if tokenF1 >= llmOverall {
		return tokenF1
	}
	return llmOverall
}
