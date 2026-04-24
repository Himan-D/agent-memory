package retrieval

import (
	"math"
	"strings"
)

type BM25 struct {
	documents []string
	frequencies []map[string]int
	avgdl float64
	k1 float64
	b float64
}

func NewBM25(documents []string) *BM25 {
	bm := &BM25{
		documents: documents,
		k1: 1.5,
		b: 0.75,
	}
	bm.buildIndex()
	return bm
}

func (bm *BM25) buildIndex() {
	bm.frequencies = make([]map[string]int, len(bm.documents))
	totalLength := 0

	for i, doc := range bm.documents {
		tokens := bm.tokenize(doc)
		bm.frequencies[i] = make(map[string]int)
		for _, token := range tokens {
			bm.frequencies[i][token]++
		}
		totalLength += len(tokens)
	}

	if len(bm.documents) > 0 {
		bm.avgdl = float64(totalLength) / float64(len(bm.documents))
	}
}

func (bm *BM25) tokenize(text string) []string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "'", "")
	text = strings.ReplaceAll(text, "\"", "")
	
	var tokens []string
	var current strings.Builder
	
	for _, r := range text {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == ' ' || r == '-' || r == '_' {
			if r == ' ' || r == '-' || r == '_' {
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(r)
			}
		}
	}
	
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	
	return tokens
}

func (bm *BM25) Score(query string, docIndex int) float64 {
	if docIndex >= len(bm.documents) {
		return 0
	}
	
	tokens := bm.tokenize(query)
	docFreq := bm.frequencies[docIndex]
	docLen := len(bm.tokenize(bm.documents[docIndex]))
	
	var score float64
	for _, token := range tokens {
		tf := float64(docFreq[token])
		if tf == 0 {
			continue
		}
		
		df := bm.documentFrequency(token)
		idf := bm.idf(df)
		
		tfidf := tf * (bm.k1 + 1) / (tf + bm.k1 * (1 - bm.b + bm.b * float64(docLen) / bm.avgdl))
		score += idf * tfidf
	}
	
	return score
}

func (bm *BM25) documentFrequency(token string) int {
	count := 0
	for _, freq := range bm.frequencies {
		if freq[token] > 0 {
			count++
		}
	}
	return count
}

func (bm *BM25) idf(df int) float64 {
	if df == 0 {
		return 0
	}
	return math.Log(float64(len(bm.documents)) / float64(df))
}

func (bm *BM25) Search(query string, topK int) []int {
	var scores []struct {
		index int
		score float64
	}
	
	for i := range bm.documents {
		score := bm.Score(query, i)
		if score > 0 {
			scores = append(scores, struct {
				index int
				score float64
			}{i, score})
		}
	}
	
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	
	var results []int
	for i := 0; i < topK && i < len(scores); i++ {
		results = append(results, scores[i].index)
	}
	
	return results
}

func (bm *BM25) UpdateDocuments(documents []string) {
	bm.documents = documents
	bm.buildIndex()
}