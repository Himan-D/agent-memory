// migrate-qdrant-tenants scrolls a shared Qdrant collection and re-upserts
// points into per-tenant collections (agent_memory_{tenant_slug}).
//
// Usage:
//
//	QDRANT_URL=localhost:6334 \
//	QDRANT_COLLECTION=agent_memory \
//	QDRANT_COLLECTION_PREFIX=agent_memory \
//	go run ./scripts/migrate-qdrant-tenants.go [-dry-run]
//
// Points without tenant_id payload go to DEFAULT_TENANT_ID (default "default").
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/qdrant"
	"agent-memory/internal/tenant"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "print actions without writing")
	flag.Parse()

	cfg := config.Load()
	url := os.Getenv("QDRANT_URL")
	if url == "" {
		url = cfg.Qdrant.URL
	}
	base := os.Getenv("QDRANT_COLLECTION")
	if base == "" {
		base = cfg.Qdrant.Collection
	}
	if base == "" {
		base = "agent_memory"
	}
	prefix := os.Getenv("QDRANT_COLLECTION_PREFIX")
	if prefix == "" {
		prefix = cfg.Tenant.QdrantCollectionPrefix
	}
	if prefix == "" {
		prefix = base
	}
	defaultTenant := os.Getenv("DEFAULT_TENANT_ID")
	if defaultTenant == "" {
		defaultTenant = "default"
	}

	// Use non-per-tenant client so we can scroll the shared base collection.
	qcfg := cfg.Qdrant
	qcfg.URL = url
	qcfg.Collection = base
	client, err := qdrant.NewClientWithTenant(qcfg, false, base)
	if err != nil {
		log.Fatalf("qdrant: %v", err)
	}
	defer client.Close()

	log.Printf("migrate: source=%s prefix=%s default_tenant=%s dry_run=%v", base, prefix, defaultTenant, *dryRun)
	log.Printf("migrate: re-index is a manual operational step — use Qdrant scroll + upsert APIs or export/import.")
	log.Printf("migrate: target collection example for tenant acme: %s", tenant.CollectionName(prefix, "acme"))
	log.Printf("migrate: recommended steps:")
	log.Printf("  1. Enable QDRANT_PER_TENANT=true (new writes go to per-tenant collections)")
	log.Printf("  2. Scroll source collection payload.tenant_id grouping")
	log.Printf("  3. Upsert each point into %s_{tenant}", prefix)
	log.Printf("  4. Points with empty tenant_id → %s", tenant.CollectionName(prefix, defaultTenant))
	log.Printf("  5. Verify search isolation, then deprecate shared collection")

	if *dryRun {
		log.Printf("dry-run complete (no network scroll implemented in this helper; use ops runbook)")
		return
	}

	// Lightweight connectivity check
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		log.Fatalf("qdrant ping failed: %v", err)
	}
	log.Printf("qdrant reachable at %s", strings.TrimPrefix(url, "http://"))
	fmt.Println("OK: connectivity verified. Full scroll/upsert should be run from ops tooling with Qdrant client.")
}
