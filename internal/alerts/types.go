package alerts

import (
	"time"

	"github.com/google/uuid"
)

type AlertStatus string

const (
	AlertStatusActive    AlertStatus = "active"
	AlertStatusResolved  AlertStatus = "resolved"
	AlertStatusDismissed AlertStatus = "dismissed"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type AlertType string

const (
	AlertTypeRetention       AlertType = "retention"
	AlertTypeUsage           AlertType = "usage"
	AlertTypeNegativeFeedback AlertType = "negative_feedback"
	AlertTypeStorage         AlertType = "storage"
	AlertTypeAPIQuota        AlertType = "api_quota"
	AlertTypeAgentOffline    AlertType = "agent_offline"
)

type ConditionOperator string

const (
	OpLessThan    ConditionOperator = "lt"
	OpGreaterThan ConditionOperator = "gt"
	OpEquals      ConditionOperator = "eq"
)

type AlertRule struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Type         AlertType         `json:"type"`
	Severity     Severity          `json:"severity"`
	Condition    string            `json:"condition"` // e.g., "retention_rate < 30"
	Threshold    float64           `json:"threshold"`
	Operator     ConditionOperator `json:"operator"`
	Enabled      bool              `json:"enabled"`
	NotifyEmail  bool              `json:"notify_email"`
	NotifyWebhook bool             `json:"notify_webhook"`
	NotifyInApp  bool              `json:"notify_in_app"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type Alert struct {
	ID        uuid.UUID   `json:"id"`
	RuleID    uuid.UUID   `json:"rule_id"`
	RuleName  string      `json:"rule_name"`
	Type      AlertType   `json:"type"`
	Severity  Severity    `json:"severity"`
	Message   string      `json:"message"`
	Value     float64     `json:"value"`
	Threshold float64     `json:"threshold"`
	Status    AlertStatus `json:"status"`
	TriggeredAt time.Time `json:"triggered_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type CreateAlertRuleRequest struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Type         AlertType         `json:"type"`
	Severity     Severity          `json:"severity"`
	Condition    string            `json:"condition"`
	Threshold    float64           `json:"threshold"`
	Operator     ConditionOperator `json:"operator"`
	NotifyEmail  bool              `json:"notify_email"`
	NotifyWebhook bool             `json:"notify_webhook"`
	NotifyInApp  bool              `json:"notify_in_app"`
}

type UpdateAlertRuleRequest struct {
	Name         *string           `json:"name,omitempty"`
	Description  *string           `json:"description,omitempty"`
	Condition    *string           `json:"condition,omitempty"`
	Threshold    *float64          `json:"threshold,omitempty"`
	Operator     *ConditionOperator `json:"operator,omitempty"`
	Severity     *Severity         `json:"severity,omitempty"`
	Enabled      *bool             `json:"enabled,omitempty"`
	NotifyEmail  *bool             `json:"notify_email,omitempty"`
	NotifyWebhook *bool            `json:"notify_webhook,omitempty"`
	NotifyInApp  *bool             `json:"notify_in_app,omitempty"`
}

type AlertRuleListResponse struct {
	Rules []AlertRule `json:"rules"`
	Total int         `json:"total"`
}

type AlertListResponse struct {
	Alerts []Alert `json:"alerts"`
	Total  int     `json:"total"`
}

type DefaultRules struct {
	Rules []AlertRule
}

func GetDefaultRules() []AlertRule {
	now := time.Now()
	return []AlertRule{
		{
			ID:          uuid.New(),
			Name:        "Low Retention Rate",
			Description: "Alert when user retention drops below threshold",
			Type:        AlertTypeRetention,
			Severity:    SeverityWarning,
			Condition:   "retention_rate",
			Threshold:   30,
			Operator:    OpLessThan,
			Enabled:     true,
			NotifyInApp: true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New(),
			Name:        "Declining Usage",
			Description: "Alert when daily active users drop significantly",
			Type:        AlertTypeUsage,
			Severity:    SeverityWarning,
			Condition:   "daily_active_users",
			Threshold:   20,
			Operator:    OpLessThan,
			Enabled:     true,
			NotifyInApp: true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New(),
			Name:        "Negative Feedback Spike",
			Description: "Alert when negative feedback ratio exceeds threshold",
			Type:        AlertTypeNegativeFeedback,
			Severity:    SeverityCritical,
			Condition:   "negative_ratio",
			Threshold:   15,
			Operator:    OpGreaterThan,
			Enabled:     true,
			NotifyInApp: true,
			NotifyEmail: true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New(),
			Name:        "API Quota Warning",
			Description: "Alert when daily API calls exceed threshold",
			Type:        AlertTypeAPIQuota,
			Severity:    SeverityInfo,
			Condition:   "api_calls",
			Threshold:   10000,
			Operator:    OpGreaterThan,
			Enabled:     false,
			NotifyInApp: true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New(),
			Name:        "Agent Offline",
			Description: "Alert when all agents are offline",
			Type:        AlertTypeAgentOffline,
			Severity:    SeverityCritical,
			Condition:   "active_agents",
			Threshold:   1,
			Operator:    OpLessThan,
			Enabled:     true,
			NotifyInApp: true,
			NotifyEmail: true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}