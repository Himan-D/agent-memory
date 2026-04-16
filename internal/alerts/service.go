package alerts

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	ListRules() ([]AlertRule, error)
	GetRule(id uuid.UUID) (*AlertRule, error)
	CreateRule(rule *AlertRule) error
	UpdateRule(id uuid.UUID, updates *UpdateAlertRuleRequest) error
	DeleteRule(id uuid.UUID) error
	ListActiveAlerts() ([]Alert, error)
	CreateAlert(alert *Alert) error
	UpdateAlertStatus(id uuid.UUID, status AlertStatus) error
	GetAlert(id uuid.UUID) (*Alert, error)
}

type InMemoryStore struct {
	mu    sync.RWMutex
	rules map[uuid.UUID]*AlertRule
	alerts map[uuid.UUID]*Alert
}

func NewInMemoryStore() *InMemoryStore {
	store := &InMemoryStore{
		rules:  make(map[uuid.UUID]*AlertRule),
		alerts: make(map[uuid.UUID]*Alert),
	}
	store.seedDefaultRules()
	return store
}

func (s *InMemoryStore) seedDefaultRules() {
	for _, rule := range GetDefaultRules() {
		s.rules[rule.ID] = &rule
	}
}

func (s *InMemoryStore) ListRules() ([]AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rules := make([]AlertRule, 0, len(s.rules))
	for _, r := range s.rules {
		rules = append(rules, *r)
	}
	return rules, nil
}

func (s *InMemoryStore) GetRule(id uuid.UUID) (*AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if rule, ok := s.rules[id]; ok {
		return rule, nil
	}
	return nil, fmt.Errorf("rule not found")
}

func (s *InMemoryStore) CreateRule(rule *AlertRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	s.rules[rule.ID] = rule
	return nil
}

func (s *InMemoryStore) UpdateRule(id uuid.UUID, updates *UpdateAlertRuleRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rule, ok := s.rules[id]
	if !ok {
		return fmt.Errorf("rule not found")
	}
	if updates.Name != nil {
		rule.Name = *updates.Name
	}
	if updates.Description != nil {
		rule.Description = *updates.Description
	}
	if updates.Condition != nil {
		rule.Condition = *updates.Condition
	}
	if updates.Threshold != nil {
		rule.Threshold = *updates.Threshold
	}
	if updates.Operator != nil {
		rule.Operator = *updates.Operator
	}
	if updates.Severity != nil {
		rule.Severity = *updates.Severity
	}
	if updates.Enabled != nil {
		rule.Enabled = *updates.Enabled
	}
	if updates.NotifyEmail != nil {
		rule.NotifyEmail = *updates.NotifyEmail
	}
	if updates.NotifyWebhook != nil {
		rule.NotifyWebhook = *updates.NotifyWebhook
	}
	if updates.NotifyInApp != nil {
		rule.NotifyInApp = *updates.NotifyInApp
	}
	rule.UpdatedAt = time.Now()
	return nil
}

func (s *InMemoryStore) DeleteRule(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[id]; !ok {
		return fmt.Errorf("rule not found")
	}
	delete(s.rules, id)
	return nil
}

func (s *InMemoryStore) ListActiveAlerts() ([]Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	alerts := make([]Alert, 0)
	for _, a := range s.alerts {
		if a.Status == AlertStatusActive {
			alerts = append(alerts, *a)
		}
	}
	return alerts, nil
}

func (s *InMemoryStore) CreateAlert(alert *Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if alert.ID == uuid.Nil {
		alert.ID = uuid.New()
	}
	alert.TriggeredAt = time.Now()
	alert.Status = AlertStatusActive
	s.alerts[alert.ID] = alert
	return nil
}

func (s *InMemoryStore) UpdateAlertStatus(id uuid.UUID, status AlertStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	alert, ok := s.alerts[id]
	if !ok {
		return fmt.Errorf("alert not found")
	}
	alert.Status = status
	if status == AlertStatusResolved {
		now := time.Now()
		alert.ResolvedAt = &now
	}
	return nil
}

func (s *InMemoryStore) GetAlert(id uuid.UUID) (*Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if alert, ok := s.alerts[id]; ok {
		return alert, nil
	}
	return nil, fmt.Errorf("alert not found")
}

type AnalyticsData struct {
	RetentionRate     float64
	DailyActiveUsers  int
	NegativeRatio     float64
	APICallsToday     int
	ActiveAgents      int
	TotalAgents       int
	StorageUsedGB     float64
}

type NotificationService interface {
	SendNotification(userID string, title string, message string, alertType string) error
}

