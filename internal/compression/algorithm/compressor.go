package algorithm

import (
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type CompressionAlgorithm interface {
	Compress(text string) (string, float64, error)
	Decompress(compressed string) (string, error)
	GetStats() map[string]interface{}
}

type RealCompressor struct {
	dictionary  *PatternDictionary
	lz77        *LZ77
	huffman     *HuffmanTree
	algorithms []CompressionAlgorithm

	mu         sync.RWMutex
	stats      CompressionStats
	frequencies map[string]int
}

type CompressionStats struct {
	TotalCompressed    int64
	TotalOriginalSize  int64
	TotalCompressedSize int64
	AvgRatio          float64
	PatternsLearned   int
}

func NewRealCompressor() *RealCompressor {
	return &RealCompressor{
		dictionary:  NewPatternDictionary(),
		lz77:        NewLZ77(4096, 18),
		frequencies: make(map[string]int),
	}
}

func (c *RealCompressor) LearnFromMemories(memories []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var allText strings.Builder
	for _, m := range memories {
		allText.WriteString(m)
		allText.WriteString(" ")
		updateFrequencies(m, c.frequencies)
	}

	text := allText.String()

	c.dictionary.LearnFromTexts([]string{text}, 1)

	huff, err := NewHuffmanTree(text)
	if err == nil {
		c.huffman = huff
	}

	c.stats.PatternsLearned = c.dictionary.Size()
	log.Printf("RealCompressor learned %d patterns from %d memories", c.stats.PatternsLearned, len(memories))
}

func updateFrequencies(text string, freqs map[string]int) {
	for _, ch := range text {
		freqs[string(ch)]++
	}

	words := strings.Fields(text)
	for _, w := range words {
		if len(w) > 3 {
			freqs[strings.ToLower(w)]++
		}
	}
}

func (c *RealCompressor) Compress(text string) (*CompressionResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := &CompressionResult{
		Original: text,
	}

	if len(text) == 0 {
		return result, nil
	}

	result.OriginalSize = len(text)

	var bestCompressed string
	bestRatio := -1.0

	if c.dictionary.Size() > 0 {
		dictCompressed, bytesSaved := c.dictionary.Compress(text)
		dictRatio := 0.0
		if len(text) > 0 {
			dictRatio = float64(bytesSaved) / float64(len(text))
		}
		if dictRatio > bestRatio {
			bestRatio = dictRatio
			bestCompressed = dictCompressed
			result.Method = "dictionary"
			result.Ratio = dictRatio
		}
	}

	lz77Compressed, lz77Ratio, _ := c.lz77.Compress(text)
	if lz77Ratio > bestRatio {
		bestRatio = lz77Ratio
		bestCompressed = lz77Compressed
		result.Method = "lz77"
		result.Ratio = lz77Ratio
	}

	if c.huffman != nil {
		encoded, err := c.huffman.Encode(text)
		if err == nil {
			huffRatio := c.huffman.CompressionRatio(text, encoded)
			if huffRatio > bestRatio {
				bestRatio = huffRatio
				bestCompressed = encoded
				result.Method = "huffman"
				result.Ratio = huffRatio
			}
		}
	}

	hybridCompressed, hybridRatio := c.compressHybrid(text)
	if hybridRatio > bestRatio {
		bestCompressed = hybridCompressed
		result.Method = "hybrid"
		result.Ratio = hybridRatio
	}

	if bestRatio <= 0 {
		result.Compressed = text
		result.Ratio = 0
	} else {
		result.Compressed = bestCompressed
	}

	result.CompressedSize = len(result.Compressed)

	c.stats.TotalCompressed++
	c.stats.TotalOriginalSize += int64(result.OriginalSize)
	c.stats.TotalCompressedSize += int64(result.CompressedSize)

	if c.stats.TotalOriginalSize > 0 {
		c.stats.AvgRatio = 1.0 - float64(c.stats.TotalCompressedSize)/float64(c.stats.TotalOriginalSize)
	}

	return result, nil
}

func (c *RealCompressor) compressHybrid(text string) (string, float64) {
	var result strings.Builder
	originalLen := len(text)
	bytesSaved := 0

	segments := splitIntoSegments(text, 100)

	for _, seg := range segments {
		compressed, saved := c.dictionary.Compress(seg)
		result.WriteString(compressed)
		result.WriteString(" ")
		bytesSaved += saved
	}

	compressed := result.String()
	if compressed == "" {
		return text, 0
	}

	ratio := float64(bytesSaved) / float64(originalLen)
	return compressed, ratio
}

func splitIntoSegments(text string, maxLen int) []string {
	var segments []string

	re := regexp.MustCompile(`[^.!?]+[.!?]`)
	matches := re.FindAllString(text, -1)

	if len(matches) == 0 {
		words := strings.Fields(text)
		var current strings.Builder

		for _, w := range words {
			if current.Len()+len(w) > maxLen {
				if current.Len() > 0 {
					segments = append(segments, current.String())
				}
				current.Reset()
			}
			current.WriteString(w)
			current.WriteString(" ")
		}

		if current.Len() > 0 {
			segments = append(segments, current.String())
		}
	} else {
		segments = matches
	}

	return segments
}

func (c *RealCompressor) Decompress(compressed, method string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch method {
	case "dictionary":
		return c.dictionary.Decompress(compressed)
	case "lz77":
		return c.lz77.Decompress(compressed)
	case "huffman":
		if c.huffman != nil {
			return c.huffman.Decode(compressed)
		}
		return "", nil
	default:
		decompressed, err := c.dictionary.Decompress(compressed)
		if err == nil && decompressed != compressed {
			return decompressed, nil
		}
		return c.lz77.Decompress(compressed)
	}
}

func (c *RealCompressor) GetStats() *CompressionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.PatternsLearned = c.dictionary.Size()
	return &stats
}

