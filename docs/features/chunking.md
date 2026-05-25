# Feature: Memory Chunking

**Priority**: P1 — Large memories (>512 tokens) retrieve poorly; vector embeddings average out, losing specific facts.
**Status**: `internal/memory/chunking/` directory exists and is empty.
**Estimated effort**: 1-2 days

---

## What to Build

Two files in `internal/memory/chunking/`:

| File | Responsibility |
|------|---------------|
| `splitter.go` | Splits large memory content into overlapping chunks |
| `merger.go` | Post-processes search results — groups chunks by parent, deduplicates |

---

## Component 1: `splitter.go`

```go
package chunking

import (
    "strings"
    "unicode"
)

type ChunkConfig struct {
    MaxTokens      int    // Default: 512
    OverlapTokens  int    // Default: 50 — overlap preserves context at boundaries
    Strategy       string // "sentence" | "paragraph" | "fixed"
    MinChunkTokens int    // Default: 100 — ignore tiny trailing chunks
}

type Chunk struct {
    Index          int
    Content        string
    TokenCount     int
    StartOffset    int    // byte offset in original content
    EndOffset      int
    ParentMemoryID string
}

type Splitter struct {
    config ChunkConfig
}

func NewSplitter(config ChunkConfig) *Splitter {
    if config.MaxTokens == 0 {
        config.MaxTokens = 512
    }
    if config.OverlapTokens == 0 {
        config.OverlapTokens = 50
    }
    if config.Strategy == "" {
        config.Strategy = "sentence"
    }
    if config.MinChunkTokens == 0 {
        config.MinChunkTokens = 100
    }
    return &Splitter{config: config}
}

func (s *Splitter) Split(content string, parentMemoryID string) []Chunk {
    if TokenCount(content) <= s.config.MaxTokens {
        return []Chunk{{
            Index:          0,
            Content:        content,
            TokenCount:     TokenCount(content),
            StartOffset:    0,
            EndOffset:      len(content),
            ParentMemoryID: parentMemoryID,
        }}
    }

    switch s.config.Strategy {
    case "paragraph":
        return s.splitByParagraph(content, parentMemoryID)
    case "fixed":
        return s.splitFixed(content, parentMemoryID)
    default:
        return s.splitBySentence(content, parentMemoryID)
    }
}

// splitBySentence splits on sentence boundaries (., ?, !)
// Accumulates sentences until token count > MaxTokens, then starts new chunk
// with OverlapTokens tokens from the end of the previous chunk
func (s *Splitter) splitBySentence(content string, parentMemoryID string) []Chunk {
    sentences := tokenizeSentences(content)
    var chunks []Chunk
    var current []string
    currentTokens := 0
    chunkIndex := 0
    offset := 0

    for _, sentence := range sentences {
        sentTokens := TokenCount(sentence)

        if currentTokens+sentTokens > s.config.MaxTokens && len(current) > 0 {
            // Flush current chunk
            chunkContent := strings.Join(current, " ")
            if TokenCount(chunkContent) >= s.config.MinChunkTokens {
                chunks = append(chunks, Chunk{
                    Index:          chunkIndex,
                    Content:        chunkContent,
                    TokenCount:     TokenCount(chunkContent),
                    StartOffset:    offset,
                    EndOffset:      offset + len(chunkContent),
                    ParentMemoryID: parentMemoryID,
                })
                chunkIndex++
            }

            // Carry over overlap: last N tokens worth of sentences
            current, currentTokens = trimToOverlap(current, s.config.OverlapTokens)
            offset += len(chunkContent) - len(strings.Join(current, " "))
        }

        current = append(current, sentence)
        currentTokens += sentTokens
    }

    // Flush remainder
    if len(current) > 0 {
        chunkContent := strings.Join(current, " ")
        if TokenCount(chunkContent) >= s.config.MinChunkTokens {
            chunks = append(chunks, Chunk{
                Index:          chunkIndex,
                Content:        chunkContent,
                TokenCount:     TokenCount(chunkContent),
                StartOffset:    offset,
                EndOffset:      offset + len(chunkContent),
                ParentMemoryID: parentMemoryID,
            })
        }
    }

    return chunks
}

// splitByParagraph splits on double newlines
func (s *Splitter) splitByParagraph(content string, parentMemoryID string) []Chunk {
    paragraphs := strings.Split(content, "\n\n")
    var chunks []Chunk
    var current []string
    currentTokens := 0
    chunkIndex := 0
    offset := 0

    for _, para := range paragraphs {
        para = strings.TrimSpace(para)
        if para == "" {
            continue
        }
        paraTokens := TokenCount(para)

        if currentTokens+paraTokens > s.config.MaxTokens && len(current) > 0 {
            chunkContent := strings.Join(current, "\n\n")
            chunks = append(chunks, Chunk{
                Index:          chunkIndex,
                Content:        chunkContent,
                TokenCount:     currentTokens,
                StartOffset:    offset,
                EndOffset:      offset + len(chunkContent),
                ParentMemoryID: parentMemoryID,
            })
            chunkIndex++
            offset += len(chunkContent) + 2
            current = current[:0]
            currentTokens = 0
        }

        current = append(current, para)
        currentTokens += paraTokens
    }

    if len(current) > 0 {
        chunkContent := strings.Join(current, "\n\n")
        chunks = append(chunks, Chunk{
            Index:          chunkIndex,
            Content:        chunkContent,
            TokenCount:     currentTokens,
            StartOffset:    offset,
            EndOffset:      offset + len(chunkContent),
            ParentMemoryID: parentMemoryID,
        })
    }

    return chunks
}

// splitFixed splits every MaxTokens characters with OverlapTokens overlap
func (s *Splitter) splitFixed(content string, parentMemoryID string) []Chunk {
    words := strings.Fields(content)
    charsPerToken := 4
    maxWords := s.config.MaxTokens * charsPerToken / 5 // approximate
    overlapWords := s.config.OverlapTokens * charsPerToken / 5

    var chunks []Chunk
    i := 0
    chunkIndex := 0

    for i < len(words) {
        end := i + maxWords
        if end > len(words) {
            end = len(words)
        }

        chunkContent := strings.Join(words[i:end], " ")
        chunks = append(chunks, Chunk{
            Index:          chunkIndex,
            Content:        chunkContent,
            TokenCount:     TokenCount(chunkContent),
            ParentMemoryID: parentMemoryID,
        })
        chunkIndex++

        i += maxWords - overlapWords
        if i >= len(words) {
            break
        }
    }

    return chunks
}

// TokenCount estimates tokens using len(content)/4 approximation.
// For production accuracy, use tiktoken-go.
func TokenCount(content string) int {
    return len(content) / 4
}

// tokenizeSentences splits text on sentence-ending punctuation
func tokenizeSentences(text string) []string {
    var sentences []string
    var current strings.Builder

    runes := []rune(text)
    for i, r := range runes {
        current.WriteRune(r)
        if r == '.' || r == '?' || r == '!' {
            // Avoid splitting on abbreviations: check if next char is space or end
            if i+1 >= len(runes) || unicode.IsSpace(runes[i+1]) {
                s := strings.TrimSpace(current.String())
                if s != "" {
                    sentences = append(sentences, s)
                }
                current.Reset()
            }
        }
    }

    if current.Len() > 0 {
        s := strings.TrimSpace(current.String())
        if s != "" {
            sentences = append(sentences, s)
        }
    }

    return sentences
}

// trimToOverlap keeps only enough trailing sentences to fill overlapTokens
func trimToOverlap(sentences []string, overlapTokens int) ([]string, int) {
    if overlapTokens == 0 || len(sentences) == 0 {
        return nil, 0
    }

    var result []string
    total := 0
    for i := len(sentences) - 1; i >= 0; i-- {
        t := TokenCount(sentences[i])
        if total+t > overlapTokens {
            break
        }
        result = append([]string{sentences[i]}, result...)
        total += t
    }
    return result, total
}
```

