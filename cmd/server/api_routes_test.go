package main

import (
	"os"
	"strings"
	"testing"
)

// Critical routes that SDKs and curl examples depend on.
var requiredRoutes = []string{
	"GET /health",
	"GET /ready",
	"GET /billing/plans",
	"POST /memories",
	"GET /memories",
	"GET /search",
	"GET /search/enhanced",
	"POST /search/hybrid",
	"POST /v3/add",
	"POST /v3/search",
	"GET /profiles/{userID}",
	"GET /skills",
	"GET /compression/stats",
	"POST /stripe/checkout",
}

func TestRequiredRoutesRegistered(t *testing.T) {
	apiPath := "api.go"
	data, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("read %s: %v", apiPath, err)
	}
	source := string(data)

	missing := make([]string, 0)
	for _, route := range requiredRoutes {
		parts := strings.SplitN(route, " ", 2)
		if len(parts) != 2 {
			t.Fatalf("invalid route spec: %s", route)
		}
		method, path := parts[0], parts[1]
		// Routes are registered as HandleFunc("/path" or Handle("/path"
		if !strings.Contains(source, `"`+path+`"`) {
			missing = append(missing, route+" (path)")
			continue
		}
		if method != "" {
			methodPattern := `Methods("` + method + `")`
			if !strings.Contains(source, methodPattern) {
				missing = append(missing, route+" (method)")
			}
		}
	}

	if len(missing) > 0 {
		t.Errorf("missing route registrations in api.go:\n  %s", strings.Join(missing, "\n  "))
	}
}
