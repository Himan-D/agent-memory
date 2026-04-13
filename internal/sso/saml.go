package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

type SAMLProvider struct {
	config       *Config
	certificate  *x509.Certificate
	publicKey    *rsa.PublicKey
	sessions     map[string]*Session
	sessionsMu   sync.RWMutex
	attributeMap map[string]string
}

func NewSAMLProvider(cfg *Config) (*SAMLProvider, error) {
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("SAML issuer URL is required")
	}
	if cfg.CallbackURL == "" {
		return nil, fmt.Errorf("SAML callback URL is required")
	}

	provider := &SAMLProvider{
		config:   cfg,
		sessions: make(map[string]*Session),
		attributeMap: map[string]string{
			"email":     "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"name":      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			"firstname": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
			"lastname":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
			"groups":    "http://schemas.microsoft.com/ws/2008/06/identity/claims/groups",
			"role":      "http://schemas.microsoft.com/ws/2008/06/identity/claims/role",
		},
	}

	return provider, nil
}

func (p *SAMLProvider) Name() string {
	return "SAML"
}

func (p *SAMLProvider) Type() ProviderType {
	return ProviderTypeSAML
}

func (p *SAMLProvider) Authenticate(ctx context.Context, samlResponse string) (*User, error) {
	if samlResponse == "" {
		return nil, fmt.Errorf("SAML response is required")
	}

	decoded, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SAML response: %w", err)
	}

	var response SAMLResponse
	if err := xml.Unmarshal(decoded, &response); err != nil {
		return nil, fmt.Errorf("failed to parse SAML response: %w", err)
	}

	if response.Status.StatusCode.Value != SuccessStatus {
		return nil, fmt.Errorf("SAML authentication failed: %s", response.Status.StatusMessage)
	}

	user := p.extractUser(&response.Assertion)

	session := &Session{
		ID:        generateSamlSessionID(),
		UserID:    user.ID,
		TenantID:  user.TenantID,
		Token:     generateSamlSessionToken(),
		ExpiresAt: response.Assertion.SessionNotOnOrAfter,
	}

	p.sessionsMu.Lock()
	p.sessions[session.Token] = session
	p.sessionsMu.Unlock()

	return user, nil
}

func (p *SAMLProvider) GetLogoutURL(redirectURL string) (string, error) {
	logoutURL := p.config.IssuerURL + "/saml/logout"
	return fmt.Sprintf("%s?redirect=%s", logoutURL, url.QueryEscape(redirectURL)), nil
}

