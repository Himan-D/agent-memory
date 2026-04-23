package algorithm

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type PatternEntry struct {
	Pattern string
	Code   string
	Freq   int
}

type PatternDictionary struct {
	entries  []PatternEntry
	lookup  map[string]string
	reverse map[string]string
	maxLen  int
}

func NewPatternDictionary() *PatternDictionary {
	return &PatternDictionary{
		entries: make([]PatternEntry, 0),
		lookup:  make(map[string]string),
		reverse: make(map[string]string),
		maxLen:  50,
	}
}

func (d *PatternDictionary) LearnFromTexts(texts []string, minFreq int) {
	patternFreq := make(map[string]int)

	for _, text := range texts {
		patterns := extractPatterns(text)
		for p := range patterns {
			patternFreq[p]++
		}
	}

	var sorted []struct {
		pattern string
		freq   int
	}
	for p, f := range patternFreq {
		if f >= minFreq {
			sorted = append(sorted, struct {
				pattern string
				freq   int
			}{p, f})
		}
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].freq > sorted[j].freq
	})

	d.entries = d.entries[:0]
	d.lookup = make(map[string]string)
	d.reverse = make(map[string]string)

	for i, s := range sorted {
		if i >= 256 {
			break
		}
		code := fmt.Sprintf("!%d!", i+1)
		d.entries = append(d.entries, PatternEntry{
			Pattern: s.pattern,
			Code:    code,
			Freq:    s.freq,
		})
		d.lookup[s.pattern] = code
		d.reverse[code] = s.pattern
	}

	d.maxLen = 0
	for _, e := range d.entries {
		if len(e.Pattern) > d.maxLen {
			d.maxLen = len(e.Pattern)
		}
	}
}

func extractPatterns(text string) map[string]bool {
	patterns := make(map[string]bool)

	words := strings.Fields(text)
	for _, w := range words {
		if len(w) > 4 {
			patterns[strings.ToLower(w)] = true
		}
	}

	re := regexp.MustCompile(`\b[A-Za-z]{5,}\b`)
	matches := re.FindAllString(text, -1)
	for _, m := range matches {
		patterns[strings.ToLower(m)] = true
	}

	for i := 0; i < len(words)-1; i++ {
		bigram := strings.ToLower(words[i]) + " " + strings.ToLower(words[i+1])
		patterns[bigram] = true
	}

	return patterns
}

func (d *PatternDictionary) Compress(text string) (string, int) {
	if len(d.lookup) == 0 {
		return text, 0
	}

	var result strings.Builder
	bytesSaved := 0

	i := 0
	for i < len(text) {
		found := false

		for l := d.maxLen; l > 0 && !found; l-- {
			if i+l > len(text) {
				continue
			}
			sub := text[i : i+l]

			if code, ok := d.lookup[sub]; ok {
				result.WriteString(code)
				result.WriteByte(' ')
				bytesSaved += len(sub) - 1
				i += l
				found = true
				break
			}
		}

		if !found {
			result.WriteByte(text[i])
			i++
		}
	}

	return result.String(), bytesSaved
}

func (d *PatternDictionary) Decompress(compressed string) (string, error) {
	if len(d.reverse) == 0 {
		return compressed, nil
	}

	var result strings.Builder
	tokens := strings.Fields(compressed)

	for _, token := range tokens {
		if pat, ok := d.reverse[token]; ok {
			result.WriteString(pat)
			result.WriteString(" ")
		} else {
			result.WriteString(token)
			result.WriteString(" ")
		}
	}

	decompressed := result.String()
	if len(decompressed) > 0 && decompressed[len(decompressed)-1] == ' ' {
		decompressed = decompressed[:len(decompressed)-1]
	}

	return decompressed, nil
}

func (d *PatternDictionary) Size() int {
	return len(d.entries)
}

func (d *PatternDictionary) GetCompressionRatio(original, compressed string) float64 {
	if len(original) == 0 {
		return 0
	}
	return 1.0 - float64(len(compressed))/float64(len(original))
}

type LearnedCompressor struct {
	dictionary *PatternDictionary
	huffman    *HuffmanTree
}

func NewLearnedCompressor() *LearnedCompressor {
	return &LearnedCompressor{
		dictionary: NewPatternDictionary(),
	}
}

func (c *LearnedCompressor) LearnPatterns(texts []string) {
	c.dictionary.LearnFromTexts(texts, 2)
}

func (c *LearnedCompressor) Compress(text string) (string, float64, error) {
	if c.dictionary.Size() == 0 {
		return text, 0, errors.New("no patterns learned")
	}

	compressed, _ := c.dictionary.Compress(text)
	ratio := c.dictionary.GetCompressionRatio(text, compressed)

	if ratio < 0 {
		return text, 0, nil
	}

	return compressed, ratio, nil
}

func (c *LearnedCompressor) Decompress(compressed string) (string, error) {
	return c.dictionary.Decompress(compressed)
}

func (c *LearnedCompressor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"patterns":    c.dictionary.Size(),
		"compression": "dictionary",
	}
}