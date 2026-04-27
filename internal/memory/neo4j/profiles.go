package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"agent-memory/internal/profiles"
)

func (c *Client) CreateProfile(ctx context.Context, profile *profiles.UserProfile) error {
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now()
	}
	profile.UpdatedAt = time.Now()

	query := `
		MERGE (p:UserProfile {id: $id})
		SET p.name = $name,
			p.email = $email,
			p.phone = $phone,
			p.avatar = $avatar,
			p.bio = $bio,
			p.location = $location,
			p.timezone = $timezone,
			p.language = $language,
			p.preferences = $preferences,
			p.interests = $interests,
			p.goals = $goals,
			p.attributes = $attributes,
			p.engagement_score = $engagement_score,
			p.trust_score = $trust_score,
			p.last_active_at = datetime($last_active_at),
			p.created_at = datetime($created_at),
			p.updated_at = datetime($updated_at)
		RETURN p.id
	`

	prefsJSON, _ := json.Marshal(profile.Preferences)
	attrsJSON, _ := json.Marshal(profile.Attributes)
	interestsJSON, _ := json.Marshal(profile.Interests)
	goalsJSON, _ := json.Marshal(profile.Goals)

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":               profile.ID,
		"name":             profile.Name,
		"email":            profile.Email,
		"phone":            profile.Phone,
		"avatar":           profile.Avatar,
		"bio":              profile.Bio,
		"location":         profile.Location,
		"timezone":          profile.Timezone,
		"language":         profile.Language,
		"preferences":      string(prefsJSON),
		"interests":        string(interestsJSON),
		"goals":            string(goalsJSON),
		"attributes":       string(attrsJSON),
		"engagement_score": profile.EngagementScore,
		"trust_score":      profile.TrustScore,
		"last_active_at":    profile.LastActiveAt.Format(time.RFC3339),
		"created_at":       profile.CreatedAt.Format(time.RFC3339),
		"updated_at":       profile.UpdatedAt.Format(time.RFC3339),
	})
	return err
}

func (c *Client) GetProfile(ctx context.Context, id string) (*profiles.UserProfile, error) {
	query := `
		MATCH (p:UserProfile {id: $id})
		RETURN p.name, p.email, p.phone, p.avatar, p.bio, p.location,
		       p.timezone, p.language, p.preferences, p.interests, p.goals,
		       p.attributes, p.engagement_score, p.trust_score,
		       p.last_active_at, p.created_at, p.updated_at
	`

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}

	record, err := result.Single(ctx)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %w", err)
	}

	profile := &profiles.UserProfile{ID: id}

	if v, ok := record.Get("p.name"); ok && v != nil {
		profile.Name = v.(string)
	}
	if v, ok := record.Get("p.email"); ok && v != nil {
		profile.Email = v.(string)
	}
	if v, ok := record.Get("p.preferences"); ok && v != nil {
		json.Unmarshal([]byte(v.(string)), &profile.Preferences)
	}
	if v, ok := record.Get("p.interests"); ok && v != nil {
		json.Unmarshal([]byte(v.(string)), &profile.Interests)
	}
	if v, ok := record.Get("p.attributes"); ok && v != nil {
		json.Unmarshal([]byte(v.(string)), &profile.Attributes)
	}

	return profile, nil
}

func (c *Client) UpdateProfile(ctx context.Context, profile *profiles.UserProfile) error {
	return c.CreateProfile(ctx, profile)
}

func (c *Client) DeleteProfile(ctx context.Context, id string) error {
	query := `MATCH (p:UserProfile {id: $id}) DETACH DELETE p`

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	return err
}

func (c *Client) RecordActivity(ctx context.Context, userID, activityType string, metadata map[string]interface{}) error {
	query := `
		MATCH (p:UserProfile {id: $user_id})
		CREATE (p)-[r:ACTIVITY {id: $id, type: $type, timestamp: datetime($timestamp)}]->(a:Activity {
			type: $type,
			metadata: $metadata,
			timestamp: datetime($timestamp)
		})
		SET p.last_active_at = datetime($timestamp)
	`

	metadataJSON, _ := json.Marshal(metadata)

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":        uuid.New().String(),
		"user_id":   userID,
		"type":      activityType,
		"metadata":  string(metadataJSON),
		"timestamp": time.Now().Format(time.RFC3339),
	})
	return err
}

func (c *Client) GetActivityHistory(ctx context.Context, userID string, limit int) ([]*profiles.ContextEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		MATCH (p:UserProfile {id: $user_id})-[:ACTIVITY]->(a:Activity)
		RETURN a.type, a.metadata, a.timestamp
		ORDER BY a.timestamp DESC
		LIMIT $limit
	`

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, map[string]interface{}{
		"user_id": userID,
		"limit":   limit,
	})
	if err != nil {
		return nil, err
	}

	var entries []*profiles.ContextEntry
	for result.Next(ctx) {
		record := result.Record()
		entry := &profiles.ContextEntry{}
		if v, ok := record.Get("a.type"); ok && v != nil {
			entry.Type = v.(string)
		}
		if v, ok := record.Get("a.metadata"); ok && v != nil {
			json.Unmarshal([]byte(v.(string)), &entry.Metadata)
		}
		if v, ok := record.Get("a.timestamp"); ok && v != nil {
			switch t := v.(type) {
			case time.Time:
				entry.Timestamp = t
			}
		}
		entries = append(entries, entry)
	}

	return entries, nil
}