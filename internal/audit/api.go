package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type APIAccessLog struct {
	Method       string
	Path         string
	StatusCode   int
	DurationMs   int64
	ActorID      string
	TenantID     string
	IPAddress    string
	UserAgent    string
	ErrorMessage string
}

type APIEventMapper struct {
	patterns []apiPattern
}

type apiPattern struct {
	method    string
	pathRegex *regexp.Regexp
	eventType EventType
	action    string
}

func NewAPIEventMapper() *APIEventMapper {
	mapper := &APIEventMapper{
		patterns: []apiPattern{
			{method: "POST", pathRegex: regexp.MustCompile("/memories"), eventType: EventTypeMemoryCreate, action: "create memory"},
			{method: "GET", pathRegex: regexp.MustCompile("^/memories/"), eventType: EventTypeMemoryRead, action: "read memory"},
			{method: "PUT", pathRegex: regexp.MustCompile("^/memories/"), eventType: EventTypeMemoryUpdate, action: "update memory"},
			{method: "DELETE", pathRegex: regexp.MustCompile("^/memories/"), eventType: EventTypeMemoryDelete, action: "delete memory"},
			{method: "GET", pathRegex: regexp.MustCompile("/memories"), eventType: EventTypeMemorySearch, action: "list memories"},
			{method: "POST", pathRegex: regexp.MustCompile("/search"), eventType: EventTypeMemorySearch, action: "search memories"},
			{method: "POST", pathRegex: regexp.MustCompile("/skills"), eventType: EventTypeSkillCreate, action: "create skill"},
			{method: "POST", pathRegex: regexp.MustCompile("/skills/.*/use"), eventType: EventTypeSkillUse, action: "use skill"},
			{method: "POST", pathRegex: regexp.MustCompile("/skills/extract"), eventType: EventTypeSkillExtract, action: "extract skills"},
			{method: "POST", pathRegex: regexp.MustCompile("/skills/synthesize"), eventType: EventTypeSkillSynthesize, action: "synthesize skills"},
			{method: "POST", pathRegex: regexp.MustCompile("/agents"), eventType: EventTypeAgentCreate, action: "create agent"},
			{method: "PUT", pathRegex: regexp.MustCompile("^/agents/"), eventType: EventTypeAgentUpdate, action: "update agent"},
			{method: "DELETE", pathRegex: regexp.MustCompile("^/agents/"), eventType: EventTypeAgentDelete, action: "delete agent"},
			{method: "POST", pathRegex: regexp.MustCompile("/groups"), eventType: EventTypeGroupCreate, action: "create group"},
			{method: "POST", pathRegex: regexp.MustCompile("/groups/.*/members"), eventType: EventTypeGroupAddMember, action: "add group member"},
			{method: "DELETE", pathRegex: regexp.MustCompile("/groups/.*/members"), eventType: EventTypeGroupRemoveMember, action: "remove group member"},
			{method: "POST", pathRegex: regexp.MustCompile("/feedback"), eventType: EventTypeReviewRequest, action: "submit feedback"},
			{method: "POST", pathRegex: regexp.MustCompile("/webhooks"), eventType: EventTypeAPI, action: "create webhook"},
			{method: "POST", pathRegex: regexp.MustCompile("/backup/export"), eventType: EventTypeAPI, action: "export backup"},
			{method: "POST", pathRegex: regexp.MustCompile("/backup/import"), eventType: EventTypeAPI, action: "import backup"},
			{method: "POST", pathRegex: regexp.MustCompile("/compact"), eventType: EventTypeAPI, action: "run compaction"},
		},
	}

	return mapper
}

func (m *APIEventMapper) Map(method, path string) (EventType, string) {
	for _, p := range m.patterns {
		if p.method == method && p.pathRegex.MatchString(path) {
			return p.eventType, p.action
		}
	}
	return EventTypeAPI, method + " " + path
}

func (a *APIAccessLog) ToEvent() *Event {
	eventType, action := NewAPIEventMapper().Map(a.Method, a.Path)

	status := "success"
	if a.StatusCode >= 400 {
		status = "failure"
	}

	return &Event{
		ID:           "",
		TenantID:     a.TenantID,
		Timestamp:    time.Now(),
		Type:         eventType,
		ActorID:      a.ActorID,
		ActorType:    "api_key",
		ResourceType: "api",
		Action:       action,
		Status:       status,
		IPAddress:    a.IPAddress,
		UserAgent:    a.UserAgent,
		DurationMs:   a.DurationMs,
		Error:        a.ErrorMessage,
	}
}

type AuditMiddleware struct {
	logger       Logger
	mapper       *APIEventMapper
	excludePaths map[string]bool
}

func NewAuditMiddleware(logger Logger) *AuditMiddleware {
	excludePaths := map[string]bool{
		"/health":  true,
		"/ready":   true,
		"/metrics": true,
	}

	return &AuditMiddleware{
		logger:       logger,
		mapper:       NewAPIEventMapper(),
		excludePaths: excludePaths,
	}
}

func (m *AuditMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.excludePaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		tenantID := ""
		actorID := ""

		if tenant := r.Context().Value("tenant_id"); tenant != nil {
			tenantID = tenant.(string)
		}
		if actor := r.Context().Value("actor_id"); actor != nil {
			actorID = actor.(string)
		}

		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" && actorID == "" {
			actorID = apiKey[:8] + "..."
		}

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		accessLog := &APIAccessLog{
			Method:     r.Method,
			Path:       r.URL.Path,
			StatusCode: rw.statusCode,
			DurationMs: duration.Milliseconds(),
			ActorID:    actorID,
			TenantID:   tenantID,
			IPAddress:  getClientIP(r),
			UserAgent:  r.UserAgent(),
		}

		if rw.statusCode >= 400 {
			accessLog.ErrorMessage = http.StatusText(rw.statusCode)
		}

		event := accessLog.ToEvent()

		ctx := context.Background()
		m.logger.Log(ctx, event)
	})
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return r.RemoteAddr
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

type AuditHandler struct {
	logger Logger
}

func NewAuditHandler(logger Logger) *AuditHandler {
	return &AuditHandler{logger: logger}
}

func (h *AuditHandler) QueryEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	startTime := r.URL.Query().Get("start_time")
	endTime := r.URL.Query().Get("end_time")
	eventTypes := r.URL.Query().Get("types")
	actorID := r.URL.Query().Get("actor_id")
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	filter := &Filter{
		TenantID: tenantID,
		ActorID:  actorID,
	}

	if startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}

	if endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	if eventTypes != "" {
		types := strings.Split(eventTypes, ",")
		for _, t := range types {
			filter.Types = append(filter.Types, EventType(strings.TrimSpace(t)))
		}
	}

	if limit != "" {
		if l, err := parseInt(limit); err == nil {
			filter.Limit = l
		}
	}

	if offset != "" {
		if o, err := parseInt(offset); err == nil {
			filter.Offset = o
		}
	}

	ctx := context.Background()
	events, err := h.logger.Query(ctx, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(events)
}

func (h *AuditHandler) ExportEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	startTime := r.URL.Query().Get("start_time")
	endTime := r.URL.Query().Get("end_time")
	format := r.URL.Query().Get("format")

	if format == "" {
		format = "json"
	}

	filter := &Filter{
		TenantID: tenantID,
	}

	if startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}

	if endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	ctx := context.Background()
	data, err := h.logger.Export(ctx, filter, format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("audit_export_%s.%s", time.Now().Format("20060102_150405"), format)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number")
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}