---

## Component 2: `merger.go`

```go
package chunking

import "sort"

type ScoredChunk struct {
    Chunk Chunk
    Score float64
}

type ChunkResult struct {
    ParentMemoryID string
    Chunks         []ScoredChunk
    BestScore      float64
}

// SearchResult is the minimal interface merger needs from vector search results.
// Match this to the actual types.SearchResult / vector.SearchResult in the codebase.
type SearchResult struct {
    ID             string
    Content        string
    Score          float64
    ParentMemoryID string  // empty if not a chunk
    ChunkIndex     int
    Metadata       map[string]interface{}
}

type Merger struct {
    TopChunksPerMemory int // Default: 3
}

func NewMerger(topChunksPerMemory int) *Merger {
    if topChunksPerMemory == 0 {
        topChunksPerMemory = 3
    }
    return &Merger{TopChunksPerMemory: topChunksPerMemory}
}

// MergeChunkResults groups search results by parent memory ID.
// For each parent: keeps top N chunks by score.
// Returns one result per parent with score = max(chunk scores).
// Non-chunk results (ParentMemoryID == "") pass through unchanged.
func (m *Merger) MergeChunkResults(results []SearchResult) []SearchResult {
    // Separate chunk results from regular results
    regular := make([]SearchResult, 0)
    byParent := make(map[string][]SearchResult)

    for _, r := range results {
        if r.ParentMemoryID == "" {
            regular = append(regular, r)
        } else {
            byParent[r.ParentMemoryID] = append(byParent[r.ParentMemoryID], r)
        }
    }

    // Merge chunks per parent
    merged := make([]SearchResult, 0, len(regular)+len(byParent))
    merged = append(merged, regular...)

    for parentID, chunks := range byParent {
        // Sort by score descending
        sort.Slice(chunks, func(i, j int) bool {
            return chunks[i].Score > chunks[j].Score
        })

        // Keep top N
        top := chunks
        if len(top) > m.TopChunksPerMemory {
            top = top[:m.TopChunksPerMemory]
        }

        // Build merged content (sort by chunk index to preserve reading order)
        sort.Slice(top, func(i, j int) bool {
            return top[i].ChunkIndex < top[j].ChunkIndex
        })

        combinedContent := m.combineContent(top)
        bestScore := chunks[0].Score // Already sorted by score desc

        merged = append(merged, SearchResult{
            ID:             parentID,
            Content:        combinedContent,
            Score:          bestScore,
            ParentMemoryID: "",
            Metadata:       top[0].Metadata, // Use first chunk's metadata
        })
    }

    // Re-sort everything by score
    sort.Slice(merged, func(i, j int) bool {
        return merged[i].Score > merged[j].Score
    })

    return merged
}

// combineContent concatenates chunk contents, deduplicating overlapping text
func (m *Merger) combineContent(chunks []SearchResult) string {
    if len(chunks) == 0 {
        return ""
    }
    if len(chunks) == 1 {
        return chunks[0].Content
    }

    var sb strings.Builder
    sb.WriteString(chunks[0].Content)

    for i := 1; i < len(chunks); i++ {
        // Simple overlap dedup: if end of previous overlaps with start of next, trim
        prev := chunks[i-1].Content
        curr := chunks[i].Content
        overlap := findOverlap(prev, curr)
        if overlap > 0 {
            sb.WriteString(" ")
            sb.WriteString(curr[overlap:])
        } else {
            sb.WriteString(" ")
            sb.WriteString(curr)
        }
    }

    return sb.String()
}

// findOverlap finds the longest suffix of a that is a prefix of b
// Returns the length of the overlap in bytes
func findOverlap(a, b string) int {
    maxOverlap := len(a)
    if len(b) < maxOverlap {
        maxOverlap = len(b)
    }

    for length := maxOverlap; length > 0; length-- {
        if strings.HasSuffix(a, b[:length]) {
            return length
        }
    }
    return 0
}
```

