package notification

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypeInfo     NotificationType = "info"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
)

type NotificationChannel string

const (
	ChannelInApp  NotificationChannel = "in_app"
	ChannelEmail  NotificationChannel = "email"
	ChannelWebhook NotificationChannel = "webhook"
)

type NotificationStatus string

const (
	NotificationStatusUnread NotificationStatus = "unread"
	NotificationStatusRead   NotificationStatus = "read"
	NotificationStatusArchived NotificationStatus = "archived"
)

type Notification struct {
	ID          string              `json:"id"`
	TenantID    string              `json:"tenant_id"`
	UserID      string              `json:"user_id"`
	Type        NotificationType    `json:"type"`
	Title       string              `json:"title"`
	Message     string              `json:"message"`
	Channel     NotificationChannel `json:"channel"`
	Status      NotificationStatus  `json:"status"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Link        string              `json:"link,omitempty"`
	ReadAt      *time.Time         `json:"read_at,omitempty"`
	ExpiresAt   *time.Time         `json:"expires_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type CreateNotificationRequest struct {
	UserID    string                 `json:"user_id"`
	Type      NotificationType       `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Channel   NotificationChannel    `json:"channel"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Link      string                 `json:"link,omitempty"`
	ExpiresIn *time.Duration        `json:"expires_in,omitempty"`
}

type NotificationPreferences struct {
	ID                 string   `json:"id"`
	TenantID           string   `json:"tenant_id"`
	UserID             string   `json:"user_id"`
	InAppEnabled       bool     `json:"in_app_enabled"`
	EmailEnabled       bool     `json:"email_enabled"`
	WebhookEnabled     bool     `json:"webhook_enabled"`
	EmailAddress       string   `json:"email_address,omitempty"`
	WebhookURL         string   `json:"webhook_url,omitempty"`
	MuteTypes          []NotificationType `json:"mute_types,omitempty"`
	MuteChannels       []NotificationChannel `json:"mute_channels,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type EmailNotification struct {
	To          string                 `json:"to"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	HTMLBody    string                 `json:"html_body,omitempty"`
	TemplateID  string                 `json:"template_id,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
}

type WebhookNotification struct {
	URL         string                 `json:"url"`
	Event       string                 `json:"event"`
	Payload     map[string]interface{} `json:"payload"`
	RetryCount  int                    `json:"retry_count,omitempty"`
}

type NotificationSummary struct {
	Total      int64 `json:"total"`
	Unread     int64 `json:"unread"`
	Read       int64 `json:"read"`
	Archived   int64 `json:"archived"`
	ByType     map[NotificationType]int64 `json:"by_type,omitempty"`
}

func NewNotification(req CreateNotificationRequest, tenantID string) *Notification {
	now := time.Now()
	notif := &Notification{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		UserID:    req.UserID,
		Type:      req.Type,
		Title:     req.Title,
		Message:   req.Message,
		Channel:   req.Channel,
		Status:    NotificationStatusUnread,
		Data:      req.Data,
		Link:      req.Link,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if req.ExpiresIn != nil {
		expires := now.Add(*req.ExpiresIn)
		notif.ExpiresAt = &expires
	}

	return notif
}

func (n *Notification) MarkRead() {
	now := time.Now()
	n.Status = NotificationStatusRead
	n.ReadAt = &now
	n.UpdatedAt = now
}

func (n *Notification) Archive() {
	n.Status = NotificationStatusArchived
	n.UpdatedAt = time.Now()
}

func (n *Notification) IsExpired() bool {
	if n.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*n.ExpiresAt)
}