func (c *RealCompressor) GetPatternsLearned() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dictionary.Size()
}

func (c *RealCompressor) GetBestMethod(text string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var bestMethod string
	bestRatio := -1.0

	results := make(map[string]float64)

	if c.dictionary.Size() > 0 {
		compressed, bytesSaved := c.dictionary.Compress(text)
		ratio := 0.0
		if len(text) > 0 {
			ratio = float64(bytesSaved) / float64(len(text))
		}
		if ratio > 0 {
			results["dictionary"] = ratio
			if ratio > bestRatio {
				bestRatio = ratio
				bestMethod = "dictionary"
				_ = compressed
			}
		}
	}

	if _, ratio, _ := c.lz77.Compress(text); ratio > 0 {
		results["lz77"] = ratio
		if ratio > bestRatio {
			bestRatio = ratio
			bestMethod = "lz77"
		}
	}

	hybridCompressed, hybridRatio := c.compressHybrid(text)
	if hybridRatio > 0 {
		results["hybrid"] = hybridRatio
		if hybridRatio > bestRatio {
			bestMethod = "hybrid"
		}
	}
	_ = hybridCompressed

	if len(results) == 0 {
		return "none"
	}

	return bestMethod
}

type CompressionResult struct {
	Original      string
	Compressed   string
	OriginalSize  int
	CompressedSize int
	Ratio        float64
	Method       string
}

func (r *CompressionResult) CompressionEfficiency() float64 {
	if r.OriginalSize == 0 {
		return 0
	}
	return 1.0 - float64(r.CompressedSize)/float64(r.OriginalSize)
}

func (r *CompressionResult) BytesSaved() int {
	return r.OriginalSize - r.CompressedSize
}

type HuffmanWordCompressor struct {
	tree     *HuffmanTree
	dictionary map[string]string
	reverse  map[string]string
}

func NewHuffmanWordCompressor() *HuffmanWordCompressor {
	return &HuffmanWordCompressor{
		dictionary: make(map[string]string),
		reverse:   make(map[string]string),
	}
}