---

## Integration: Wire into `internal/memory/service.go`

**In `CreateMemory()`**:
```go
func (s *Service) CreateMemory(ctx context.Context, req CreateMemoryRequest) (*types.Memory, error) {
    if s.config.Chunking.Enabled {
        tokenCount := chunking.TokenCount(req.Content)
        if tokenCount > s.config.Chunking.MaxTokens {
            return s.createChunkedMemory(ctx, req)
        }
    }
    return s.createMemoryRecord(ctx, req)
}

func (s *Service) createChunkedMemory(ctx context.Context, req CreateMemoryRequest) (*types.Memory, error) {
    splitter := chunking.NewSplitter(chunking.ChunkConfig{
        MaxTokens:     s.config.Chunking.MaxTokens,
        OverlapTokens: s.config.Chunking.OverlapTokens,
        Strategy:      s.config.Chunking.Strategy,
    })
    
    // Create parent memory (summary = first 200 chars)
    parentReq := req
    parentReq.Content = truncate(req.Content, 200) + "..." // summary placeholder
    parent, err := s.createMemoryRecord(ctx, parentReq)
    if err != nil {
        return nil, fmt.Errorf("chunking: create parent: %w", err)
    }
    
    // Create chunk memories
    chunks := splitter.Split(req.Content, parent.ID)
    for i, chunk := range chunks {
        chunkReq := req
        chunkReq.Content = chunk.Content
        chunkReq.ParentMemoryID = parent.ID
        if chunkReq.Metadata == nil {
            chunkReq.Metadata = make(map[string]interface{})
        }
        chunkReq.Metadata["chunk_index"] = i
        chunkReq.Metadata["chunk_total"] = len(chunks)
        
        if _, err := s.createMemoryRecord(ctx, chunkReq); err != nil {
            s.logger.Warn("chunking: failed to create chunk", "index", i, "error", err)
        }
    }
    
    return parent, nil
}
```

