package sso

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
)

type LDAPProvider struct {
	config       *Config
	conn         *ldap.Conn
	connMu       sync.RWMutex
	sessions     map[string]*Session
	sessionsMu   sync.RWMutex
	attributeMap map[string]string
}

type LDAPUser struct {
	DN        string
	CN        string
	Email     string
	Name      string
	FirstName string
	LastName  string
	Groups    []string
	MemberOf  []string
}

func NewLDAPProvider(cfg *Config) (*LDAPProvider, error) {
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("LDAP server URL is required")
	}

	provider := &LDAPProvider{
		config:   cfg,
		sessions: make(map[string]*Session),
		attributeMap: map[string]string{
			"email":     "mail",
			"name":      "cn",
			"firstName": "givenName",
			"lastName":  "sn",
			"groups":    "memberOf",
			"dn":        "dn",
		},
	}

	return provider, nil
}

func (p *LDAPProvider) Name() string {
	return "LDAP"
}

func (p *LDAPProvider) Type() ProviderType {
	return ProviderTypeLDAP
}

func (p *LDAPProvider) getConnection() (*ldap.Conn, error) {
	p.connMu.RLock()
	if p.conn != nil && !p.conn.IsClosing() {
		conn := p.conn
		p.connMu.RUnlock()
		return conn, nil
	}
	p.connMu.RUnlock()

	p.connMu.Lock()
	defer p.connMu.Unlock()

	if p.conn != nil && !p.conn.IsClosing() {
		return p.conn, nil
	}

	server := p.config.IssuerURL
	if !strings.HasPrefix(server, "ldap://") && !strings.HasPrefix(server, "ldaps://") {
		if strings.HasPrefix(server, ":") || strings.HasPrefix(server, "localhost") {
			server = "ldap://" + server
		} else {
			server = "ldaps://" + server
		}
	}

	var conn *ldap.Conn
	var err error

	if strings.HasPrefix(server, "ldaps://") {
		conn, err = ldap.DialURL(server, ldap.DialWithTLSConfig(&tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}))
	} else {
		conn, err = ldap.DialURL(server)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	p.conn = conn
	return conn, nil
}

func (p *LDAPProvider) closeConnection() {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}
}

func (p *LDAPProvider) Authenticate(ctx context.Context, usernameOrEmail string) (*User, error) {
	if usernameOrEmail == "" {
		return nil, fmt.Errorf("username or email is required")
	}

	conn, err := p.getConnection()
	if err != nil {
		return nil, fmt.Errorf("LDAP connection: %w", err)
	}

	var user *LDAPUser
	var password string

	if strings.Contains(usernameOrEmail, ":") {
		parts := strings.SplitN(usernameOrEmail, ":", 2)
		usernameOrEmail = parts[0]
		password = parts[1]
	}

	if password == "" {
		return nil, fmt.Errorf("LDAP authentication requires password in format 'username:password'")
	}

	user, err = p.searchUser(conn, usernameOrEmail)
	if err != nil {
		return nil, fmt.Errorf("user search: %w", err)
	}

	if err := conn.Bind(user.DN, password); err != nil {
		return nil, fmt.Errorf("LDAP bind failed: %w", err)
	}

	if p.config.TenantID != "" {
		if err := p.addToTenant(conn, user); err != nil {
			return nil, fmt.Errorf("failed to add user to tenant: %w", err)
		}
	}

	session := &Session{
		ID:        generateSessionID(),
		UserID:    user.Email,
		TenantID:  p.config.TenantID,
		Token:     generateSessionToken(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	p.sessionsMu.Lock()
	p.sessions[session.Token] = session
	p.sessionsMu.Unlock()

	return p.ldapUserToUser(user), nil
}

func (p *LDAPProvider) searchUser(conn *ldap.Conn, usernameOrEmail string) (*LDAPUser, error) {
	searchFilter := fmt.Sprintf("(|(uid=%s)(sAMAccountName=%s)(mail=%s)(cn=%s))",
		ldap.EscapeFilter(usernameOrEmail),
		ldap.EscapeFilter(usernameOrEmail),
		ldap.EscapeFilter(usernameOrEmail),
		ldap.EscapeFilter(usernameOrEmail))

	baseDN := p.config.CallbackURL
	if baseDN == "" {
		baseDN = "dc=example,dc=com"
	}

	attributes := []string{"dn", "cn", "sn", "givenName", "mail", "uid", "sAMAccountName", "memberOf", "displayName"}

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		10,
		false,
		searchFilter,
		attributes,
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search: %w", err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found: %s", usernameOrEmail)
	}

	if len(result.Entries) > 1 {
		return nil, fmt.Errorf("multiple users found for: %s", usernameOrEmail)
	}

	entry := result.Entries[0]

	user := &LDAPUser{
		DN:        entry.DN,
		CN:        entry.GetAttributeValue("cn"),
		Name:      entry.GetAttributeValue("displayName"),
		Email:     entry.GetAttributeValue("mail"),
		FirstName: entry.GetAttributeValue("givenName"),
		LastName:  entry.GetAttributeValue("sn"),
		MemberOf:  entry.GetAttributeValues("memberOf"),
	}

	if user.Email == "" {
		user.Email = entry.GetAttributeValue("uid")
		if user.Email == "" {
			user.Email = entry.GetAttributeValue("sAMAccountName")
		}
	}

	if user.Name == "" {
		user.Name = user.CN
	}

	return user, nil
}

