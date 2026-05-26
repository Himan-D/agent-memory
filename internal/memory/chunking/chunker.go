package chunking

import (
	"context"
	"strings"
	"unicode"
)

type Chunker interface {
	Chunk(text string) []string
}

type ChunkResult struct {
	Text       string
	ChunkIndex int
	StartPos   int
	EndPos     int
}

type RecursiveChunker struct {
	chunkSize    int
	chunkOverlap int
	separators   []string
}

func NewRecursiveChunker(chunkSize, overlap int, separator string) *RecursiveChunker {
	if separator == "" {
		separator = "\n\n"
	}
	return &RecursiveChunker{
		chunkSize:    chunkSize,
		chunkOverlap: overlap,
		separators:   strings.Split(separator, "|"),
	}
}

func (c *RecursiveChunker) Chunk(text string) []string {
	if text == "" {
		return nil
	}

	var chunks []string
	paragraphs := c.splitBySeparators(text)

	var currentChunk strings.Builder
	currentLen := 0

	for i, para := range paragraphs {
		paraLen := len(para)

		if currentLen+paraLen > c.chunkSize && currentLen > 0 {
			chunk := strings.TrimSpace(currentChunk.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}

			if c.chunkOverlap > 0 {
				overlapText := c.getOverlap(paragraphs, i, c.chunkOverlap)
				currentChunk.Reset()
				currentChunk.WriteString(overlapText)
				currentLen = len(overlapText)
			} else {
				currentChunk.Reset()
				currentLen = 0
			}
		}

		if paraLen > c.chunkSize {
			if currentLen > 0 {
				chunk := strings.TrimSpace(currentChunk.String())
				if chunk != "" {
					chunks = append(chunks, chunk)
				}
				currentChunk.Reset()
				currentLen = 0
			}

			subChunks := c.chunkLargeText(para)
			for _, sub := range subChunks {
				chunks = append(chunks, sub)
			}
		} else {
			if currentLen > 0 {
				currentChunk.WriteString("\n\n")
				currentLen += 2
			}
			currentChunk.WriteString(para)
			currentLen += paraLen
		}
	}

	if remaining := strings.TrimSpace(currentChunk.String()); remaining != "" {
		chunks = append(chunks, remaining)
	}

	return chunks
}

func (c *RecursiveChunker) splitBySeparators(text string) []string {
	var result []string
	var current strings.Builder

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if current.Len() > 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			}
			continue
		}

		for _, sep := range c.separators {
			if strings.Contains(trimmed, sep) {
				parts := strings.Split(trimmed, sep)
				for j, part := range parts {
					part = strings.TrimSpace(part)
					if part != "" {
						if current.Len() > 0 {
							current.WriteString("\n\n")
						}
						current.WriteString(part)
					}
					if j < len(parts)-1 && current.Len() > 0 {
						result = append(result, strings.TrimSpace(current.String()))
						current.Reset()
					}
				}
				break
			}
		}

		if !strings.Contains(trimmed, c.separators[0]) && trimmed != "" {
			if current.Len() > 0 {
				current.WriteString("\n")
			}
			current.WriteString(trimmed)
		}
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	if len(result) == 0 && text != "" {
		return []string{text}
	}

	return result
}

func (c *RecursiveChunker) chunkLargeText(text string) []string {
	var chunks []string
	runes := []rune(text)
	start := 0

	for start < len(runes) {
		end := start + c.chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunk := string(runes[start:end])
		if start+c.chunkSize < len(runes) {
			boundary := c.findWordBoundary(chunk)
			if boundary > 0 {
				chunk = chunk[:boundary]
			}
		}

		chunk = strings.TrimSpace(chunk)
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		nextStart := end - c.chunkOverlap
		if nextStart <= start {
			nextStart = end
		}
		start = nextStart
		if start >= len(runes) {
			break
		}
	}

	return chunks
}

func (c *RecursiveChunker) findWordBoundary(text string) int {
	if len(text) == 0 {
		return 0
	}

	for i := len(text) - 1; i > 0; i-- {
		if unicode.IsSpace(rune(text[i])) {
			return i
		}
	}

	return 0
}

