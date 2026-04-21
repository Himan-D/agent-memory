package radix

import (
	"strings"
	"sync"
)

type Node struct {
	children map[rune]*Node
	value    string
	isEnd    bool
}

type RadixTree struct {
	root *Node
	mu   sync.RWMutex
}

func New() *RadixTree {
	return &RadixTree{
		root: &Node{children: make(map[rune]*Node)},
	}
}

func (r *RadixTree) Insert(key, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if key == "" || value == "" {
		return
	}

	current := r.root
	keyLower := strings.ToLower(key)

	for _, char := range keyLower {
		if _, exists := current.children[char]; !exists {
			current.children[char] = &Node{children: make(map[rune]*Node)}
		}
		current = current.children[char]
	}
	current.isEnd = true
	current.value = value
}

func (r *RadixTree) Search(key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	current := r.root
	keyLower := strings.ToLower(key)

	for _, char := range keyLower {
		if _, exists := current.children[char]; !exists {
			return "", false
		}
		current = current.children[char]
	}

	if current.isEnd {
		return current.value, true
	}
	return "", false
}

func (r *RadixTree) Delete(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	current := r.root
	keyLower := strings.ToLower(key)
	var path []*Node

	for _, char := range keyLower {
		if _, exists := current.children[char]; !exists {
			return false
		}
		current = current.children[char]
		path = append(path, current)
	}

	if !current.isEnd {
		return false
	}

	current.isEnd = false
	current.value = ""

	for i := len(path) - 1; i >= 0; i-- {
		node := path[i]
		if len(node.children) > 0 || node.isEnd {
			break
		}
	}

	return true
}

func (r *RadixTree) FindByPrefix(prefix string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []string
	current := r.root
	prefixLower := strings.ToLower(prefix)

	for _, char := range prefixLower {
		if _, exists := current.children[char]; !exists {
			return results
		}
		current = current.children[char]
	}

	r.collectValues(current, prefix, &results)
	return results
}

func (r *RadixTree) collectValues(node *Node, prefix string, results *[]string) {
	if node.isEnd {
		*results = append(*results, node.value)
	}

	for char, child := range node.children {
		r.collectValues(child, prefix+string(char), results)
	}
}

func (r *RadixTree) Compress(memory string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	words := strings.Fields(memory)
	var compressed []string

	for _, word := range words {
		current := r.root
		wordLower := strings.ToLower(word)
		found := false

		for _, char := range wordLower {
			if child, exists := current.children[char]; exists {
				current = child
				if current.isEnd {
					compressed = append(compressed, "["+current.value+"]")
					found = true
					break
				}
			} else {
				break
			}
		}

		if !found {
			compressed = append(compressed, word)
		}
	}

	return strings.Join(compressed, " ")
}

type MemoryCompressor struct {
	tree        *RadixTree
	patterns    map[string]string
	maxPatternLen int
}

func NewMemoryCompressor() *MemoryCompressor {
	return &MemoryCompressor{
		tree:         New(),
		patterns:     make(map[string]string),
		maxPatternLen: 50,
	}
}

func (c *MemoryCompressor) AddPattern(key, value string) {
	if len(key) > c.maxPatternLen {
		key = key[:c.maxPatternLen]
	}
	c.tree.Insert(key, value)
	c.patterns[key] = value
}

func (c *MemoryCompressor) Compress(text string) string {
	// First apply learned patterns
	compressed := text
	for key, value := range c.patterns {
		compressed = strings.ReplaceAll(compressed, key, "["+value+"]")
	}
	
	// Then use radix tree for common words
	compressed = c.tree.Compress(compressed)

	lines := strings.Split(compressed, "\n")
	if len(lines) > 1 {
		var deduped []string
		seen := make(map[string]bool)
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !seen[trimmed] {
				seen[trimmed] = true
				deduped = append(deduped, trimmed)
			}
		}
		compressed = strings.Join(deduped, "\n")
	}

	return compressed
}

func (c *MemoryCompressor) Decompress(text string) string {
	result := text

	for key, value := range c.patterns {
		result = strings.ReplaceAll(result, "["+value+"]", key)
	}

	return result
}

func (c *MemoryCompressor) LearnFromMemories(memories []string) {
	wordCounts := make(map[string]int)

	for _, mem := range memories {
		words := strings.Fields(strings.ToLower(mem))
		for _, word := range words {
			if len(word) > 3 {
				wordCounts[word]++
			}
		}
	}

	for word, count := range wordCounts {
		if count > 2 {
			abbrev := generateAbbreviation(word)
			c.AddPattern(word, abbrev)
		}
	}
}

func generateAbbreviation(word string) string {
	if len(word) <= 4 {
		return word
	}

	vowels := "aeiou"
	var result []rune
	for i, char := range word {
		if i == 0 || i == len(word)-1 || strings.ContainsRune(vowels, char) {
			result = append(result, char)
		}
	}

	if len(result) < 2 {
		return word[:4]
	}
	return string(result)
}

type CompressionStats struct {
	OriginalSize  int
	CompressedSize int
	PatternsUsed  int
	Reduction     float64
}

func (c *MemoryCompressor) GetStats(text string) CompressionStats {
	compressed := c.Compress(text)
	stats := CompressionStats{
		OriginalSize:   len(text),
		CompressedSize: len(compressed),
		PatternsUsed:   len(c.patterns),
	}

	if stats.OriginalSize > 0 {
		stats.Reduction = 1.0 - (float64(stats.CompressedSize) / float64(stats.OriginalSize))
	}

	return stats
}
