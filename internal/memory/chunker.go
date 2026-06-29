package memory

import (
	"strings"

	"github.com/google/uuid"
)

// Chunker splits a document into semantically coherent chunks. The default
// implementation is paragraph-based: it accumulates whole paragraphs until the
// running size would exceed MaxChunkSize, then emits a chunk and starts a new
// one. This mirrors Cognee's TextChunker while exposing an interface so other
// strategies (sentence, token, sliding window) can plug in.
type Chunker interface {
	// Chunk returns chunks for the given documentID + text. The documentID
	// is used to derive deterministic chunk IDs.
	Chunk(documentID, text string) []DocumentChunk
}

// ParagraphChunker is the default Chunker implementation. It splits on
// blank-line paragraph boundaries and never breaks a paragraph across chunks.
type ParagraphChunker struct {
	// MaxChunkSize is the inclusive byte budget for a single chunk. Chunks
	// may be slightly smaller than this because the chunker never splits a
	// paragraph. Values <= 0 fall back to 1600 bytes.
	MaxChunkSize int
}

// NewParagraphChunker returns a ParagraphChunker with the given max chunk size.
func NewParagraphChunker(maxChunkSize int) *ParagraphChunker {
	if maxChunkSize <= 0 {
		maxChunkSize = 1600
	}
	return &ParagraphChunker{MaxChunkSize: maxChunkSize}
}

// DocumentChunk represents one chunk of a source document. The ID is
// deterministic from the source document ID and the chunk's index, so
// re-chunking the same text yields stable IDs.
type DocumentChunk struct {
	ID         string
	DocumentID string
	Index      int
	Text       string
	ByteSize   int
	// IndexFields tells the vector store which fields are searchable.
	IndexFields []string
	// Metadata carries optional provenance (tenant, user, pipeline run).
	Metadata map[string]string
}

// Chunk implements Chunker.
func (p *ParagraphChunker) Chunk(documentID, text string) []DocumentChunk {
	max := p.MaxChunkSize
	if max <= 0 {
		max = 1600
	}

	paragraphs := splitParagraphs(text)
	if len(paragraphs) == 0 {
		// Nothing to chunk. Return a single empty chunk with the deterministic
		// id for index 0 so callers always get at least one chunk per document.
		return []DocumentChunk{emptyChunk(documentID, 0, text)}
	}

	var (
		chunks       []DocumentChunk
		bucket       []string
		bucketSize   int
		chunkIndex   int
	)
	flush := func() {
		if len(bucket) == 0 {
			return
		}
		joined := strings.Join(bucket, "\n\n")
		chunks = append(chunks, DocumentChunk{
			ID:          chunkID(documentID, chunkIndex),
			DocumentID:  documentID,
			Index:       chunkIndex,
			Text:        joined,
			ByteSize:    len(joined),
			IndexFields: []string{"text"},
		})
		chunkIndex++
		bucket = bucket[:0]
		bucketSize = 0
	}

	for _, para := range paragraphs {
		pSize := len(para)
		// If a single paragraph exceeds the budget, emit it as its own chunk
		// to avoid dropping content. Subsequent paragraphs start a new chunk.
		if pSize > max {
			flush()
			chunks = append(chunks, DocumentChunk{
				ID:          chunkID(documentID, chunkIndex),
				DocumentID:  documentID,
				Index:       chunkIndex,
				Text:        para,
				ByteSize:    pSize,
				IndexFields: []string{"text"},
			})
			chunkIndex++
			continue
		}
		// Would adding this paragraph overflow? If yes, emit what we have.
		if bucketSize > 0 && bucketSize+pSize+2 > max {
			flush()
		}
		bucket = append(bucket, para)
		bucketSize += pSize + 2 // +2 for the "\n\n" separator we will join with
	}
	flush()

	return chunks
}

// splitParagraphs splits text on blank lines and trims surrounding whitespace.
// Empty paragraphs are dropped.
func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func emptyChunk(documentID string, idx int, text string) DocumentChunk {
	return DocumentChunk{
		ID:          chunkID(documentID, idx),
		DocumentID:  documentID,
		Index:       idx,
		Text:        text,
		ByteSize:    len(text),
		IndexFields: []string{"text"},
	}
}

// chunkID returns a deterministic UUIDv5-based id for a chunk. The same
// (documentID, index) pair always produces the same id, which keeps re-ingestion
// idempotent.
func chunkID(documentID string, index int) string {
	ns := uuid.NewSHA1(uuid.NameSpaceOID, []byte("agent-memory:chunker:v1"))
	key := documentID + ":" + itoa(index)
	return uuid.NewSHA1(ns, []byte(key)).String()
}

// itoa avoids importing strconv into the chunker's hot path.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