func (c *HuffmanWordCompressor) Learn(texts []string) {
	freqs := make(map[string]int)
	for _, text := range texts {
		words := strings.Fields(text)
		for _, w := range words {
			freqs[strings.ToLower(w)]++
		}
	}

	root := BuildWordTree(freqs)
	if root != nil {
		c.tree = &HuffmanTree{root: root}
		c.dictionary, c.reverse = BuildWordCodesFromRoot(root)
	}
}

func BuildWordTree(freqs map[string]int) *HuffmanNode {
	if len(freqs) == 0 {
		return nil
	}

	type wordFreq struct {
		word string
		freq int
		node *HuffmanNode
	}

	var heapData []wordFreq
	for word, freq := range freqs {
		heapData = append(heapData, wordFreq{word, freq, &HuffmanNode{Char: word, Freq: freq}})
	}

	sort.Slice(heapData, func(i, j int) bool {
		return heapData[i].freq < heapData[j].freq
	})

	for len(heapData) > 1 {
		left := heapData[0]
		heapData = heapData[1:]
		right := heapData[0]
		heapData = heapData[1:]

		parent := &HuffmanNode{
			Char: "",
			Freq: left.freq + right.freq,
			Left: left.node,
			Right: right.node,
		}
		heapData = append(heapData, wordFreq{"", parent.Freq, parent})
		sort.Slice(heapData, func(i, j int) bool {
			return heapData[i].freq < heapData[j].freq
		})
	}

	return heapData[0].node
}

func BuildWordCodes(node *HuffmanNode, prefix string, codes map[string]string) (map[string]string, map[string]string) {
	if node == nil {
		return codes, nil
	}

	if node.Left == nil && node.Right == nil {
		if node.Char != "" {
			if prefix == "" {
				prefix = "0"
			}
			codes[node.Char] = prefix
		}
		return codes, nil
	}

	BuildWordCodes(node.Left, prefix+"0", codes)
	BuildWordCodes(node.Right, prefix+"1", codes)

	return codes, nil
}

func BuildWordCodesFromRoot(node *HuffmanNode) (map[string]string, map[string]string) {
	if node == nil {
		return make(map[string]string), make(map[string]string)
	}

	codes := make(map[string]string)
	reverse := make(map[string]string)

	buildCodesRecursive(node, "", codes)

	for k, v := range codes {
		reverse[v] = k
	}

	return codes, reverse
}

func buildCodesRecursive(node *HuffmanNode, prefix string, codes map[string]string) {
	if node == nil {
		return
	}

	if node.Left == nil && node.Right == nil {
		if node.Char != "" {
			if prefix == "" {
				prefix = "0"
			}
			codes[node.Char] = prefix
		}
		return
	}

	buildCodesRecursive(node.Left, prefix+"0", codes)
	buildCodesRecursive(node.Right, prefix+"1", codes)
}

func (c *HuffmanWordCompressor) Compress(text string) (string, float64) {
	if len(c.dictionary) == 0 {
		return text, 0
	}

	words := strings.Fields(text)
	var compressed strings.Builder
	originalLen := 0
	compressedLen := 0

	for _, word := range words {
		originalLen += len(word) + 1
		if code, ok := c.dictionary[strings.ToLower(word)]; ok {
			compressed.WriteString(code)
			compressedLen += len(code)
		} else {
			compressed.WriteString(word)
			compressedLen += len(word)
		}
		compressed.WriteString(" ")
	}

	result := compressed.String()
	ratio := 0.0
	if originalLen > 0 {
		ratio = 1.0 - float64(compressedLen)/float64(originalLen)
	}

	return result, ratio
}

func (c *HuffmanWordCompressor) Decompress(compressed string) (string, error) {
	if len(c.reverse) == 0 {
		return compressed, nil
	}

	words := strings.Fields(compressed)
	var result strings.Builder

	for _, word := range words {
		if code, ok := c.reverse[word]; ok {
			result.WriteString(code)
		} else {
			result.WriteString(word)
		}
		result.WriteString(" ")
	}

	return result.String(), nil
}