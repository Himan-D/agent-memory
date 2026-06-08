package main

import "testing"

func TestParseSSOProviderJSON(t *testing.T) {
	configs, err := parseSSOProviderJSON(`{"providers":[{"tenant_id":"acme","provider_type":"oidc","client_id":"id","client_secret":"secret","issuer_url":"https://idp.example.com"}]}`)
	if err != nil {
		t.Fatalf("parse wrapped: %v", err)
	}
	if len(configs) != 1 || configs[0].TenantID != "acme" {
		t.Fatalf("unexpected wrapped configs: %+v", configs)
	}

	configs, err = parseSSOProviderJSON(`[{"tenant_id":"beta","provider_type":"ldap","issuer_url":"ldap://ldap.example.com"}]`)
	if err != nil {
		t.Fatalf("parse array: %v", err)
	}
	if len(configs) != 1 || configs[0].TenantID != "beta" {
		t.Fatalf("unexpected array configs: %+v", configs)
	}

	configs, err = parseSSOProviderJSON(`{"providers":[]}`)
	if err != nil {
		t.Fatalf("parse empty wrapped: %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("expected empty configs, got %+v", configs)
	}
}
