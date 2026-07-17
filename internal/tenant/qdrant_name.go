package tenant

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// Qdrant collection name limits (Qdrant allows up to 255 chars; keep conservative).
const maxCollectionNameLen = 64

var nonSlug = regexp.MustCompile(`[^a-z0-9_]+`)

// CollectionName builds a per-tenant Qdrant collection name.
// Format: {prefix}_{sanitized_tenant} — if too long, suffix with short hash.
func CollectionName(prefix, tenantID string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "agent_memory"
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = "default"
	}

	slug := sanitizeSlug(tenantID)
	name := prefix + "_" + slug
	if len(name) <= maxCollectionNameLen {
		return name
	}

	// Truncate slug and append hash for uniqueness.
	sum := sha256.Sum256([]byte(tenantID))
	hash := hex.EncodeToString(sum[:])[:8]
	maxSlug := maxCollectionNameLen - len(prefix) - 1 - len(hash) - 1
	if maxSlug < 4 {
		maxSlug = 4
	}
	if len(slug) > maxSlug {
		slug = slug[:maxSlug]
	}
	return prefix + "_" + slug + "_" + hash
}

func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = nonSlug.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "default"
	}
	// Collapse repeated underscores
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return s
}

// ValidateSlug returns an error message if the slug is invalid for tenant creation.
func ValidateSlug(slug string) string {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return "slug is required"
	}
	if len(slug) < 2 {
		return "slug must be at least 2 characters"
	}
	if len(slug) > 48 {
		return "slug must be at most 48 characters"
	}
	if sanitizeSlug(slug) != strings.ReplaceAll(slug, "-", "_") && sanitizeSlug(slug) != slug {
		// allow hyphens in input (normalized to underscore)
		normalized := sanitizeSlug(slug)
		if normalized == "" {
			return "slug must contain letters or numbers"
		}
	}
	return ""
}
