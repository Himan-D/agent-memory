package loaders

import (
	"testing"
)

func TestNewPDFLoader(t *testing.T) {
	loader := NewPDFLoader()
	if loader.maxChunkSize != 1000 {
		t.Errorf("Expected maxChunkSize to be 1000, got %d", loader.maxChunkSize)
	}
}