func (p *SAMLProvider) ValidateSession(ctx context.Context, token string) (*Session, error) {
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

func (p *SAMLProvider) RefreshSession(ctx context.Context, token string) (*Session, error) {
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

	session.Token = generateSamlSessionToken()
	session.ExpiresAt = time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	p.sessionsMu.Lock()
	p.sessions[session.Token] = session
	p.sessionsMu.Unlock()

	return session, nil
}

func (p *SAMLProvider) InitiateLogin(callbackURL string) (string, string, error) {
	authnRequest := p.createAuthnRequest(callbackURL)
	encoded := base64.StdEncoding.EncodeToString([]byte(authnRequest))

	ssoURL := p.config.IssuerURL + "/saml/sso"
	requestID := extractSamlRequestID(authnRequest)

	return ssoURL + "?SAMLRequest=" + url.QueryEscape(encoded), requestID, nil
}

func (p *SAMLProvider) extractUser(assertion *Assertion) *User {
	user := &User{
		ID:       assertion.Subject.NameID.Value,
		Email:    assertion.Subject.NameID.Value,
		TenantID: p.config.TenantID,
		Groups:   []string{},
		Roles:    []string{},
	}

	for _, attr := range assertion.Attributes {
		attrName := strings.ToLower(attr.Name)
		switch attrName {
		case "email", "emailaddress":
			user.Email = attr.Value
		case "name", "displayname":
			user.Name = attr.Value
		case "givenname", "firstname":
			if user.Name == "" {
				user.Name = attr.Value
			} else {
				user.Name = attr.Value + " " + user.Name
			}
		case "surname", "lastname":
			parts := strings.Split(user.Name, " ")
			if len(parts) > 1 {
				user.Name = strings.Join(append([]string{attr.Value}, parts[1:]...), " ")
			} else {
				user.Name = attr.Value
			}
		case "groups":
			user.Groups = append(user.Groups, attr.Value)
		case "role", "roles":
			user.Roles = append(user.Roles, attr.Value)
		}
	}

	if user.Name == "" {
		user.Name = user.Email
	}

	return user
}

func (p *SAMLProvider) createAuthnRequest(callbackURL string) string {
	id := fmt.Sprintf("_%s", generateSamlRequestID())
	issueInstant := time.Now().UTC().Format(time.RFC3339)

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="%s" Version="2.0" IssueInstant="%s"
    AssertionConsumerServiceURL="%s"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST">
    <saml:Issuer>%s</saml:Issuer>
    <samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress" AllowCreate="true"/>
</samlp:AuthnRequest>`, id, issueInstant, callbackURL, p.config.IssuerURL)
}

func parseSamlCertificate(certPEM string) (*x509.Certificate, error) {
	certPEM = strings.TrimSpace(certPEM)

	var certBlock *pem.Block
	for {
		block, rest := pem.Decode([]byte(certPEM))
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			certBlock = block
			break
		}
		certPEM = string(rest)
	}

	if certBlock == nil {
		return nil, fmt.Errorf("no certificate found in PEM data")
	}

	return x509.ParseCertificate(certBlock.Bytes)
}

func generateSamlSessionID() string {
	return fmt.Sprintf("sess_%s", generateSamlRequestID())
}

func generateSamlSessionToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		timestamp := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(timestamp >> uint(i%8))
		}
	}
	return base64.URLEncoding.EncodeToString(b)
}

func generateSamlRequestID() string {
	timestamp := time.Now().UnixNano()
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(timestamp >> uint(i*8))
	}
	return base64.URLEncoding.EncodeToString(b)[:16]
}

func extractSamlRequestID(authnRequest string) string {
	start := strings.Index(authnRequest, `ID="`)
	if start == -1 {
		start = strings.Index(authnRequest, `ID='`)
		if start == -1 {
			return ""
		}
		start += 4
		end := strings.Index(authnRequest[start+4:], `"`)
		if end == -1 {
			return ""
		}
		return authnRequest[start : start+4+end]
	}
	start += 4
	end := strings.Index(authnRequest[start:], `"`)
	if end == -1 {
		return ""
	}
	return authnRequest[start : start+end]
}

const SuccessStatus = "urn:oasis:names:tc:SAML:2.0:status:Success"

type SAMLResponse struct {
	XMLName   xml.Name  `xml:"urn:oasis:names:tc:SAML:2.0:protocol Response"`
	Status    Status    `xml:"Status"`
	Assertion Assertion `xml:"Assertion"`
}

type Status struct {
	StatusCode    StatusCode `xml:"StatusCode"`
	StatusMessage string     `xml:"StatusMessage,omitempty"`
}

type StatusCode struct {
	Value string `xml:"Value,attr"`
}

type Assertion struct {
	XMLName             xml.Name       `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
	ID                  string         `xml:"ID,attr"`
	IssueInstant        string         `xml:"IssueInstant,attr"`
	Version             string         `xml:"Version,attr"`
	Issuer              string         `xml:"Issuer"`
	Subject             Subject        `xml:"Subject"`
	Conditions          Conditions     `xml:"Conditions"`
	AuthnStatement      AuthnStatement `xml:"AuthnStatement"`
	SessionNotOnOrAfter string         `xml:"SessionNotOnOrAfter,attr"`
	Attributes          []Attribute    `xml:"AttributeStatement>Attribute"`
}

type Subject struct {
	NameID              NameID              `xml:"NameID"`
	SubjectConfirmation SubjectConfirmation `xml:"SubjectConfirmation"`
}

type NameID struct {
	Value string `xml:",chardata"`
}

type SubjectConfirmation struct {
	Method                  string                  `xml:"Method,attr"`
	SubjectConfirmationData SubjectConfirmationData `xml:"SubjectConfirmationData"`
}

type SubjectConfirmationData struct {
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
	Recipient    string `xml:"Recipient,attr"`
}

type Conditions struct {
	NotBefore    string `xml:"NotBefore,attr"`
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
}

type AuthnStatement struct {
	AuthnInstant string `xml:"AuthnInstant,attr"`
}

type Attribute struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:"AttributeValue"`
}
