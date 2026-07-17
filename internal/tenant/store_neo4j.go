package tenant

import (
	"context"
	"fmt"
	"time"
)

// Neo4jRunner abstracts the subset of Neo4j client needed for tenant persistence.
type Neo4jRunner interface {
	// RunRead executes a Cypher read and returns raw records as maps.
	RunRead(ctx context.Context, cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
	// RunWrite executes a Cypher write and returns raw records as maps.
	RunWrite(ctx context.Context, cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
}

// Neo4jStore persists tenants, memberships, and invites in Neo4j.
type Neo4jStore struct {
	db Neo4jRunner
}

// NewNeo4jStore creates a durable tenant store.
func NewNeo4jStore(db Neo4jRunner) *Neo4jStore {
	return &Neo4jStore{db: db}
}

// EnsureSchema creates indexes/constraints for tenant nodes.
func (s *Neo4jStore) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE CONSTRAINT tenant_id_unique IF NOT EXISTS FOR (t:Tenant) REQUIRE t.id IS UNIQUE`,
		`CREATE CONSTRAINT tenant_slug_unique IF NOT EXISTS FOR (t:Tenant) REQUIRE t.slug IS UNIQUE`,
		`CREATE INDEX tenant_status_idx IF NOT EXISTS FOR (t:Tenant) ON (t.status)`,
		`CREATE INDEX invite_token_idx IF NOT EXISTS FOR (i:TenantInvite) ON (i.token)`,
	}
	for _, q := range stmts {
		if _, err := s.db.RunWrite(ctx, q, nil); err != nil {
			// Non-fatal: older Neo4j or existing constraints
			_ = err
		}
	}
	return nil
}

func (s *Neo4jStore) CreateTenant(ctx context.Context, t *Tenant) error {
	_, err := s.db.RunWrite(ctx, `
		CREATE (t:Tenant {
			id: $id,
			slug: $slug,
			name: $name,
			status: $status,
			plan: $plan,
			created_by: $created_by,
			created_at: datetime($created_at),
			updated_at: datetime($updated_at),
			member_count: $member_count
		})
	`, map[string]interface{}{
		"id":           t.ID,
		"slug":         t.Slug,
		"name":         t.Name,
		"status":       string(t.Status),
		"plan":         t.Plan,
		"created_by":   t.CreatedBy,
		"created_at":   t.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":   t.UpdatedAt.UTC().Format(time.RFC3339),
		"member_count": t.MemberCount,
	})
	return err
}

func (s *Neo4jStore) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	rows, err := s.db.RunRead(ctx, `
		MATCH (t:Tenant {id: $id})
		RETURN t.id AS id, t.slug AS slug, t.name AS name, t.status AS status,
		       t.plan AS plan, t.created_by AS created_by, t.member_count AS member_count,
		       t.created_at AS created_at, t.updated_at AS updated_at
	`, map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("tenant: not found")
	}
	return rowToTenant(rows[0]), nil
}

func (s *Neo4jStore) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	rows, err := s.db.RunRead(ctx, `
		MATCH (t:Tenant {slug: $slug})
		RETURN t.id AS id, t.slug AS slug, t.name AS name, t.status AS status,
		       t.plan AS plan, t.created_by AS created_by, t.member_count AS member_count,
		       t.created_at AS created_at, t.updated_at AS updated_at
	`, map[string]interface{}{"slug": slug})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("tenant: not found")
	}
	return rowToTenant(rows[0]), nil
}

func (s *Neo4jStore) UpdateTenant(ctx context.Context, t *Tenant) error {
	_, err := s.db.RunWrite(ctx, `
		MATCH (t:Tenant {id: $id})
		SET t.name = $name, t.plan = $plan, t.status = $status,
		    t.updated_at = datetime($updated_at), t.member_count = $member_count
	`, map[string]interface{}{
		"id":           t.ID,
		"name":         t.Name,
		"plan":         t.Plan,
		"status":       string(t.Status),
		"updated_at":   t.UpdatedAt.UTC().Format(time.RFC3339),
		"member_count": t.MemberCount,
	})
	return err
}

func (s *Neo4jStore) ListTenantsForUser(ctx context.Context, userID string) ([]Tenant, error) {
	rows, err := s.db.RunRead(ctx, `
		MATCH (u:User {id: $user_id})-[m:MEMBER_OF]->(t:Tenant)
		RETURN t.id AS id, t.slug AS slug, t.name AS name, t.status AS status,
		       t.plan AS plan, t.created_by AS created_by, t.member_count AS member_count,
		       t.created_at AS created_at, t.updated_at AS updated_at
		ORDER BY t.name
	`, map[string]interface{}{"user_id": userID})
	if err != nil {
		// Fallback: membership may be stored without User node
		rows, err = s.db.RunRead(ctx, `
			MATCH (t:Tenant)<-[m:MEMBER_OF]-(u)
			WHERE u.id = $user_id OR m.user_id = $user_id
			RETURN DISTINCT t.id AS id, t.slug AS slug, t.name AS name, t.status AS status,
			       t.plan AS plan, t.created_by AS created_by, t.member_count AS member_count,
			       t.created_at AS created_at, t.updated_at AS updated_at
			ORDER BY t.name
		`, map[string]interface{}{"user_id": userID})
		if err != nil {
			return nil, err
		}
	}
	out := make([]Tenant, 0, len(rows))
	for _, row := range rows {
		if t := rowToTenant(row); t != nil {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (s *Neo4jStore) ListAllTenants(ctx context.Context, limit, offset int) ([]Tenant, int, error) {
	countRows, err := s.db.RunRead(ctx, `MATCH (t:Tenant) RETURN count(t) AS c`, nil)
	if err != nil {
		return nil, 0, err
	}
	total := 0
	if len(countRows) > 0 {
		total = asInt(countRows[0]["c"])
	}
	rows, err := s.db.RunRead(ctx, `
		MATCH (t:Tenant)
		RETURN t.id AS id, t.slug AS slug, t.name AS name, t.status AS status,
		       t.plan AS plan, t.created_by AS created_by, t.member_count AS member_count,
		       t.created_at AS created_at, t.updated_at AS updated_at
		ORDER BY t.created_at DESC
		SKIP $offset LIMIT $limit
	`, map[string]interface{}{"offset": offset, "limit": limit})
	if err != nil {
		return nil, 0, err
	}
	out := make([]Tenant, 0, len(rows))
	for _, row := range rows {
		if t := rowToTenant(row); t != nil {
			out = append(out, *t)
		}
	}
	return out, total, nil
}

func (s *Neo4jStore) AddMember(ctx context.Context, m *Membership) error {
	_, err := s.db.RunWrite(ctx, `
		MATCH (t:Tenant {id: $tenant_id})
		MERGE (u:User {id: $user_id})
		ON CREATE SET u.email = $email, u.created_at = datetime()
		ON MATCH SET u.email = CASE WHEN $email <> '' THEN $email ELSE u.email END
		MERGE (u)-[r:MEMBER_OF]->(t)
		SET r.role = $role, r.email = $email, r.created_at = datetime($created_at), r.user_id = $user_id
		WITH t
		OPTIONAL MATCH (t)<-[all:MEMBER_OF]-()
		WITH t, count(all) AS cnt
		SET t.member_count = cnt
	`, map[string]interface{}{
		"tenant_id":  m.TenantID,
		"user_id":    m.UserID,
		"email":      m.Email,
		"role":       string(m.Role),
		"created_at": m.CreatedAt.UTC().Format(time.RFC3339),
	})
	return err
}

func (s *Neo4jStore) RemoveMember(ctx context.Context, tenantID, userID string) error {
	_, err := s.db.RunWrite(ctx, `
		MATCH (u:User {id: $user_id})-[r:MEMBER_OF]->(t:Tenant {id: $tenant_id})
		DELETE r
		WITH t
		OPTIONAL MATCH (t)<-[all:MEMBER_OF]-()
		WITH t, count(all) AS cnt
		SET t.member_count = cnt
	`, map[string]interface{}{"tenant_id": tenantID, "user_id": userID})
	return err
}

func (s *Neo4jStore) GetMember(ctx context.Context, tenantID, userID string) (*Membership, error) {
	rows, err := s.db.RunRead(ctx, `
		MATCH (u:User {id: $user_id})-[r:MEMBER_OF]->(t:Tenant {id: $tenant_id})
		RETURN r.role AS role, r.email AS email, r.created_at AS created_at,
		       $user_id AS user_id, $tenant_id AS tenant_id
	`, map[string]interface{}{"tenant_id": tenantID, "user_id": userID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("tenant: member not found")
	}
	return rowToMembership(rows[0]), nil
}

func (s *Neo4jStore) ListMembers(ctx context.Context, tenantID string) ([]Membership, error) {
	rows, err := s.db.RunRead(ctx, `
		MATCH (u:User)-[r:MEMBER_OF]->(t:Tenant {id: $tenant_id})
		RETURN u.id AS user_id, $tenant_id AS tenant_id, r.role AS role,
		       coalesce(r.email, u.email, '') AS email, r.created_at AS created_at
		ORDER BY r.created_at
	`, map[string]interface{}{"tenant_id": tenantID})
	if err != nil {
		return nil, err
	}
	out := make([]Membership, 0, len(rows))
	for _, row := range rows {
		if m := rowToMembership(row); m != nil {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (s *Neo4jStore) SaveInvite(ctx context.Context, inv *Invite) error {
	_, err := s.db.RunWrite(ctx, `
		MATCH (t:Tenant {id: $tenant_id})
		CREATE (i:TenantInvite {
			id: $id,
			tenant_id: $tenant_id,
			email: $email,
			role: $role,
			token: $token,
			invited_by: $invited_by,
			expires_at: datetime($expires_at),
			created_at: datetime($created_at)
		})
		CREATE (i)-[:FOR_TENANT]->(t)
	`, map[string]interface{}{
		"id":         inv.ID,
		"tenant_id":  inv.TenantID,
		"email":      inv.Email,
		"role":       string(inv.Role),
		"token":      inv.Token,
		"invited_by": inv.InvitedBy,
		"expires_at": inv.ExpiresAt.UTC().Format(time.RFC3339),
		"created_at": inv.CreatedAt.UTC().Format(time.RFC3339),
	})
	return err
}

func (s *Neo4jStore) GetInviteByToken(ctx context.Context, token string) (*Invite, error) {
	rows, err := s.db.RunRead(ctx, `
		MATCH (i:TenantInvite {token: $token})
		RETURN i.id AS id, i.tenant_id AS tenant_id, i.email AS email, i.role AS role,
		       i.token AS token, i.invited_by AS invited_by,
		       i.expires_at AS expires_at, i.created_at AS created_at
	`, map[string]interface{}{"token": token})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("tenant: invite not found")
	}
	return rowToInvite(rows[0]), nil
}

func (s *Neo4jStore) DeleteInvite(ctx context.Context, id string) error {
	_, err := s.db.RunWrite(ctx, `
		MATCH (i:TenantInvite {id: $id})
		DETACH DELETE i
	`, map[string]interface{}{"id": id})
	return err
}

func (s *Neo4jStore) ListInvites(ctx context.Context, tenantID string) ([]Invite, error) {
	rows, err := s.db.RunRead(ctx, `
		MATCH (i:TenantInvite {tenant_id: $tenant_id})
		RETURN i.id AS id, i.tenant_id AS tenant_id, i.email AS email, i.role AS role,
		       i.token AS token, i.invited_by AS invited_by,
		       i.expires_at AS expires_at, i.created_at AS created_at
		ORDER BY i.created_at DESC
	`, map[string]interface{}{"tenant_id": tenantID})
	if err != nil {
		return nil, err
	}
	out := make([]Invite, 0, len(rows))
	for _, row := range rows {
		if inv := rowToInvite(row); inv != nil {
			out = append(out, *inv)
		}
	}
	return out, nil
}

// --- helpers ---

func rowToTenant(row map[string]interface{}) *Tenant {
	if row == nil {
		return nil
	}
	return &Tenant{
		ID:          asString(row["id"]),
		Slug:        asString(row["slug"]),
		Name:        asString(row["name"]),
		Status:      Status(asString(row["status"])),
		Plan:        asString(row["plan"]),
		CreatedBy:   asString(row["created_by"]),
		MemberCount: asInt(row["member_count"]),
		CreatedAt:   asTime(row["created_at"]),
		UpdatedAt:   asTime(row["updated_at"]),
	}
}

func rowToMembership(row map[string]interface{}) *Membership {
	if row == nil {
		return nil
	}
	return &Membership{
		TenantID:  asString(row["tenant_id"]),
		UserID:    asString(row["user_id"]),
		Email:     asString(row["email"]),
		Role:      Role(asString(row["role"])),
		CreatedAt: asTime(row["created_at"]),
	}
}

func rowToInvite(row map[string]interface{}) *Invite {
	if row == nil {
		return nil
	}
	return &Invite{
		ID:        asString(row["id"]),
		TenantID:  asString(row["tenant_id"]),
		Email:     asString(row["email"]),
		Role:      Role(asString(row["role"])),
		Token:     asString(row["token"]),
		InvitedBy: asString(row["invited_by"]),
		ExpiresAt: asTime(row["expires_at"]),
		CreatedAt: asTime(row["created_at"]),
	}
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", v)
	}
}

func asInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}

func asTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed
		}
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return parsed
		}
	}
	// neo4j driver may return time-like types with a Time() method
	type timer interface{ Time() time.Time }
	if t, ok := v.(timer); ok {
		return t.Time()
	}
	if s, ok := v.(fmt.Stringer); ok {
		if parsed, err := time.Parse(time.RFC3339, s.String()); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
