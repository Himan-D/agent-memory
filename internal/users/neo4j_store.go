package users

import (
	"context"
	"fmt"
	"time"

	"agent-memory/internal/memory/neo4j"

	"github.com/google/uuid"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

type Neo4jStore struct {
	client *neo4j.Client
}

func NewNeo4jStore(client *neo4j.Client) *Neo4jStore {
	return &Neo4jStore{client: client}
}

func (s *Neo4jStore) Init(ctx context.Context) error {
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	constraints := []string{
		"CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE",
		"CREATE CONSTRAINT user_email IF NOT EXISTS FOR (u:User) REQUIRE u.email IS UNIQUE",
		"CREATE CONSTRAINT invite_id IF NOT EXISTS FOR (i:Invite) REQUIRE i.id IS UNIQUE",
	}
	for _, c := range constraints {
		if _, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx, c, nil)
			return nil, err
		}); err != nil {
			return fmt.Errorf("users init constraint: %w", err)
		}
	}
	return nil
}

func (s *Neo4jStore) ListUsers() ([]User, error) {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, "MATCH (u:User) RETURN u ORDER BY u.created_at DESC", nil)
		if err != nil {
			return nil, err
		}
		var users []User
		for records.Next(ctx) {
			user, err := recordToUser(records.Record())
			if err != nil {
				return nil, err
			}
			users = append(users, *user)
		}
		return users, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	if result == nil {
		return []User{}, nil
	}
	return result.([]User), nil
}

func (s *Neo4jStore) GetUser(id uuid.UUID) (*User, error) {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, "MATCH (u:User {id: $id}) RETURN u", map[string]any{"id": id.String()})
		if err != nil {
			return nil, err
		}
		if !records.Next(ctx) {
			return nil, fmt.Errorf("user not found")
		}
		return recordToUser(records.Record())
	})
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return result.(*User), nil
}

func (s *Neo4jStore) CreateUser(user *User) error {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	_, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		params := map[string]any{
			"id":            user.ID.String(),
			"email":         user.Email,
			"name":          user.Name,
			"role":          string(user.Role),
			"status":        user.Status,
			"avatar_url":    user.AvatarURL,
			"password_hash": user.PasswordHash,
			"created_at":    user.CreatedAt,
			"updated_at":    user.UpdatedAt,
		}
		if user.LastLogin != nil {
			params["last_login"] = *user.LastLogin
		}
		_, err := tx.Run(ctx, `
			CREATE (u:User {
				id: $id,
				email: $email,
				name: $name,
				role: $role,
				status: $status,
				avatar_url: $avatar_url,
				password_hash: $password_hash,
				created_at: datetime($created_at),
				updated_at: datetime($updated_at)
			})
			WITH u
			CALL apoc.do.when($last_login IS NOT NULL,
				'SET u.last_login = datetime($last_login)',
				'',
				{u: u, last_login: $last_login})
			YIELD value
			RETURN u
		`, params)
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *Neo4jStore) UpdateUser(id uuid.UUID, updates *UpdateUserRequest) error {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	_, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, "MATCH (u:User {id: $id}) RETURN u", map[string]any{"id": id.String()})
		if err != nil {
			return nil, err
		}
		if !result.Next(ctx) {
			return nil, fmt.Errorf("user not found")
		}

		setClauses := "SET u.updated_at = datetime()"
		params := map[string]any{"id": id.String()}
		if updates.Name != "" {
			setClauses += ", u.name = $name"
			params["name"] = updates.Name
		}
		if updates.Role != "" {
			setClauses += ", u.role = $role"
			params["role"] = string(updates.Role)
		}
		if updates.Status != "" {
			setClauses += ", u.status = $status"
			params["status"] = updates.Status
		}
		if updates.PasswordHash != "" {
			setClauses += ", u.password_hash = $password_hash"
			params["password_hash"] = updates.PasswordHash
		}

		_, err = tx.Run(ctx, "MATCH (u:User {id: $id}) "+setClauses, params)
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (s *Neo4jStore) DeleteUser(id uuid.UUID) error {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	_, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, "MATCH (u:User {id: $id}) RETURN u", map[string]any{"id": id.String()})
		if err != nil {
			return nil, err
		}
		if !result.Next(ctx) {
			return nil, fmt.Errorf("user not found")
		}
		_, err = tx.Run(ctx, "MATCH (u:User {id: $id}) DETACH DELETE u", map[string]any{"id": id.String()})
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *Neo4jStore) ListInvites() ([]Invite, error) {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, "MATCH (i:Invite) RETURN i ORDER BY i.created_at DESC", nil)
		if err != nil {
			return nil, err
		}
		var invites []Invite
		for records.Next(ctx) {
			invite, err := recordToInvite(records.Record())
			if err != nil {
				return nil, err
			}
			invites = append(invites, *invite)
		}
		return invites, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	if result == nil {
		return []Invite{}, nil
	}
	return result.([]Invite), nil
}