func (c *RecursiveChunker) getOverlap(paragraphs []string, currentIdx, overlap int) string {
	if overlap >= len(paragraphs) {
		return strings.Join(paragraphs, "\n\n")
	}

	start := currentIdx - overlap
	if start < 0 {
		start = 0
	}

	return strings.Join(paragraphs[start:currentIdx], "\n\n")
}

type FixedSizeChunker struct {
	chunkSize    int
	chunkOverlap int
}

func NewFixedSizeChunker(chunkSize, overlap int) *FixedSizeChunker {
	return &FixedSizeChunker{
		chunkSize:    chunkSize,
		chunkOverlap: overlap,
	}
}

func (c *FixedSizeChunker) Chunk(text string) []string {
	if text == "" {
		return nil
	}

	runes := []rune(text)
	var chunks []string

	for i := 0; i < len(runes); i += c.chunkSize - c.chunkOverlap {
		end := i + c.chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunk := string(runes[i:end])
		chunk = strings.TrimSpace(chunk)

		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		if end == len(runes) {
			break
		}
	}

	return chunks
}

type SemanticChunker struct {
	llm                 LLMProvider
	similarityThreshold float64 // Jaccard similarity below this → semantic boundary
}

type LLMProvider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

func NewSemanticChunker(llm LLMProvider) *SemanticChunker {
	return &SemanticChunker{
		llm:                 llm,
		similarityThreshold: 0.1, // empirically: word overlap < 10% signals topic shift
	}
}

// Chunk splits text at semantic boundaries using word-overlap similarity.
// Adjacent sentences with Jaccard similarity below the threshold are treated as
// topic shifts — the real semantic chunking algorithm (no LLM call needed).
func (c *SemanticChunker) Chunk(text string) []string {
	if text == "" {
		return nil
	}

	sentences := splitIntoSentences(text)
	if len(sentences) <= 2 {
		// Too short to chunk semantically — fall back to recursive
		return NewRecursiveChunker(1000, 100, "\n\n").Chunk(text)
	}

	// Find semantic boundaries using word-set Jaccard similarity
	var boundaries []int
	for i := 1; i < len(sentences); i++ {
		sim := jaccardSimilarity(sentences[i-1], sentences[i])
		if sim < c.similarityThreshold {
			boundaries = append(boundaries, i)
		}
	}

	// No boundaries found → single chunk (semantically cohesive)
	if len(boundaries) == 0 {
		return []string{text}
	}

	// Group sentences by boundaries
	var chunks []string
	start := 0
	for _, boundary := range boundaries {
		chunk := strings.Join(sentences[start:boundary], " ")
		if strings.TrimSpace(chunk) != "" {
			chunks = append(chunks, strings.TrimSpace(chunk))
		}
		start = boundary
	}
	// Final chunk
	if start < len(sentences) {
		chunk := strings.Join(sentences[start:], " ")
		if strings.TrimSpace(chunk) != "" {
			chunks = append(chunks, strings.TrimSpace(chunk))
		}
	}

	if len(chunks) == 0 {
		return []string{text}
	}
	return chunks
}

// splitIntoSentences splits text on sentence-ending punctuation.
func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			// Check for abbreviations: if next char is a space or end of string, it's a sentence end
			next := i + 1
			if next >= len(runes) || runes[next] == ' ' || runes[next] == '\n' {
				sent := strings.TrimSpace(current.String())
				if sent != "" {
					sentences = append(sentences, sent)
				}
				current.Reset()
			}
		}
	}
	if rem := strings.TrimSpace(current.String()); rem != "" {
		sentences = append(sentences, rem)
	}
	return sentences
}

// jaccardSimilarity computes the Jaccard similarity of word sets between two sentences.
func jaccardSimilarity(a, b string) float64 {
	wordsA := wordSet(a)
	wordsB := wordSet(b)

	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0
	}

	intersection := 0
	for w := range wordsA {
		if wordsB[w] {
			intersection++
		}
	}

	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// wordSet returns a set of lowercase non-stop-words from a sentence.
func wordSet(s string) map[string]bool {
	set := make(map[string]bool)
	for _, word := range strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		w := strings.ToLower(word)
		// Skip very short words (likely stop words)
		if len(w) > 2 {
			set[w] = true
		}
	}
	return set
}