func (p *LDAPProvider) addToTenant(conn *ldap.Conn, user *LDAPUser) error {
	baseDN := fmt.Sprintf("ou=users,%s", p.config.TenantID)

	userDN := fmt.Sprintf("uid=%s,%s", ldap.EscapeFilter(user.Email), baseDN)

	attributes := []*ldap.EntryAttribute{
		{Name: "objectClass", Values: []string{"inetOrgPerson", "top"}},
		{Name: "uid", Values: []string{user.Email}},
		{Name: "cn", Values: []string{user.Name}},
		{Name: "sn", Values: []string{user.LastName}},
		{Name: "givenName", Values: []string{user.FirstName}},
		{Name: "mail", Values: []string{user.Email}},
	}

	if len(user.MemberOf) > 0 {
		attributes = append(attributes, &ldap.EntryAttribute{Name: "memberOf", Values: user.MemberOf})
	}

	addRequest := ldap.NewAddRequest(userDN, nil)
	for _, attr := range attributes {
		addRequest.Attribute(attr.Name, attr.Values)
	}

	if err := conn.Add(addRequest); err != nil {
		if !ldap.IsErrorWithCode(err, ldap.LDAPResultEntryAlreadyExists) {
			return fmt.Errorf("failed to add user to tenant: %w", err)
		}
	}

	return nil
}

func (p *LDAPProvider) ldapUserToUser(u *LDAPUser) *User {
	return &User{
		ID:       u.Email,
		Email:    u.Email,
		Name:     u.Name,
		TenantID: p.config.TenantID,
		Groups:   u.MemberOf,
		Roles:    extractRolesFromGroups(u.MemberOf),
	}
}

func extractRolesFromGroups(groups []string) []string {
	roles := make([]string, 0)
	roleMappings := map[string]string{
		"admins":         "admin",
		"administrators": "admin",
		"users":          "user",
		"editors":        "editor",
		"developers":     "developer",
		"managers":       "manager",
	}

	for _, group := range groups {
		groupLower := strings.ToLower(group)
		for pattern, role := range roleMappings {
			if strings.Contains(groupLower, pattern) {
				roles = append(roles, role)
				break
			}
		}
	}

	if len(roles) == 0 {
		roles = append(roles, "user")
	}

	return uniqueStrings(roles)
}

func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func (p *LDAPProvider) GetLogoutURL(redirectURL string) (string, error) {
	if redirectURL != "" {
		return redirectURL, nil
	}
	return p.config.IssuerURL, nil
}