**In `Search()`**, after getting vector results:
```go
if s.config.Chunking.Enabled {
    merger := chunking.NewMerger(3)
    results = merger.MergeChunkResults(convertToChunkResults(results))
}
```

---

## Config Additions

Add to `internal/config/config.go`:

```go
type ChunkingConfig struct {
    Enabled        bool   `env:"CHUNKING_ENABLED" default:"true"`
    MaxTokens      int    `env:"CHUNKING_MAX_TOKENS" default:"512"`
    OverlapTokens  int    `env:"CHUNKING_OVERLAP_TOKENS" default:"50"`
    Strategy       string `env:"CHUNKING_STRATEGY" default:"sentence"`
    MinChunkTokens int    `env:"CHUNKING_MIN_TOKENS" default:"100"`
    MergeTopN      int    `env:"CHUNKING_MERGE_TOP_N" default:"3"`
}
```

Add `Chunking ChunkingConfig` to the main `Config` struct.

---

## Environment Variables

```bash
CHUNKING_ENABLED=true
CHUNKING_MAX_TOKENS=512         # Tokens before splitting
CHUNKING_OVERLAP_TOKENS=50      # Tokens to carry over between chunks
CHUNKING_STRATEGY=sentence      # sentence | paragraph | fixed
CHUNKING_MIN_TOKENS=100         # Minimum chunk size (discard smaller trailing chunks)
CHUNKING_MERGE_TOP_N=3          # Max chunks per parent to include in merged result
```

---

## Tests

Create `internal/memory/chunking/splitter_test.go`:

