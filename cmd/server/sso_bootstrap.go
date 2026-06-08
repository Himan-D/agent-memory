package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"agent-memory/internal/config"
	"agent-memory/internal/sso"
)

func bootstrapSSOManager(ctx context.Context, cfg *config.Config) (*sso.Manager, sso.Store) {
	manager := sso.NewManager()
	var store sso.Store

	if cfg != nil && cfg.SSO.ConfigFile != "" {
		store = sso.NewFileStore(cfg.SSO.ConfigFile)
		configs, err := store.Load(ctx)
		if err != nil {
			log.Printf("warning: SSO provider store unavailable: %v", err)
		} else {
			registerSSOConfigs(ctx, manager, configs, "file")
		}
	}

	if cfg != nil && cfg.SSO.ProvidersJSON != "" {
		configs, err := parseSSOProviderJSON(cfg.SSO.ProvidersJSON)
		if err != nil {
			log.Printf("warning: SSO_PROVIDERS_JSON ignored: %v", err)
		} else {
			registerSSOConfigs(ctx, manager, configs, "env")
		}
	}

	return manager, store
}

func parseSSOProviderJSON(raw string) ([]*sso.Config, error) {
	var wrapped struct {
		Providers []*sso.Config `json:"providers"`
	}
	if strings.HasPrefix(strings.TrimSpace(raw), "{") {
		if err := json.Unmarshal([]byte(raw), &wrapped); err != nil {
			return nil, fmt.Errorf("parse provider object: %w", err)
		}
		return wrapped.Providers, nil
	}

	var configs []*sso.Config
	if err := json.Unmarshal([]byte(raw), &configs); err != nil {
		return nil, fmt.Errorf("parse provider list: %w", err)
	}
	return configs, nil
}

func registerSSOConfigs(ctx context.Context, manager *sso.Manager, configs []*sso.Config, source string) {
	for _, cfg := range configs {
		if err := ctx.Err(); err != nil {
			log.Printf("warning: stop loading SSO providers from %s: %v", source, err)
			return
		}
		if cfg == nil || cfg.TenantID == "" {
			log.Printf("warning: skipped SSO provider from %s with missing tenant_id", source)
			continue
		}
		if err := manager.RegisterProvider(cfg.TenantID, cfg); err != nil {
			log.Printf("warning: skipped SSO provider %s from %s: %v", cfg.TenantID, source, err)
			continue
		}
		log.Printf("loaded SSO provider tenant=%s type=%s source=%s", cfg.TenantID, cfg.ProviderType, source)
	}
}