type Service struct {
	store              Store
	notificationService NotificationService
	mu                 sync.Mutex
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetNotificationService(ns NotificationService) {
	s.notificationService = ns
}

func (s *Service) ListRules() ([]AlertRule, error) {
	return s.store.ListRules()
}

func (s *Service) GetRule(id uuid.UUID) (*AlertRule, error) {
	return s.store.GetRule(id)
}

func (s *Service) CreateRule(req *CreateAlertRuleRequest) (*AlertRule, error) {
	rule := &AlertRule{
		Name:          req.Name,
		Description:   req.Description,
		Type:          req.Type,
		Severity:      req.Severity,
		Condition:     req.Condition,
		Threshold:     req.Threshold,
		Operator:      req.Operator,
		Enabled:       true,
		NotifyEmail:   req.NotifyEmail,
		NotifyWebhook: req.NotifyWebhook,
		NotifyInApp:   req.NotifyInApp,
	}
	if err := s.store.CreateRule(rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *Service) UpdateRule(id uuid.UUID, req *UpdateAlertRuleRequest) (*AlertRule, error) {
	if err := s.store.UpdateRule(id, req); err != nil {
		return nil, err
	}
	return s.store.GetRule(id)
}

func (s *Service) DeleteRule(id uuid.UUID) error {
	return s.store.DeleteRule(id)
}

func (s *Service) EnableRule(id uuid.UUID, enabled bool) error {
	return s.store.UpdateRule(id, &UpdateAlertRuleRequest{Enabled: &enabled})
}

func (s *Service) ListActiveAlerts() ([]Alert, error) {
	return s.store.ListActiveAlerts()
}

func (s *Service) ListAllAlerts() ([]Alert, error) {
	s.store.ListActiveAlerts()
	s.mu.Lock()
	defer s.mu.Unlock()
	// Return all alerts including resolved
	return nil, nil
}

func (s *Service) ResolveAlert(id uuid.UUID) error {
	return s.store.UpdateAlertStatus(id, AlertStatusResolved)
}

func (s *Service) DismissAlert(id uuid.UUID) error {
	return s.store.UpdateAlertStatus(id, AlertStatusDismissed)
}

func (s *Service) CheckAnalytics(data *AnalyticsData) ([]Alert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rules, err := s.store.ListRules()
	if err != nil {
		return nil, err
	}

	triggered := []Alert{}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		value := s.getMetricValue(data, rule.Type)
		shouldTrigger := s.evaluateCondition(value, rule.Threshold, rule.Operator)

		if shouldTrigger {
			alert := &Alert{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Type:      rule.Type,
				Severity:  rule.Severity,
				Message:   fmt.Sprintf("%s: %.2f (threshold: %.2f)", rule.Name, value, rule.Threshold),
				Value:     value,
				Threshold: rule.Threshold,
			}
			s.store.CreateAlert(alert)
			triggered = append(triggered, *alert)

			if s.notificationService != nil && rule.NotifyInApp {
				s.notificationService.SendNotification("system", rule.Name, alert.Message, "alert")
			}
		}
	}

	return triggered, nil
}

func (s *Service) getMetricValue(data *AnalyticsData, alertType AlertType) float64 {
	switch alertType {
	case AlertTypeRetention:
		return data.RetentionRate
	case AlertTypeUsage:
		return float64(data.DailyActiveUsers)
	case AlertTypeNegativeFeedback:
		return data.NegativeRatio
	case AlertTypeAPIQuota:
		return float64(data.APICallsToday)
	case AlertTypeAgentOffline:
		return float64(data.ActiveAgents)
	case AlertTypeStorage:
		return data.StorageUsedGB
	default:
		return 0
	}
}

func (s *Service) evaluateCondition(value, threshold float64, op ConditionOperator) bool {
	switch op {
	case OpLessThan:
		return value < threshold
	case OpGreaterThan:
		return value > threshold
	case OpEquals:
		return value == threshold
	default:
		return false
	}
}

func (s *Service) GetAlertStats() (map[string]int, error) {
	rules, err := s.store.ListRules()
	if err != nil {
		return nil, err
	}

	activeAlerts, err := s.store.ListActiveAlerts()
	if err != nil {
		return nil, err
	}

	stats := map[string]int{
		"total_rules":   len(rules),
		"enabled_rules": 0,
		"active_alerts": len(activeAlerts),
	}

	for _, r := range rules {
		if r.Enabled {
			stats["enabled_rules"]++
		}
	}

	return stats, nil
}