```go
func TestSplit_ShortContent_NoSplit(t *testing.T) {
    // Content with 100 tokens → returns single chunk, no splitting
    s := NewSplitter(ChunkConfig{MaxTokens: 512})
    chunks := s.Split("short content", "parent-1")
    assert.Len(t, chunks, 1)
    assert.Equal(t, "short content", chunks[0].Content)
}

func TestSplit_LongContent_SplitsCorrectly(t *testing.T) {
    // 2000-token content → splits into multiple chunks ≤512 tokens each
    content := strings.Repeat("This is a sentence. ", 200) // ~1600 chars / 4 = 400 tokens? repeat more
    s := NewSplitter(ChunkConfig{MaxTokens: 100, Strategy: "sentence"})
    chunks := s.Split(content, "parent-1")
    assert.Greater(t, len(chunks), 1)
    for _, c := range chunks {
        assert.LessOrEqual(t, c.TokenCount, 150) // some slack for sentence boundaries
    }
}

func TestSplit_OverlapPreserved(t *testing.T) {
    // Last 50 tokens of chunk N appear at start of chunk N+1
    content := buildLongContent(600) // build 600-token content
    s := NewSplitter(ChunkConfig{MaxTokens: 200, OverlapTokens: 50, Strategy: "fixed"})
    chunks := s.Split(content, "p1")
    assert.Greater(t, len(chunks), 1)
    // Verify overlap: end of chunks[0] matches start of chunks[1]
    // (check at word level to avoid whitespace issues)
    end0words := lastNWords(chunks[0].Content, 10)
    start1words := firstNWords(chunks[1].Content, 10)
    assert.Equal(t, end0words, start1words)
}

func TestSplit_ParagraphStrategy(t *testing.T) {
    content := "Para one content.\n\nPara two content.\n\nPara three content."
    s := NewSplitter(ChunkConfig{MaxTokens: 10, Strategy: "paragraph"})
    chunks := s.Split(content, "p1")
    assert.GreaterOrEqual(t, len(chunks), 2)
}

func TestMergeChunkResults_GroupsByParent(t *testing.T) {
    // Two chunks from same parent → merged into one result
    results := []SearchResult{
        {ID: "c1", Score: 0.9, ParentMemoryID: "parent-1", ChunkIndex: 0, Content: "first chunk"},
        {ID: "c2", Score: 0.7, ParentMemoryID: "parent-1", ChunkIndex: 1, Content: "second chunk"},
        {ID: "r1", Score: 0.8, Content: "regular memory"},
    }
    m := NewMerger(3)
    merged := m.MergeChunkResults(results)
    
    assert.Len(t, merged, 2) // one merged + one regular
    // Find parent result
    var parentResult SearchResult
    for _, r := range merged {
        if r.ID == "parent-1" {
            parentResult = r
        }
    }
    assert.Equal(t, 0.9, parentResult.Score) // best chunk score
    assert.Contains(t, parentResult.Content, "first chunk")
    assert.Contains(t, parentResult.Content, "second chunk")
}

func TestMergeChunkResults_TopNEnforced(t *testing.T) {
    // 5 chunks from same parent with MergeTopN=3 → only top 3 by score included
    results := make([]SearchResult, 5)
    for i := 0; i < 5; i++ {
        results[i] = SearchResult{
            ID:             fmt.Sprintf("c%d", i),
            Score:          float64(i) * 0.1,
            ParentMemoryID: "parent-1",
            ChunkIndex:     i,
            Content:        fmt.Sprintf("chunk %d", i),
        }
    }
    m := NewMerger(3)
    merged := m.MergeChunkResults(results)
    assert.Len(t, merged, 1)
    // Score should be max (0.4)
    assert.Equal(t, 0.4, merged[0].Score)
}

func TestTokenCount(t *testing.T) {
    assert.Equal(t, 25, TokenCount(strings.Repeat("x", 100)))
    assert.Equal(t, 0, TokenCount(""))
}
```