func (p *LDAPProvider) ValidateSession(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, fmt.Errorf("session token is required")
	}

	p.sessionsMu.RLock()
	session, ok := p.sessions[token]
	p.sessionsMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	if session.ExpiresAt != "" {
		expiresAt, _ := time.Parse(time.RFC3339, session.ExpiresAt)
		if time.Now().After(expiresAt) {
			p.sessionsMu.Lock()
			delete(p.sessions, token)
			p.sessionsMu.Unlock()
			return nil, fmt.Errorf("session expired")
		}
	}

	return session, nil
}

func (p *LDAPProvider) RefreshSession(ctx context.Context, token string) (*Session, error) {
	session, err := p.ValidateSession(ctx, token)
	if err != nil {
		return nil, err
	}

	if session.ExpiresAt != "" {
		expiresAt, _ := time.Parse(time.RFC3339, session.ExpiresAt)
		if time.Now().After(expiresAt.Add(-5 * time.Minute)) {
			return nil, fmt.Errorf("session cannot be refreshed within 5 minutes of expiry")
		}
	}

	session.Token = generateSessionToken()
	session.ExpiresAt = time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	p.sessionsMu.Lock()
	p.sessions[session.Token] = session
	p.sessionsMu.Unlock()

	return session, nil
}

func (p *LDAPProvider) SearchUsers(ctx context.Context, query string, limit int) ([]*User, error) {
	conn, err := p.getConnection()
	if err != nil {
		return nil, fmt.Errorf("LDAP connection: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}

	searchFilter := fmt.Sprintf("(|(uid=*%s*)(cn=*%s*)(mail=*%s*))",
		ldap.EscapeFilter(query),
		ldap.EscapeFilter(query),
		ldap.EscapeFilter(query))

	baseDN := p.config.CallbackURL
	if baseDN == "" {
		baseDN = "dc=example,dc=com"
	}

	attributes := []string{"dn", "cn", "sn", "givenName", "mail", "uid", "memberOf"}

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		ldap.NeverDerefAliases,
		0,
		false,
		searchFilter,
		attributes,
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search: %w", err)
	}

	users := make([]*User, 0, len(result.Entries))
	for _, entry := range result.Entries {
		user := &LDAPUser{
			DN:        entry.DN,
			Name:      entry.GetAttributeValue("cn"),
			Email:     entry.GetAttributeValue("mail"),
			FirstName: entry.GetAttributeValue("givenName"),
			LastName:  entry.GetAttributeValue("sn"),
			MemberOf:  entry.GetAttributeValues("memberOf"),
		}

		if user.Email == "" {
			user.Email = entry.GetAttributeValue("uid")
		}

		users = append(users, p.ldapUserToUser(user))
	}

	return users, nil
}

func (p *LDAPProvider) ListGroups(ctx context.Context) ([]string, error) {
	conn, err := p.getConnection()
	if err != nil {
		return nil, fmt.Errorf("LDAP connection: %w", err)
	}

	baseDN := p.config.CallbackURL
	if baseDN == "" {
		baseDN = "dc=example,dc=com"
	}

	searchFilter := "(objectClass=groupOfNames)"

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		ldap.NeverDerefAliases,
		0,
		false,
		searchFilter,
		[]string{"cn"},
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		searchRequest2 := ldap.NewSearchRequest(
			baseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			ldap.NeverDerefAliases,
			0,
			false,
			"(objectClass=*)",
			[]string{"cn"},
			nil,
		)
		result, err = conn.Search(searchRequest2)
		if err != nil {
			return nil, fmt.Errorf("LDAP group search: %w", err)
		}
	}

	groups := make([]string, 0, len(result.Entries))
	seen := make(map[string]bool)
	for _, entry := range result.Entries {
		cn := entry.GetAttributeValue("cn")
		if cn != "" && !seen[cn] {
			seen[cn] = true
			groups = append(groups, cn)
		}
	}

	return groups, nil
}

func (p *LDAPProvider) Close() error {
	p.closeConnection()

	p.sessionsMu.Lock()
	p.sessions = make(map[string]*Session)
	p.sessionsMu.Unlock()

	return nil
}

func generateSessionToken() string {
	b := make([]byte, 32)
	timestamp := time.Now().UnixNano()
	for i := range b {
		b[i] = byte(timestamp >> uint(i%8))
	}
	return fmt.Sprintf("%x", b)
}
