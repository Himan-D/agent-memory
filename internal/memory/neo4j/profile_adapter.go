package neo4j

import (
	"context"

	"agent-memory/internal/profiles"
)

type ProfileAdapter struct {
	client *Client
}

func NewProfileAdapter(client *Client) *ProfileAdapter {
	return &ProfileAdapter{client: client}
}

func (a *ProfileAdapter) CreateProfile(ctx context.Context, profile *profiles.UserProfile) error {
	return a.client.CreateProfile(ctx, profile)
}

func (a *ProfileAdapter) GetProfile(ctx context.Context, id string) (*profiles.UserProfile, error) {
	return a.client.GetProfile(ctx, id)
}

func (a *ProfileAdapter) UpdateProfile(ctx context.Context, profile *profiles.UserProfile) error {
	return a.client.UpdateProfile(ctx, profile)
}

func (a *ProfileAdapter) DeleteProfile(ctx context.Context, id string) error {
	return a.client.DeleteProfile(ctx, id)
}

func (a *ProfileAdapter) RecordActivity(ctx context.Context, userID, activityType string, metadata map[string]interface{}) error {
	return a.client.RecordActivity(ctx, userID, activityType, metadata)
}

func (a *ProfileAdapter) GetActivityHistory(ctx context.Context, userID string, limit int) ([]*profiles.ContextEntry, error) {
	return a.client.GetActivityHistory(ctx, userID, limit)
}