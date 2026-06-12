package playground

import (
	"context"
	"testing"
)

func TestCompressionUsesSemanticFallbackForShortUIText(t *testing.T) {
	svc := NewPlaygroundService(nil, nil)
	input := "Edit User Role Change the role for dixithimanshu012@gmail.com user user active"

	resp, err := svc.TestCompression(context.Background(), CompressionTestRequest{
		Text:  input,
		Modes: []string{"extraction", "relational", "radix", "hybrid"},
	})
	if err != nil {
		t.Fatalf("TestCompression returned error: %v", err)
	}

	for _, mode := range []string{"extraction", "relational", "radix", "hybrid"} {
		result := resp.Results[mode]
		if result == nil {
			t.Fatalf("missing %s result", mode)
		}
		if result.Compressed == input {
			t.Fatalf("%s did not compress input", mode)
		}
		if result.Reduction <= 0.05 {
			t.Fatalf("%s reduction = %.3f, want > 0.05", mode, result.Reduction)
		}
	}
}
