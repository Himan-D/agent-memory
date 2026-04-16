package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"sync"
	"time"

	"agent-memory/internal/config"
)

type Service struct {
	notifications map[string]*Notification
	preferences  map[string]*NotificationPreferences
	emailConfig  *config.EmailConfig
	webhookURL   string
	mu           sync.RWMutex
	stopCh       chan struct{}
}

func NewService(cfg *config.Config) *Service {
	s := &Service{
		notifications: make(map[string]*Notification),
		preferences:   make(map[string]*NotificationPreferences),
		emailConfig:   &cfg.Email,
		webhookURL:    cfg.Webhook.URL,
		stopCh:        make(chan struct{}),
	}
	return s
}

func (s *Service) Create(ctx context.Context, tenantID string, req CreateNotificationRequest) (*Notification, error) {
	notif := NewNotification(req, tenantID)

	s.mu.Lock()
	s.notifications[notif.ID] = notif
	s.mu.Unlock()

	s.deliverAsync(notif)

	return notif, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notif, ok := s.notifications[id]
	if !ok {
		return nil, fmt.Errorf("notification not found: %s", id)
	}

	if notif.IsExpired() {
		return nil, fmt.Errorf("notification expired: %s", id)
	}

	return notif, nil
}

func (s *Service) List(ctx context.Context, userID string, req ListNotificationsRequest) ([]*Notification, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []*Notification{}
	var total int64

	for _, n := range s.notifications {
		if n.UserID != userID {
			continue
		}
		if n.IsExpired() {
			continue
		}

		switch req.Status {
		case "unread":
			if n.Status != NotificationStatusUnread {
				continue
			}
		case "read":
			if n.Status != NotificationStatusRead {
				continue
			}
		case "archived":
			if n.Status != NotificationStatusArchived {
				continue
			}
		}

		if req.Type != "" && n.Type != req.Type {
			continue
		}

		if req.Channel != "" && n.Channel != req.Channel {
			continue
		}

		total++
		if int64(len(result)) < req.Limit {
			result = append(result, n)
		}
	}

	return result, total, nil
}

func (s *Service) MarkRead(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	notif, ok := s.notifications[id]
	if !ok {
		return fmt.Errorf("notification not found: %s", id)
	}

	notif.MarkRead()
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, n := range s.notifications {
		if n.UserID == userID && n.Status == NotificationStatusUnread {
			n.Status = NotificationStatusRead
			n.ReadAt = &now
			n.UpdatedAt = now
		}
	}

	return nil
}

func (s *Service) Archive(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	notif, ok := s.notifications[id]
	if !ok {
		return fmt.Errorf("notification not found: %s", id)
	}

	notif.Archive()
	return nil
}

