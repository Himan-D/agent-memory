package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// SecretManager loads secrets from GCP Secret Manager via the REST API.
type SecretManager struct {
	projectID string
	client    *http.Client
}

// NewSecretManager returns a SecretManager for the given GCP project.
func NewSecretManager(projectID string) *SecretManager {
	return &SecretManager{projectID: projectID, client: http.DefaultClient}
}

// GetSecret fetches the latest version of a secret from GCP Secret Manager.
// Authentication uses the GOOGLE_ACCESS_TOKEN env var when present; in
// production this should be replaced with Application Default Credentials.
func (sm *SecretManager) GetSecret(ctx context.Context, name string) (string, error) {
	url := fmt.Sprintf(
		"https://secretmanager.googleapis.com/v1/projects/%s/secrets/%s/versions/latest:access",
		sm.projectID, name,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	if token := os.Getenv("GOOGLE_ACCESS_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := sm.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("secret manager error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Payload struct {
			Data string `json:"data"`
		} `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Payload.Data, nil
}

// LoadSecrets populates config fields from GCP Secret Manager when
// GCP_USE_SECRET_MANAGER=true. Failures are non-fatal — env vars serve as
// the fallback so the service still starts without Secret Manager access.
func LoadSecrets(cfg *Config) error {
	if !cfg.GCP.UseSecretManager || cfg.GCP.ProjectID == "" {
		return nil
	}
	sm := NewSecretManager(cfg.GCP.ProjectID)
	ctx := context.Background()

	secrets := map[string]*string{
		"neo4j-password":    &cfg.Neo4j.Password,
		"qdrant-api-key":    &cfg.Qdrant.APIKey,
		"llm-api-key":       &cfg.LLM.APIKey,
		"jwt-secret":        &cfg.Auth.JWTSecret,
		"openai-api-key":    &cfg.OpenAI.APIKey,
		"anthropic-api-key": &cfg.Compression.AnthropicAPIKey,
	}

	for name, target := range secrets {
		val, err := sm.GetSecret(ctx, name)
		if err == nil && val != "" {
			*target = val
		}
		// Non-fatal — fall back to env vars if Secret Manager is unavailable.
	}
	return nil
}
