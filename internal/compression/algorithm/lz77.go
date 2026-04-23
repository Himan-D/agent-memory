package algorithm

import (
	"fmt"
	"strings"
)

type LZ77 struct {
	windowSize int
	lookAhead  int
}

type LZ77Token struct {
	Offset int
	Length int
	Char   string
}

func NewLZ77(windowSize, lookAhead int) *LZ77 {
	if windowSize <= 0 {
		windowSize = 4096
	}
	if lookAhead <= 0 {
		lookAhead = 18
	}
	return &LZ77{
		windowSize: windowSize,
		lookAhead:  lookAhead,
	}
}

func (l *LZ77) Compress(text string) (string, float64, error) {
	if len(text) == 0 {
		return "", 0, nil
	}

	var result strings.Builder
	originalLen := len(text)
	bytesSaved := 0

	windowStart := 0

	for i := 0; i < len(text); {
		match := l.findLongestMatch(text, windowStart, i)

		if match.Length >= 2 {
			result.WriteString(fmt.Sprintf("[%d,%d,%s]", match.Offset, match.Length, match.Char))
			bytesSaved += match.Length - len(fmt.Sprintf("[%d,%d,%s]", match.Offset, match.Length, match.Char))
			i += match.Length
		} else {
			result.WriteByte(text[i])
			i++
		}

		if i-l.windowSize > 0 {
			windowStart = i - l.windowSize
			if windowStart < 0 {
				windowStart = 0
			}
		}
	}

	compressed := result.String()
	ratio := 1.0 - float64(len(compressed))/float64(originalLen)
	return compressed, ratio, nil
}

func (l *LZ77) findLongestMatch(text string, windowStart, pos int) *LZ77Token {
	if pos >= len(text) {
		return &LZ77Token{}
	}

	lookAheadEnd := pos + l.lookAhead
	if lookAheadEnd > len(text) {
		lookAheadEnd = len(text)
	}

	bestOffset := 0
	bestLength := 0

	for i := windowStart; i < pos; i++ {
		length := 0
		for (i+length < pos) && (pos+length < lookAheadEnd) && (text[i+length] == text[pos+length]) {
			length++
			if length >= l.lookAhead {
				break
			}
		}

		if length > bestLength {
			bestLength = length
			bestOffset = pos - i
		}
	}

	nextChar := ""
	if pos+bestLength < len(text) {
		nextChar = string(text[pos+bestLength])
	}

	return &LZ77Token{
		Offset: bestOffset,
		Length: bestLength,
		Char:   nextChar,
	}
}

func (l *LZ77) Decompress(compressed string) (string, error) {
	if len(compressed) == 0 {
		return "", nil
	}

	var result strings.Builder
	i := 0

	for i < len(compressed) {
		if compressed[i] == '[' {
			end := strings.Index(compressed[i:], "]")
			if end == -1 {
				result.WriteByte(compressed[i])
				i++
				continue
			}

			token := compressed[i+1 : i+end]
			var offset, length int
			n, err := fmt.Sscanf(token, "%d,%d", &offset, &length)
			if err != nil || n < 2 {
				result.WriteByte(compressed[i])
				i++
				continue
			}

			pos := result.Len()
			start := pos - offset

			for j := 0; j < length; j++ {
				if start+j >= 0 && start+j < result.Len() {
					result.WriteByte(result.String()[start+j])
				}
			}

			i += end + 1
		} else {
			result.WriteByte(compressed[i])
			i++
		}
	}

	return result.String(), nil
}

func (l *LZ77) GetStats(original, compressed string) map[string]interface{} {
	ratio := 0.0
	if len(original) > 0 {
		ratio = 1.0 - float64(len(compressed))/float64(len(original))
	}
	return map[string]interface{}{
		"window_size":   l.windowSize,
		"look_ahead":    l.lookAhead,
		"original_size":  len(original),
		"compressed_size": len(compressed),
		"ratio":         ratio,
	}
}

type DeltaEncoder struct{}

func NewDeltaEncoder() *DeltaEncoder {
	return &DeltaEncoder{}
}

func (d *DeltaEncoder) Compress(numbers []int) ([]int, error) {
	if len(numbers) == 0 {
		return []int{}, nil
	}

	compressed := make([]int, len(numbers))
	compressed[0] = numbers[0]

	for i := 1; i < len(numbers); i++ {
		compressed[i] = numbers[i] - numbers[i-1]
	}

	return compressed, nil
}

func (d *DeltaEncoder) Decompress(compressed []int) ([]int, error) {
	if len(compressed) == 0 {
		return []int{}, nil
	}

	numbers := make([]int, len(compressed))
	numbers[0] = compressed[0]

	for i := 1; i < len(compressed); i++ {
		numbers[i] = numbers[i-1] + compressed[i]
	}

	return numbers, nil
}

func (d *DeltaEncoder) GetRatio(original, compressed []int) float64 {
	if len(original) == 0 {
		return 0
	}
	return 1.0 - float64(len(compressed))/float64(len(original))
}
