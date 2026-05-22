package privacy

import (
	"testing"
)

func TestFilter_APIKey(t *testing.T) {
	f := NewDefaultFilter()
	result := f.Filter("my api_key=sk-abc123def456ghi789jkl012mno345")
	if !result.WasFiltered {
		t.Error("expected content to be filtered")
	}
	if !contains(result.Content, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in output, got: %s", result.Content)
	}
}

func TestFilter_OpenAIKey(t *testing.T) {
	f := NewDefaultFilter()
	result := f.Filter("key is sk-proj-1234567890abcdefghij")
	if !result.WasFiltered {
		t.Error("expected OpenAI key to be filtered")
	}
}

func TestFilter_BearerToken(t *testing.T) {
	f := NewDefaultFilter()
	result := f.Filter("Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")
	if !result.WasFiltered {
		t.Error("expected bearer token to be filtered")
	}
}

func TestFilter_JWT(t *testing.T) {
	f := NewDefaultFilter()
	result := f.Filter("token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")
	if !result.WasFiltered {
		t.Error("expected JWT to be filtered")
	}
}

func TestFilter_Password(t *testing.T) {
	f := NewDefaultFilter()
	result := f.Filter("password=supersecret123")
	if !result.WasFiltered {
		t.Error("expected password to be filtered")
	}
}

func TestFilter_PrivateKey(t *testing.T) {
	f := NewDefaultFilter()
	content := "private_key=-----BEGIN RSA PRIVATE KEY-----\nMIIBogIBAAJBALRiMLAHudeSA\n-----END RSA PRIVATE KEY-----"
	result := f.Filter(content)
	if !result.WasFiltered {
		t.Error("expected private key to be filtered")
	}
}

func TestFilter_AWSKey(t *testing.T) {
	f := NewDefaultFilter()
	result := f.Filter("aws_access_key_id=AKIAIOSFODNN7EXAMPLE")
	if !result.WasFiltered {
		t.Error("expected AWS key to be filtered")
	}
}

func TestFilter_StripeKey(t *testing.T) {
	f := NewDefaultFilter()
	result := f.Filter("key=sk_live_abcdef1234567890abcdef")
	if !result.WasFiltered {
		t.Error("expected Stripe key to be filtered")
	}
}

func TestFilter_NoSensitiveData(t *testing.T) {
	f := NewDefaultFilter()
	result := f.Filter("the user prefers dark mode and likes cats")
	if result.WasFiltered {
		t.Error("expected no filtering for safe content")
	}
	if result.Content != "the user prefers dark mode and likes cats" {
		t.Errorf("content was modified unexpectedly: %s", result.Content)
	}
}

func TestFilter_Disabled(t *testing.T) {
	f := NewFilter(FilterConfig{Enabled: false})
	result := f.Filter("api_key=sk-abc123def456ghi789jkl012mno345")
	if result.WasFiltered {
		t.Error("disabled filter should not filter")
	}
}

func TestFilterMetadata(t *testing.T) {
	f := NewDefaultFilter()
	meta := map[string]interface{}{
		"token":  "sk-abc123def456ghi789jkl012mno345",
		"normal": "hello world",
		"number": 42,
	}
	filtered := f.FilterMetadata(meta)
	if filtered["token"] == meta["token"] {
		t.Error("expected token to be filtered in metadata")
	}
	if filtered["normal"] != "hello world" {
		t.Error("expected normal value unchanged")
	}
	if filtered["number"] != 42 {
		t.Error("expected non-string value unchanged")
	}
}

func TestContainsSensitiveData(t *testing.T) {
	f := NewDefaultFilter()
	if !f.ContainsSensitiveData("api_key=sk-abc123") {
		t.Error("expected to detect sensitive data")
	}
	if f.ContainsSensitiveData("just normal text") {
		t.Error("expected no sensitive data detection")
	}
}

func TestJaccardSimilarity_Identical(t *testing.T) {
	score := JaccardSimilarity("the cat sat on the mat", "the cat sat on the mat")
	if score != 1.0 {
		t.Errorf("expected 1.0 for identical strings, got %f", score)
	}
}

func TestJaccardSimilarity_NoOverlap(t *testing.T) {
	score := JaccardSimilarity("alpha beta", "gamma delta")
	if score != 0 {
		t.Errorf("expected 0 for disjoint strings, got %f", score)
	}
}

func TestJaccardSimilarity_Partial(t *testing.T) {
	score := JaccardSimilarity("the cat sat on the mat", "the dog sat on the rug")
	if score <= 0 || score >= 1 {
		t.Errorf("expected partial overlap, got %f", score)
	}
}

func TestJaccardSimilarity_Empty(t *testing.T) {
	score := JaccardSimilarity("", "")
	if score != 1.0 {
		t.Errorf("expected 1.0 for both empty, got %f", score)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