func (s *Service) ArchiveAll(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, n := range s.notifications {
		if n.UserID == userID {
			n.Archive()
		}
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.notifications[id]; !ok {
		return fmt.Errorf("notification not found: %s", id)
	}

	delete(s.notifications, id)
	return nil
}

func (s *Service) GetSummary(ctx context.Context, userID string) (*NotificationSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &NotificationSummary{
		ByType: make(map[NotificationType]int64),
	}

	for _, n := range s.notifications {
		if n.UserID != userID {
			continue
		}
		if n.IsExpired() {
			continue
		}

		summary.Total++
		switch n.Status {
		case NotificationStatusUnread:
			summary.Unread++
		case NotificationStatusRead:
			summary.Read++
		case NotificationStatusArchived:
			summary.Archived++
		}

		summary.ByType[n.Type]++
	}

	return summary, nil
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (*NotificationPreferences, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if pref, ok := s.preferences[userID]; ok {
		return pref, nil
	}

	return nil, fmt.Errorf("preferences not found for user: %s", userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (*NotificationPreferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pref, ok := s.preferences[userID]
	if !ok {
		pref = &NotificationPreferences{
			ID:        userID,
			UserID:    userID,
			CreatedAt: time.Now(),
		}
		s.preferences[userID] = pref
	}

	if req.InAppEnabled != nil {
		pref.InAppEnabled = *req.InAppEnabled
	}
	if req.EmailEnabled != nil {
		pref.EmailEnabled = *req.EmailEnabled
	}
	if req.WebhookEnabled != nil {
		pref.WebhookEnabled = *req.WebhookEnabled
	}
	if req.EmailAddress != nil {
		pref.EmailAddress = *req.EmailAddress
	}
	if req.WebhookURL != nil {
		pref.WebhookURL = *req.WebhookURL
	}
	if req.MuteTypes != nil {
		pref.MuteTypes = req.MuteTypes
	}
	if req.MuteChannels != nil {
		pref.MuteChannels = req.MuteChannels
	}

	pref.UpdatedAt = time.Now()

	return pref, nil
}

func (s *Service) deliverAsync(n *Notification) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.deliver(ctx, n); err != nil {
			fmt.Printf("notification delivery error for %s: %v\n", n.ID, err)
		}
	}()
}

func (s *Service) deliver(ctx context.Context, n *Notification) error {
	pref, err := s.GetPreferences(ctx, n.UserID)
	if err != nil {
		pref = nil
	}

	if pref != nil {
		if n.Channel == ChannelEmail && !pref.EmailEnabled {
			return nil
		}
		if n.Channel == ChannelWebhook && !pref.WebhookEnabled {
			return nil
		}
		for _, t := range pref.MuteTypes {
			if n.Type == t {
				return nil
			}
		}
		for _, c := range pref.MuteChannels {
			if n.Channel == c {
				return nil
			}
		}
	}

	switch n.Channel {
	case ChannelInApp:
		return nil
	case ChannelEmail:
		return s.sendEmail(n, pref)
	case ChannelWebhook:
		return s.sendWebhook(n, pref)
	}

	return nil
}

func (s *Service) sendEmail(n *Notification, pref *NotificationPreferences) error {
	if s.emailConfig.SMTPHost == "" {
		return fmt.Errorf("email not configured")
	}

	to := ""
	if pref != nil && pref.EmailAddress != "" {
		to = pref.EmailAddress
	} else if n.Data != nil {
		if email, ok := n.Data["email"].(string); ok {
			to = email
		}
	}

	if to == "" {
		return fmt.Errorf("no email address for notification: %s", n.ID)
	}

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n%s",
		s.emailConfig.FromAddress,
		to,
		n.Title,
		n.Message,
	)

	addr := fmt.Sprintf("%s:%d", s.emailConfig.SMTPHost, s.emailConfig.SMTPPort)

	var auth smtp.Auth
	if s.emailConfig.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.emailConfig.SMTPUsername, s.emailConfig.SMTPPassword, s.emailConfig.SMTPHost)
	}

	return smtp.SendMail(addr, auth, s.emailConfig.FromAddress, []string{to}, []byte(msg))
}

func (s *Service) sendWebhook(n *Notification, pref *NotificationPreferences) error {
	webhookURL := s.webhookURL
	if pref != nil && pref.WebhookURL != "" {
		webhookURL = pref.WebhookURL
	}

	if webhookURL == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	payload := map[string]interface{}{
		"event":     n.ID,
		"type":      n.Type,
		"title":     n.Title,
		"message":   n.Message,
		"data":      n.Data,
		"timestamp": n.CreatedAt,
	}

	return s.deliverWebhook(webhookURL, payload)
}

func (s *Service) deliverWebhook(url string, payload map[string]interface{}) error {
	if url == "" {
		return nil
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

func (s *Service) SendNotification(userID, title, message, notifType string) error {
	ctx := context.Background()
	expiresIn := 7 * 24 * time.Hour
	req := CreateNotificationRequest{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Type:      NotificationType(notifType),
		Channel:   ChannelInApp,
		ExpiresIn: &expiresIn,
	}
	
	tenantID := "system"
	if userID != "system" {
		tenantID = userID
	}
	
	_, err := s.Create(ctx, tenantID, req)
	return err
}

func (s *Service) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, n := range s.notifications {
		if n.ExpiresAt != nil && now.After(*n.ExpiresAt) {
			delete(s.notifications, id)
		}
	}
}

type ListNotificationsRequest struct {
	Status  string
	Type    NotificationType
	Channel NotificationChannel
	Limit   int64
	Offset  int64
}

type UpdatePreferencesRequest struct {
	InAppEnabled   *bool
	EmailEnabled   *bool
	WebhookEnabled *bool
	EmailAddress   *string
	WebhookURL     *string
	MuteTypes      []NotificationType
	MuteChannels   []NotificationChannel
}