func (s *Neo4jStore) GetInvite(id uuid.UUID) (*Invite, error) {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, "MATCH (i:Invite {id: $id}) RETURN i", map[string]any{"id": id.String()})
		if err != nil {
			return nil, err
		}
		if !records.Next(ctx) {
			return nil, fmt.Errorf("invite not found")
		}
		return recordToInvite(records.Record())
	})
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	return result.(*Invite), nil
}

func (s *Neo4jStore) CreateInvite(invite *Invite) error {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	_, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
			CREATE (i:Invite {
				id: $id,
				email: $email,
				role: $role,
				status: $status,
				invited_by: $invited_by,
				expires_at: datetime($expires_at),
				created_at: datetime($created_at)
			})
		`, map[string]any{
			"id":         invite.ID.String(),
			"email":      invite.Email,
			"role":       string(invite.Role),
			"status":     invite.Status,
			"invited_by": invite.InvitedBy.String(),
			"expires_at": invite.ExpiresAt,
			"created_at": invite.CreatedAt,
		})
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	return nil
}

func (s *Neo4jStore) UpdateInvite(id uuid.UUID, status string) error {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	_, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, "MATCH (i:Invite {id: $id}) RETURN i", map[string]any{"id": id.String()})
		if err != nil {
			return nil, err
		}
		if !result.Next(ctx) {
			return nil, fmt.Errorf("invite not found")
		}
		_, err = tx.Run(ctx, "MATCH (i:Invite {id: $id}) SET i.status = $status", map[string]any{
			"id":     id.String(),
			"status": status,
		})
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("update invite: %w", err)
	}
	return nil
}

func (s *Neo4jStore) DeleteInvite(id uuid.UUID) error {
	ctx := context.Background()
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	_, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, "MATCH (i:Invite {id: $id}) RETURN i", map[string]any{"id": id.String()})
		if err != nil {
			return nil, err
		}
		if !result.Next(ctx) {
			return nil, fmt.Errorf("invite not found")
		}
		_, err = tx.Run(ctx, "MATCH (i:Invite {id: $id}) DETACH DELETE i", map[string]any{"id": id.String()})
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}
	return nil
}

func neo4jTimeToTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case neo4jdriver.Date:
		return t.Time(), nil
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time.Time", v)
	}
}

func recordToUser(rec *neo4jdriver.Record) (*User, error) {
	nodeVal, ok := rec.Get("u")
	if !ok {
		return nil, fmt.Errorf("missing 'u' in record")
	}
	node, ok := nodeVal.(neo4jdriver.Node)
	if !ok {
		return nil, fmt.Errorf("'u' is not a Node")
	}
	props := node.Props

	idStr, _ := props["id"].(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	user := &User{
		ID:           id,
		Email:        strProp(props, "email"),
		Name:         strProp(props, "name"),
		Role:         Role(strProp(props, "role")),
		Status:       strProp(props, "status"),
		AvatarURL:    strProp(props, "avatar_url"),
		PasswordHash: strProp(props, "password_hash"),
	}

	if v, ok := props["created_at"]; ok {
		t, err := neo4jTimeToTime(v)
		if err == nil {
			user.CreatedAt = t
		}
	}
	if v, ok := props["updated_at"]; ok {
		t, err := neo4jTimeToTime(v)
		if err == nil {
			user.UpdatedAt = t
		}
	}
	if v, ok := props["last_login"]; ok && v != nil {
		t, err := neo4jTimeToTime(v)
		if err == nil {
			user.LastLogin = &t
		}
	}

	return user, nil
}

func recordToInvite(rec *neo4jdriver.Record) (*Invite, error) {
	nodeVal, ok := rec.Get("i")
	if !ok {
		return nil, fmt.Errorf("missing 'i' in record")
	}
	node, ok := nodeVal.(neo4jdriver.Node)
	if !ok {
		return nil, fmt.Errorf("'i' is not a Node")
	}
	props := node.Props

	idStr, _ := props["id"].(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid invite id: %w", err)
	}

	invitedByStr, _ := props["invited_by"].(string)
	invitedBy, err := uuid.Parse(invitedByStr)
	if err != nil {
		return nil, fmt.Errorf("invalid invited_by: %w", err)
	}

	invite := &Invite{
		ID:        id,
		Email:     strProp(props, "email"),
		Role:      Role(strProp(props, "role")),
		Status:    strProp(props, "status"),
		InvitedBy: invitedBy,
	}

	if v, ok := props["expires_at"]; ok {
		t, err := neo4jTimeToTime(v)
		if err == nil {
			invite.ExpiresAt = t
		}
	}
	if v, ok := props["created_at"]; ok {
		t, err := neo4jTimeToTime(v)
		if err == nil {
			invite.CreatedAt = t
		}
	}

	return invite, nil
}

func strProp(props map[string]any, key string) string {
	v, _ := props[key].(string)
	return v
}
