package safety

import (
	"regexp"
	"strings"
)

// Classifier detects potentially harmful or malicious memory content.
// Based on FSFM paper (arXiv:2604.20300) -- safety-triggered forgetting class.
type Classifier struct {
	patterns     []*compiledPattern
	keywords     []keywordRule
	blockEnabled bool
}

type compiledPattern struct {
	re       *regexp.Regexp
	category string
	reason   string
}

type keywordRule struct {
	phrases  []string
	category string
	reason   string
}

// ClassificationResult holds the result of content classification.
type ClassificationResult struct {
	Safe       bool     `json:"safe"`
	Category   string   `json:"category"` // safe, suspicious, malicious, sensitive
	Reason     string   `json:"reason"`
	Confidence float64  `json:"confidence"`
	Triggers   []string `json:"triggers"`
}

// NewClassifier creates a Classifier with built-in pattern and keyword rules.
func NewClassifier() *Classifier {
	c := &Classifier{
		blockEnabled: true,
	}

	// Prompt injection patterns.
	c.addPattern(`(?i)ignore\s+(all\s+)?previous\s+instructions`, "malicious", "prompt injection: instruction override")
	c.addPattern(`(?i)system\s*prompt\s*:`, "malicious", "prompt injection: system prompt leak attempt")
	c.addPattern(`(?i)you\s+are\s+now\s+`, "suspicious", "prompt injection: role reassignment")
	c.addPattern(`(?i)disregard\s+(all\s+)?prior`, "malicious", "prompt injection: disregard prior")
	c.addPattern(`(?i)forget\s+(everything|all)\s+(you|above)`, "malicious", "prompt injection: memory wipe attempt")
	c.addPattern(`(?i)\bDAN\b.*\bmode\b`, "malicious", "prompt injection: jailbreak (DAN)")

	// Credential / secret patterns.
	c.addPattern(`(?i)(api[_\-]?key|apikey)\s*[:=]\s*\S{10,}`, "sensitive", "credential: API key detected")
	c.addPattern(`(?i)(password|passwd|pwd)\s*[:=]\s*\S{4,}`, "sensitive", "credential: password detected")
	c.addPattern(`(?i)(secret|token)\s*[:=]\s*\S{10,}`, "sensitive", "credential: secret/token detected")
	c.addPattern(`\b[A-Z0-9]{20,}\b`, "suspicious", "possible credential: long uppercase alphanumeric string")

	// SSN pattern (XXX-XX-XXXX).
	c.addPattern(`\b\d{3}-\d{2}-\d{4}\b`, "sensitive", "PII: SSN pattern detected")

	// Credit card patterns (basic Luhn-eligible formats).
	c.addPattern(`\b(?:\d[ -]*?){13,19}\b`, "suspicious", "PII: possible credit card number")

	// Email addresses.
	c.addPattern(`\b[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}\b`, "suspicious", "PII: email address detected")

	// Phone numbers (US-style).
	c.addPattern(`\b(?:\+1[ -]?)?\(?\d{3}\)?[ -]?\d{3}[ -]?\d{4}\b`, "suspicious", "PII: phone number detected")

	// SQL injection patterns.
	c.addPattern(`(?i)(\b(SELECT|INSERT|UPDATE|DELETE|DROP|UNION|ALTER)\b.*\b(FROM|INTO|TABLE|SET|WHERE)\b)`, "malicious", "code injection: SQL")
	c.addPattern(`(?i)(;\s*(DROP|DELETE|UPDATE)\s)`, "malicious", "code injection: SQL statement chaining")

	// Code injection.
	c.addPattern(`(?i)<script[^>]*>`, "malicious", "code injection: script tag")
	c.addPattern(`(?i)eval\s*\(`, "suspicious", "code injection: eval usage")

	// Keyword-based rules (require exact phrase matching, case-insensitive).
	c.addKeywordRule([]string{"how to hack", "bypass security", "exploit vulnerability"}, "suspicious", "harmful instruction inquiry")
	c.addKeywordRule([]string{"jailbreak prompt", "bypass content filter"}, "malicious", "jailbreak attempt")

	return c
}

// Classify evaluates content against all patterns and keywords, returning
// the highest-severity match. Normal content returns Safe: true.
func (c *Classifier) Classify(content string) *ClassificationResult {
	if content == "" {
		return &ClassificationResult{
			Safe:       true,
			Category:   "safe",
			Reason:     "empty content",
			Confidence: 1.0,
		}
	}

	var triggers []string
	highestCategory := "safe"
	highestReason := ""
	highestConfidence := 0.0

	lower := strings.ToLower(content)

	// Check regex patterns.
	for _, p := range c.patterns {
		if p.re.MatchString(content) {
			triggers = append(triggers, p.reason)
			cat, conf := c.categoryPriority(p.category)
			if conf > highestConfidence {
				highestCategory = cat
				highestReason = p.reason
				highestConfidence = conf
			}
		}
	}

	// Check keyword rules.
	for _, k := range c.keywords {
		for _, phrase := range k.phrases {
			if strings.Contains(lower, phrase) {
				triggers = append(triggers, k.reason+": "+phrase)
				cat, conf := c.categoryPriority(k.category)
				if conf > highestConfidence {
					highestCategory = cat
					highestReason = k.reason
					highestConfidence = conf
				}
			}
		}
	}

	if len(triggers) == 0 {
		return &ClassificationResult{
			Safe:       true,
			Category:   "safe",
			Reason:     "no suspicious patterns detected",
			Confidence: 1.0,
		}
	}

	return &ClassificationResult{
		Safe:       false,
		Category:   highestCategory,
		Reason:     highestReason,
		Confidence: highestConfidence,
		Triggers:   triggers,
	}
}

func (c *Classifier) addPattern(pattern, category, reason string) {
	c.patterns = append(c.patterns, &compiledPattern{
		re:       regexp.MustCompile(pattern),
		category: category,
		reason:   reason,
	})
}

func (c *Classifier) addKeywordRule(phrases []string, category, reason string) {
	c.keywords = append(c.keywords, keywordRule{
		phrases:  phrases,
		category: category,
		reason:   reason,
	})
}

// categoryPriority returns (category, confidence) where confidence also encodes
// the severity ordering: malicious > sensitive > suspicious > safe.
func (c *Classifier) categoryPriority(category string) (string, float64) {
	switch category {
	case "malicious":
		return "malicious", 0.95
	case "sensitive":
		return "sensitive", 0.85
	case "suspicious":
		return "suspicious", 0.70
	default:
		return "safe", 0.0
	}
}
