package notification

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"sync"
	"time"

	"agent-memory/internal/config"
)

// notificationStore is the file-based persistence layer for notifications.
type notificationStore struct {
	mu       sync.Mutex
	filePath string
	file     *os.File
}

// openNotificationStore opens (or creates) the notifications journal at path
// and returns a store ready for appending.
func openNotificationStore(path string) (*notificationStore, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("notification store: open: %w", err)
	}
	return &notificationStore{filePath: path, file: f}, nil
}

// append writes one notification as a JSON line.
func (ns *notificationStore) append(n *Notification) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}
	_, err = ns.file.Write(append(data, '\n'))
	return err
}

// loadAll reads every notification from the journal.
func (ns *notificationStore) loadAll() ([]*Notification, error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	f, err := os.Open(ns.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []*Notification
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var n Notification
		if json.Unmarshal(line, &n) == nil {
			cp := n
			out = append(out, &cp)
		}
	}
	return out, sc.Err()
}

// rewrite rewrites the journal atomically with only the provided notifications.
func (ns *notificationStore) rewrite(notifications map[string]*Notification) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	tmp := ns.filePath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("notification store: rewrite open: %w", err)
	}
	for _, n := range notifications {
		data, _ := json.Marshal(n)
		f.Write(append(data, '\n'))
	}
	f.Close()
	if err := os.Rename(tmp, ns.filePath); err != nil {
		return fmt.Errorf("notification store: rewrite rename: %w", err)
	}

	// Reopen the append file after rewrite.
	ns.file.Close()
	ns.file, err = os.OpenFile(ns.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	return err
}

// Close closes the underlying file.
func (ns *notificationStore) Close() error {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	return ns.file.Close()
}

// notificationDirOf returns the directory portion of a file path.
func notificationDirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

type Service struct {
	notifications map[string]*Notification
	preferences   map[string]*NotificationPreferences
	emailConfig   *config.EmailConfig
	webhookURL    string
	mu            sync.RWMutex
	stopCh        chan struct{}
	store         *notificationStore // nil when persistence is disabled
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

// NewPersistentService creates a Service that persists notifications to a
// JSON-lines file at filePath and reloads them on startup.
func NewPersistentService(cfg *config.Config, filePath string) (*Service, error) {
	if dir := notificationDirOf(filePath); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("notification service: mkdir: %w", err)
		}
	}

	// Load existing notifications before opening for append so we don't
	// read our own new writes.
	store, err := openNotificationStore(filePath)
	if err != nil {
		return nil, err
	}

	existing, err := store.loadAll()
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("notification service: load: %w", err)
	}

	notifications := make(map[string]*Notification, len(existing))
	for _, n := range existing {
		notifications[n.ID] = n
	}

	s := &Service{
		notifications: notifications,
		preferences:   make(map[string]*NotificationPreferences),
		emailConfig:   &cfg.Email,
		webhookURL:    cfg.Webhook.URL,
		stopCh:        make(chan struct{}),
		store:         store,
	}
	return s, nil
}

// Close flushes any pending state and closes the persistence file.
func (s *Service) Close() error {
	if s.store == nil {
		return nil
	}
	// Rewrite with current in-memory state so status changes (MarkRead, Archive,
	// Delete) are durably persisted.
	s.mu.RLock()
	notifications := make(map[string]*Notification, len(s.notifications))
	for k, v := range s.notifications {
		cp := *v
		notifications[k] = &cp
	}
	s.mu.RUnlock()
	return s.store.rewrite(notifications)
}

func (s *Service) Create(ctx context.Context, tenantID string, req CreateNotificationRequest) (*Notification, error) {
	notif := NewNotification(req, tenantID)

	s.mu.Lock()
	s.notifications[notif.ID] = notif
	s.mu.Unlock()

	// Persist the new notification immediately if a store is configured.
	if s.store != nil {
		if err := s.store.append(notif); err != nil {
			fmt.Printf("notification store: append error: %v\n", err)
		}
	}

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
	InAppEnabled   *bool                  `json:"in_app_enabled,omitempty"`
	EmailEnabled   *bool                  `json:"email_enabled,omitempty"`
	WebhookEnabled *bool                  `json:"webhook_enabled,omitempty"`
	EmailAddress   *string                `json:"email_address,omitempty"`
	WebhookURL     *string                `json:"webhook_url,omitempty"`
	MuteTypes      []NotificationType     `json:"mute_types,omitempty"`
	MuteChannels   []NotificationChannel  `json:"mute_channels,omitempty"`
}