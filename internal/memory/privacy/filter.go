package privacy

import (
	"regexp"
	"strings"
)

type FilterConfig struct {
	Enabled        bool
	RedactMode     string
	CustomPatterns []string
}

type Filter struct {
	patterns    []*regexp.Regexp
	replacement string
	enabled     bool
}

type FilterResult struct {
	Content     string
	Redactions  []Redaction
	WasFiltered bool
}

type Redaction struct {
	Type  string
	Field string
}

var defaultPatterns = []struct {
	regex string
	typ   string
}{
	{`(?i)(api[_-]?key|apikey)\s*[:=]\s*['"]?[a-zA-Z0-9\-_]{8,}['"]?`, "api_key"},
	{`(?i)(secret[_-]?key|secretkey)\s*[:=]\s*['"]?[a-zA-Z0-9\-_]{8,}['"]?`, "secret_key"},
	{`(?i)(access[_-]?token|accesstoken)\s*[:=]\s*['"]?[a-zA-Z0-9\-_.]{8,}['"]?`, "access_token"},
	{`(?i)(bearer)\s+[a-zA-Z0-9\-_.]{20,}`, "bearer_token"},
	{`sk-[a-zA-Z0-9\-_]{20,}`, "openai_key"},
	{`sk_live_[a-zA-Z0-9]{20,}`, "stripe_live_key"},
	{`sk_test_[a-zA-Z0-9]{20,}`, "stripe_test_key"},
	{`(?i)(password|passwd|pwd)\s*[:=]\s*['"]?[^\s'"]{8,}['"]?`, "password"},
	{`(?i)(private[_-]?key|privatekey)\s*[:=]\s*['"]?-----BEGIN [A-Z ]+ PRIVATE KEY-----[\s\S]*?-----END [A-Z ]+ PRIVATE KEY-----['"]?`, "private_key"},
	{`-----BEGIN [A-Z ]+ PRIVATE KEY-----[\s\S]*?-----END [A-Z ]+ PRIVATE KEY-----`, "private_key_block"},
	{`(?i)(aws[_-]?access[_-]?key[_-]?id)\s*[:=]\s*['"]?AKIA[0-9A-Z]{16}['"]?`, "aws_access_key"},
	{`(?i)(aws[_-]?secret[_-]?access[_-]?key)\s*[:=]\s*['"]?[a-zA-Z0-9/+=]{40}['"]?`, "aws_secret_key"},
	{`eyJ[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}`, "jwt_token"},
}

func NewFilter(config FilterConfig) *Filter {
	f := &Filter{
		replacement: "[REDACTED]",
		enabled:     config.Enabled,
	}

	patterns := defaultPatterns
	for _, p := range patterns {
		re := regexp.MustCompile(p.regex)
		f.patterns = append(f.patterns, re)
	}

	for _, custom := range config.CustomPatterns {
		re := regexp.MustCompile(custom)
		f.patterns = append(f.patterns, re)
	}

	return f
}

func NewDefaultFilter() *Filter {
	return NewFilter(FilterConfig{Enabled: true})
}

func (f *Filter) Filter(content string) *FilterResult {
	if !f.enabled {
		return &FilterResult{Content: content, WasFiltered: false}
	}

	result := &FilterResult{
		Content: content,
	}

	filtered := content
	for _, pattern := range f.patterns {
		matches := pattern.FindAllString(filtered, -1)
		if len(matches) > 0 {
			filtered = pattern.ReplaceAllString(filtered, f.replacement)
			result.WasFiltered = true
			for range matches {
				result.Redactions = append(result.Redactions, Redaction{
					Type: "sensitive_data",
				})
			}
		}
	}

	result.Content = filtered
	return result
}

func (f *Filter) FilterMetadata(metadata map[string]interface{}) map[string]interface{} {
	if !f.enabled || metadata == nil {
		return metadata
	}

	filtered := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			result := f.Filter(str)
			filtered[k] = result.Content
		} else {
			filtered[k] = v
		}
	}
	return filtered
}

func (f *Filter) ContainsSensitiveData(content string) bool {
	if !f.enabled {
		return false
	}
	for _, pattern := range f.patterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func JaccardSimilarity(a, b string) float64 {
	setA := tokenSet(a)
	setB := tokenSet(b)

	if len(setA) == 0 && len(setB) == 0 {
		return 1.0
	}

	intersection := 0
	for token := range setA {
		if setB[token] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func tokenSet(text string) map[string]bool {
	text = strings.ToLower(text)
	tokens := strings.Fields(text)
	set := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		t = strings.Trim(t, ".,;:!?\"'()[]{}")
		if len(t) > 2 {
			set[t] = true
		}
	}
	return set
}
