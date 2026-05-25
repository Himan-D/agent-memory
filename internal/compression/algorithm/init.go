package algorithm

import (
	"math"
)

type CompressionMode string

const (
	ModeDictionary CompressionMode = "dictionary"
	ModeLZ77       CompressionMode = "lz77"
	ModeHuffman    CompressionMode = "huffman"
	ModeHybrid    CompressionMode = "hybrid"
	ModeAuto     CompressionMode = "auto"
)

type Stats struct {
	TotalOriginal     int64
	TotalCompressed  int64
	AvgRatio         float64
	PatternsLearned   int
	BestMethod       string
}

func (s *Stats) Efficiency() float64 {
	if s.TotalOriginal == 0 {
		return 0
	}
	return (1.0 - float64(s.TotalCompressed)/float64(s.TotalOriginal)) * 100
}

func (s *Stats) BytesSaved() int64 {
	return s.TotalOriginal - s.TotalCompressed
}

func (s *Stats) SavingsPercent() float64 {
	if s.TotalOriginal == 0 {
		return 0
	}
	return float64(s.TotalOriginal-s.TotalCompressed) / float64(s.TotalOriginal) * 100
}

func Round(x float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(x*ratio) / ratio
}