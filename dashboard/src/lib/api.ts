const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.com";
const PROXY_URL = "/api/proxy";

let currentSessionToken: string | null = null;

export function setSessionToken(token: string) {
  currentSessionToken = token;
  if (typeof window !== "undefined") {
    try {
      localStorage.setItem("hystersis_session_token", token);
    } catch (e) {
      // ignore
    }
  }
}

export function clearSessionToken() {
  currentSessionToken = null;
  if (typeof window !== "undefined") {
    try {
      localStorage.removeItem("hystersis_session_token");
    } catch (e) {
      // ignore
    }
  }
}

function getSessionToken(): string | null {
  if (currentSessionToken) return currentSessionToken;
  if (typeof window !== "undefined") {
    try {
      const token = localStorage.getItem("hystersis_session_token");
      if (token) {
        currentSessionToken = token;
        return token;
      }
    } catch (e) {
      // ignore
    }
  }
  return null;
}

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
}

async function request<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const { params, ...fetchOptions } = options;

  let searchParams = "";
  if (params) {
    const sp = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        sp.append(key, String(value));
      }
    });
    const qs = sp.toString();
    if (qs) searchParams = `?${qs}`;
  }

  const isFormData = fetchOptions.body instanceof FormData;
  const sessionToken = getSessionToken();

  if (typeof window !== "undefined") {
    const url = `${PROXY_URL}?endpoint=${encodeURIComponent(endpoint)}${searchParams}`;
    
    const headers: HeadersInit = {
      ...(sessionToken && { "Authorization": `Bearer ${sessionToken}` }),
      ...(!isFormData && { "Content-Type": "application/json" }),
      ...(fetchOptions.headers as Record<string, string> || {}),
    };

    const response = await fetch(url, {
      method: fetchOptions.method || "GET",
      headers,
      body: isFormData ? fetchOptions.body : fetchOptions.body,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: response.statusText }));
      throw new Error(error.message || `HTTP error! status: ${response.status}`);
    }

    if (response.status === 204) {
      return {} as T;
    }

    return response.json();
  } else {
    const url = `${API_BASE_URL}${endpoint}${searchParams}`;
    
    const headers: HeadersInit = {
      ...(!isFormData && { "Content-Type": "application/json" }),
      ...(sessionToken && { "Authorization": `Bearer ${sessionToken}` }),
      ...(!isFormData && (fetchOptions.headers as Record<string, string> || {})),
    };

    const response = await fetch(url, {
      ...fetchOptions,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: response.statusText }));
      throw new Error(error.message || `HTTP error! status: ${response.status}`);
    }

    if (response.status === 204) {
      return {} as T;
    }

    return response.json();
  }
}

export interface Memory {
  id: string;
  content: string;
  type: "conversation" | "session" | "user" | "org";
  user_id?: string;
  org_id?: string;
  agent_id?: string;
  category?: string;
  importance?: "critical" | "high" | "medium" | "low";
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  tags?: string[];
}

export interface Entity {
  id: string;
  name: string;
  type: string;
  role?: string;
  properties?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Agent {
  id: string;
  name: string;
  status: "active" | "inactive" | "suspended";
  config?: {
    max_memories?: number;
    auto_extract?: boolean;
    sharing_policy?: string;
    skill_domains?: string[];
  };
  created_at: string;
  updated_at: string;
}

export interface Skill {
  id: string;
  name: string;
  description: string;
  trigger: string;
  domain: string;
  prompt?: string;
  is_builtin?: boolean;
  usage_count?: number;
  created_at: string;
  updated_at: string;
}

export interface APIKey {
  id: string;
  key?: string;
  label: string;
  scope: string;
  tenant_id: string;
  created_at: string;
  expires_at?: string;
  usage_count: number;
}

export interface Chain {
  id: string;
  name: string;
  trigger: string;
  steps: Array<{ skill_id: string; order: number }>;
  conditions?: Array<{ field: string; operator: string; value: string; action: string }>;
  confidence: number;
  created_at: string;
  updated_at: string;
}

export interface ChainExecution {
  id: string;
  chain_id: string;
  status: "pending" | "running" | "completed" | "failed";
  result?: unknown;
  error?: string;
  started_at: string;
  completed_at?: string;
}

export interface Analytics {
  period: string;
  generated_at: string;
  memory_growth: {
    total_created: number;
    total_archived: number;
    total_deleted: number;
    by_category: Record<string, number>;
    by_type: Record<string, number>;
    by_importance: Record<string, number> | null;
  };
  search_analytics: {
    total_searches: number;
    avg_results_per_query: number;
    top_queries: Array<{ query: string; count: number }> | null;
    zero_result_queries: number;
    top_recall_memories: Array<{ memory_id: string; content: string; recall_count: number }> | null;
  };
  skill_metrics: {
    total_skills: number;
    active_skills: number;
    top_skills: Array<{ skill_id: string; name: string; usage_count: number; success_rate: number; confidence: number }> | null;
    chain_usage: {
      total_chains: number;
      total_executions: number;
      success_rate: number;
      avg_steps_per_chain: number;
    };
    avg_confidence: number;
    skills_by_domain: Record<string, number>;
  };
  agent_activity: null;
  retention: {
    period: string;
    active_users: number;
    returning_users: number;
    retention_rate: number;
    avg_memories_per_user: number;
  };
}

export interface GraphNode {
  id: string;
  name: string;
  type: string;
  properties?: Record<string, unknown>;
}

export interface GraphLink {
  source: string;
  target: string;
  type: string;
}

export interface GraphData {
  nodes: GraphNode[];
  links: GraphLink[];
}

export const memoriesApi = {
  list: (params?: { user_id?: string; org_id?: string; agent_id?: string; category?: string; limit?: number; offset?: number }) =>
    request<{ memories: Memory[]; total: number; count: number; limit: number; offset: number }>("/memories", { params }),
  get: (id: string) => request<Memory>(`/memories/${id}`),
  create: (data: Partial<Memory>) =>
    request<Memory>("/memories", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Memory>) =>
    request<Memory>(`/memories/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/memories/${id}`, { method: "DELETE" }),
  search: (params: { q: string; limit?: number; threshold?: number }) =>
    request<{ memories: Memory[]; count: number }>("/search", { params }),
};

export const entitiesApi = {
  list: (params?: { limit?: number }) =>
    request<{ entities: Entity[] }>("/entities", { params }),
  get: (id: string) => request<Entity>(`/entities/${id}`),
  create: (data: Partial<Entity>) =>
    request<Entity>("/entities", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Entity>) =>
    request<Entity>(`/entities/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/entities/${id}`, { method: "DELETE" }),
  getRelations: (id: string) =>
    request<{ relations: Array<{ id: string; from_id: string; to_id: string; type: string }> }>(`/entities/${id}/relations`),
  getMemories: (id: string) =>
    request<{ memories: Memory[] }>(`/entities/${id}/memories`),
};

export const sessionsApi = {
  list: () => request<{ sessions: Session[] }>("/sessions"),
  get: (id: string) => request<Session>(`/sessions/${id}`),
  create: (data: { agent_id: string; metadata?: Record<string, unknown> }) =>
    request<Session>("/sessions", { method: "POST", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/sessions/${id}`, { method: "DELETE" }),
  getMessages: (id: string, params?: { limit?: number }) =>
    request<{ messages: Array<{ id: string; role: string; content: string; created_at: string }> }>(
      `/sessions/${id}/messages`,
      { params }
    ),
  addMessage: (id: string, data: { role: string; content: string }) =>
    request<{ id: string }>(`/sessions/${id}/messages`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  getContext: (id: string) =>
    request<{ context: unknown }>(`/sessions/${id}/context`),
};

export interface Session {
  id: string;
  agent_id: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  last_message_at?: string;
}

export const agentsApi = {
  list: (params?: { tenant_id?: string; limit?: number; offset?: number }) =>
    request<{ agents: Agent[] }>("/agents", { params }),
  get: (id: string) => request<Agent>(`/agents/${id}`),
  create: (data: Partial<Agent>) =>
    request<Agent>("/agents", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Agent>) =>
    request<Agent>(`/agents/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/agents/${id}`, { method: "DELETE" }),
};

export const skillsApi = {
  list: (params?: { tenant_id?: string; domain?: string; limit?: number; offset?: number }) =>
    request<{ skills: Skill[] }>("/skills", { params }),
  get: (id: string) => request<Skill>(`/skills/${id}`),
  create: (data: Partial<Skill>) =>
    request<Skill>("/skills", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Skill>) =>
    request<Skill>(`/skills/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/skills/${id}`, { method: "DELETE" }),
  suggest: (data: { trigger: string; context?: string; limit?: number }) =>
    request<{ skills: Skill[] }>("/skills/suggest", { method: "POST", body: JSON.stringify(data) }),
  use: (id: string, data?: { input?: string; context?: Record<string, unknown> }) =>
    request<{ result: unknown }>(`/skills/${id}/use`, { method: "POST", body: JSON.stringify(data || {}) }),
};

export const chainsApi = {
  list: (params?: { tenant_id?: string }) =>
    request<{ chains: Chain[] }>("/chains", { params }),
  get: (id: string) => request<Chain>(`/chains/${id}`),
  create: (data: Partial<Chain>) =>
    request<Chain>("/chains", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Chain>) =>
    request<Chain>(`/chains/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/chains/${id}`, { method: "DELETE" }),
  execute: (id: string, context?: Record<string, unknown>) =>
    request<{ result: unknown }>(`/chains/${id}/execute`, {
      method: "POST",
      body: JSON.stringify(context),
    }),
  getExecutions: (id: string) =>
    request<{ executions: ChainExecution[] }>(`/chains/${id}/executions`),
};

export const apiKeysApi = {
  list: () => request<APIKey[]>("/admin/api-keys"),
  create: (data: { label?: string; scope?: string; expires_in_hours?: number }) =>
    request<{ id: string; key: string; label: string; tenant: string; expires?: string }>(
      "/admin/api-keys",
      { method: "POST", body: JSON.stringify(data) }
    ),
  delete: (id: string) => request<void>(`/admin/api-keys/${id}`, { method: "DELETE" }),
};

export const userApiKeysApi = {
  list: () => request<APIKey[]>("/api-keys"),
  create: (data: { label?: string; scope?: string; expires_in_hours?: number }) =>
    request<{ id: string; key: string; label: string; tenant: string; expires?: string }>(
      "/api-keys",
      { method: "POST", body: JSON.stringify(data) }
    ),
  delete: (id: string) => request<void>(`/api-keys/${id}`, { method: "DELETE" }),
};

export const analyticsApi = {
  dashboard: (params?: { tenant_id?: string; period?: string }) =>
    request<Analytics>("/analytics/dashboard", { params }),
};

export interface BillingUsage {
  tenant_id: string;
  tier: string;
  memory_count: number;
  search_count: number;
  period_start?: string;
  period_end?: string;
}

export interface BillingSubscription {
  tenant_id: string;
  tier: string;
  status: string;
}

export const billingApi = {
  getUsage: () => request<BillingUsage>("/billing/usage"),
  getSubscription: () => request<BillingSubscription>("/billing/subscription"),
  createCheckout: (plan: string) =>
    request<{ url: string }>("/stripe/checkout", {
      method: "POST",
      body: JSON.stringify({ plan }),
    }),
};

export const authApi = {
  updateProfile: (data: { name?: string; org_id?: string; avatar_url?: string }) =>
    request<{ success: boolean; user?: User }>("/admin/users/me", {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  changePassword: (data: { current_password: string; new_password: string }) =>
    request<{ success: boolean }>("/auth/change-password", {
      method: "POST",
      body: JSON.stringify(data),
    }),
};

export const graphApi = {
  traverse: (entityId: string, depth?: number) =>
    request<GraphData>(`/graph/traverse/${entityId}`, { params: { depth: depth || 3 } }),
  query: (cypher: string, params?: Record<string, unknown>) =>
    request<{ results: unknown }>("/graph/query", {
      method: "POST",
      body: JSON.stringify({ cypher, params }),
    }),
};

export const systemApi = {
  health: () => request<{ status: string }>("/health"),
  ready: () => request<{ status: string; neo4j: boolean; qdrant: boolean }>("/ready"),
};

export type NotificationType = "info" | "success" | "warning" | "error";
export type NotificationChannel = "in_app" | "email" | "webhook";
export type NotificationStatus = "unread" | "read" | "archived";

export interface Notification {
  id: string;
  tenant_id: string;
  user_id: string;
  type: NotificationType;
  title: string;
  message: string;
  channel: NotificationChannel;
  status: NotificationStatus;
  data?: Record<string, unknown>;
  link?: string;
  read_at?: string;
  expires_at?: string;
  created_at: string;
  updated_at: string;
}

export interface NotificationSummary {
  total: number;
  unread: number;
  read: number;
  archived: number;
  by_type?: Record<NotificationType, number>;
}

export interface NotificationPreferences {
  id: string;
  tenant_id: string;
  user_id: string;
  in_app_enabled: boolean;
  email_enabled: boolean;
  webhook_enabled: boolean;
  email_address?: string;
  webhook_url?: string;
  mute_types?: NotificationType[];
  mute_channels?: NotificationChannel[];
  created_at: string;
  updated_at: string;
}

export const notificationsApi = {
  list: (params?: { user_id?: string; status?: string; type?: string; channel?: string; limit?: number }) =>
    request<{ notifications: Notification[]; total: number; limit: number }>("/notifications", { params }),
  get: (id: string) => request<Notification>(`/notifications/${id}`),
  create: (data: { user_id: string; type: NotificationType; title: string; message: string; channel?: NotificationChannel; data?: Record<string, unknown>; link?: string }) =>
    request<Notification>("/notifications", { method: "POST", body: JSON.stringify(data) }),
  markRead: (id: string) =>
    request<{ success: boolean }>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: (params?: { user_id?: string }) =>
    request<{ success: boolean }>(`/notifications/read-all`, { method: "POST", params }),
  archive: (id: string) =>
    request<{ success: boolean }>(`/notifications/${id}/archive`, { method: "POST" }),
  archiveAll: (params?: { user_id?: string }) =>
    request<{ success: boolean }>(`/notifications/archive-all`, { method: "POST", params }),
  delete: (id: string) => request<void>(`/notifications/${id}`, { method: "DELETE" }),
  summary: (params?: { user_id?: string }) =>
    request<NotificationSummary>(`/notifications/summary`, { params }),
  getPreferences: (params?: { user_id?: string }) =>
    request<NotificationPreferences>(`/notifications/preferences`, { params }),
  updatePreferences: (data: Partial<NotificationPreferences>) =>
    request<NotificationPreferences>("/notifications/preferences", { method: "PUT", body: JSON.stringify(data) }),
};

export interface Project {
  id: string;
  name: string;
  description?: string;
  metadata?: Record<string, unknown>;
  memory_count?: number;
  agent_count?: number;
  created_at: string;
  updated_at: string;
}

export const projectsApi = {
  list: (params?: { limit?: number; offset?: number }) =>
    request<{ projects: Project[]; total: number }>("/projects", { params }),
  get: (id: string) => request<Project>(`/projects/${id}`),
  create: (data: Partial<Project>) =>
    request<Project>("/projects", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Project>) =>
    request<Project>(`/projects/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/projects/${id}`, { method: "DELETE" }),
};

export interface Webhook {
  id: string;
  tenant_id?: string;
  project_id?: string;
  url: string;
  events: string[];
  fields?: string[];
  active: boolean;
  secret?: string;
  verified_at?: string;
  last_triggered?: string;
  success_count?: number;
  failure_count?: number;
  last_delivery_at?: string;
  last_status_code?: number;
  created_at: string;
  updated_at: string;
}

export interface WebhookDelivery {
  id: string;
  webhook_id: string;
  event: string;
  status: "success" | "failed" | "pending";
  response_code?: number;
  error?: string;
  duration_ms?: number;
  created_at: string;
}

export interface WebhookDeadLetterEntry {
  id: string;
  webhook_id: string;
  event: string;
  error?: string;
  attempts: number;
  created_at: string;
}

export const webhooksApi = {
  list: () => request<{ webhooks: Webhook[] }>("/webhooks"),
  get: (id: string) => request<Webhook>(`/webhooks/${id}`),
  create: (data: Partial<Webhook>) =>
    request<Webhook>("/webhooks", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Webhook>) =>
    request<Webhook>(`/webhooks/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/webhooks/${id}`, { method: "DELETE" }),
  test: (id: string) =>
    request<{ success: boolean; message?: string; status_code?: number; event?: string }>(`/webhooks/${id}/test`, {
      method: "POST",
    }),
  getDeliveries: (id: string) =>
    request<{ deliveries: WebhookDelivery[]; webhook_id: string }>(`/webhooks/${id}/deliveries`),
  getDeadLetter: () =>
    request<{ entries: WebhookDeadLetterEntry[] }>("/webhooks/dead-letter"),
  retryDeadLetter: (webhookId: string, event: string) =>
    request<{ success: boolean; message: string }>(`/webhooks/${webhookId}/retry`, {
      method: "POST",
      body: JSON.stringify({ event }),
    }),
};

export interface AgentGroup {
  id: string;
  name: string;
  description?: string;
  members?: Array<{
    agent_id: string;
    group_id: string;
    role: "admin" | "member" | "viewer" | "contributor";
    joined_at: string;
  }>;
  member_count?: number;
  created_at: string;
  updated_at: string;
}

export const groupsApi = {
  list: () => request<{ groups: AgentGroup[]; total: number }>("/groups"),
  get: (id: string) => request<AgentGroup>(`/groups/${id}`),
  create: (data: Partial<AgentGroup>) =>
    request<AgentGroup>("/groups", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<AgentGroup>) =>
    request<AgentGroup>(`/groups/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/groups/${id}`, { method: "DELETE" }),
  addMember: (groupId: string, agentId: string, role?: string) =>
    request<{ success: boolean }>(`/groups/${groupId}/members`, {
      method: "POST",
      body: JSON.stringify({ agent_id: agentId, role }),
    }),
  removeMember: (groupId: string, agentId: string) =>
    request<void>(`/groups/${groupId}/members/${agentId}`, { method: "DELETE" }),
  getSkills: (id: string) =>
    request<{ skills: Skill[] }>(`/groups/${id}/skills`),
  getMemories: (id: string) =>
    request<{ memories: Memory[] }>(`/groups/${id}/memories`),
};

// ============ Users & RBAC ============

export interface User {
  id: string;
  email: string;
  name: string;
  role: "admin" | "member" | "viewer";
  status: "active" | "inactive" | "pending";
  avatar_url?: string;
  created_at: string;
  updated_at: string;
  last_login?: string;
}

export interface Invite {
  id: string;
  email: string;
  role: "admin" | "member" | "viewer";
  status: "pending" | "accepted" | "rejected" | "expired";
  invited_by: string;
  expires_at: string;
  created_at: string;
}

export const usersApi = {
  list: (params?: { tenant_id?: string; limit?: number; offset?: number }) =>
    request<{ users: User[]; total: number }>("/admin/users", { params }),
  get: (id: string) => request<User>(`/admin/users/${id}`),
  create: (data: { email: string; name: string; role: string }) =>
    request<User>("/admin/users", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: { name?: string; role?: string; status?: string }) =>
    request<User>(`/admin/users/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) =>
    request<{ status: string }>(`/admin/users/${id}`, { method: "DELETE" }),
  listInvites: () => request<{ invites: Invite[]; total: number }>("/admin/invites"),
  createInvite: (data: { email: string; role: string }) =>
    request<Invite>("/admin/invites", { method: "POST", body: JSON.stringify(data) }),
  acceptInvite: (id: string) =>
    request<{ success: boolean }>(`/admin/invites/${id}/accept`, { method: "POST" }),
  cancelInvite: (id: string) =>
    request<{ status: string }>(`/admin/invites/${id}`, { method: "DELETE" }),
};

// ============ Alerts ============

export interface AlertRule {
  id: string;
  name: string;
  description: string;
  type: "retention" | "usage" | "negative_feedback" | "storage" | "api_quota" | "agent_offline";
  severity: "info" | "warning" | "critical";
  condition: string;
  threshold: number;
  operator: "lt" | "gt" | "eq";
  enabled: boolean;
  notify_email: boolean;
  notify_webhook: boolean;
  notify_in_app: boolean;
  created_at: string;
  updated_at: string;
}

export interface Alert {
  id: string;
  rule_id: string;
  rule_name: string;
  type: string;
  severity: string;
  message: string;
  value: number;
  threshold: number;
  status: "active" | "resolved" | "dismissed";
  triggered_at: string;
  resolved_at?: string;
}

export const alertsApi = {
  listRules: () => request<{ rules: AlertRule[]; total: number }>("/alerts/rules"),
  createRule: (data: Partial<AlertRule>) =>
    request<AlertRule>("/alerts/rules", { method: "POST", body: JSON.stringify(data) }),
  updateRule: (id: string, data: Partial<AlertRule>) =>
    request<AlertRule>(`/alerts/rules/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteRule: (id: string) =>
    request<{ status: string }>(`/alerts/rules/${id}`, { method: "DELETE" }),
  enableRule: (id: string, enabled: boolean) =>
    request<{ success: boolean }>(`/alerts/rules/${id}/enable`, { method: "PUT", body: JSON.stringify({ enabled }) }),
  listActive: () => request<{ alerts: Alert[]; total: number }>("/alerts/active"),
  resolveAlert: (id: string) =>
    request<{ success: boolean }>(`/alerts/${id}/resolve`, { method: "POST" }),
  dismissAlert: (id: string) =>
    request<{ success: boolean }>(`/alerts/${id}/dismiss`, { method: "POST" }),
  getStats: () => request<Record<string, number>>("/alerts/stats"),

  // Compression Engine APIs (PROPRIETARY)
  compression: {
    getStats: () => request<CompressionStats>("/compression/stats"),
    getMode: () => request<{ mode: string }>("/compression/mode"),
    setMode: (mode: string) => request<{ success: boolean }>("/compression/mode", { method: "PUT", body: JSON.stringify({ mode }) }),
    getTierPolicy: () => request<{ policy: string }>("/tier/policy"),
    setTierPolicy: (policy: string) => request<{ success: boolean }>("/tier/policy", { method: "PUT", body: JSON.stringify({ policy }) }),
  },
};

export interface CompressionStats {
  accuracy_retention: number;
  token_reduction: number;
  total_tokens_saved: number;
  extractions_performed: number;
  spreading_activations: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
}

export interface EnhancedSearchResult {
  id: string;
  content: string;
  score: number;
  mode: string;
  hops?: number;
}

export const CompressionMode = {
  EXTRACT: "extract",
  BALANCED: "balanced",
  AGGRESSIVE: "aggressive",
} as const;

export const TierPolicy = {
  AGGRESSIVE: "aggressive",
  BALANCED: "balanced",
  CONSERVATIVE: "conservative",
} as const;

export const SearchMode = {
  VECTOR: "vector",
  SPREADING: "spreading",
  HYBRID: "hybrid",
} as const;

export const CompressionPlaygroundMode = {
  EXTRACTION: "extraction",
  RELATIONAL: "relational",
  RADIX: "radix",
  HYBRID: "hybrid",
} as const;

export interface PlaygroundCompressionRequest {
  text: string;
  modes?: string[];
  show_entities?: boolean;
  show_facts?: boolean;
  learn_patterns?: boolean;
}

export interface PlaygroundCompressionResult {
  original: string;
  results: Record<string, {
    compressed: string;
    reduction_percent: number;
    token_savings: number;
    latency_ms: number;
    entities?: Entity[];
    facts?: string[];
  }>;
  best_mode: string;
  entities?: Entity[];
  total_latency_ms: number;
}

export interface PlaygroundSearchRequest {
  query: string;
  modes?: string[];
  limit?: number;
  show_graph?: boolean;
  compare_modes?: boolean;
}

export interface PlaygroundSearchResult {
  query: string;
  results: Record<string, SearchResult[]>;
  comparison?: {
    overlap_count: number;
    unique_to_vector: string[];
    unique_to_spreading: string[];
    best_mode: string;
    score_difference: number;
  };
  graph?: {
    nodes: { id: string; label: string; type: string; score?: number }[];
    edges: { from: string; to: string; type: string }[];
  };
  stats: {
    vector_latency_ms: number;
    spreading_latency_ms: number;
    hybrid_latency_ms: number;
    total_results: number;
  };
}

export interface SearchResult {
  id: string;
  content: string;
  score: number;
  hops?: number;
  entity?: string;
}

export interface PlaygroundStats {
  total_requests: number;
  compressions: number;
  searches: number;
  extractions: number;
  avg_latency_ms: number;
}

export const playgroundApi = {
  async testCompression(req: PlaygroundCompressionRequest): Promise<PlaygroundCompressionResult> {
    return request<PlaygroundCompressionResult>("/playground/compress", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
  },

  async testSearch(req: PlaygroundSearchRequest): Promise<PlaygroundSearchResult> {
    return request<PlaygroundSearchResult>("/playground/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
  },

  async getStats(): Promise<PlaygroundStats> {
    return request<PlaygroundStats>("/playground/stats");
  },
};

export const api = {
  memories: memoriesApi,
  entities: entitiesApi,
  sessions: sessionsApi,
  agents: agentsApi,
  groups: groupsApi,
  projects: projectsApi,
  webhooks: webhooksApi,
  skills: skillsApi,
  alerts: alertsApi,
  apiKeys: apiKeysApi,
  playground: playgroundApi,
  compression: {
    getStats: () => request<CompressionStats>("/compression/stats"),
    getMode: () => request<{ mode: string }>("/compression/mode"),
    setMode: (mode: string) => request<void>("/compression/mode", { method: "PUT", body: JSON.stringify({ mode }) }),
    getTierPolicy: () => request<{ policy: string }>("/tier/policy"),
    setTierPolicy: (policy: string) => request<void>("/tier/policy", { method: "PUT", body: JSON.stringify({ policy }) }),
  },
  documents: {
    extract: (file: File) => {
      const formData = new FormData();
      formData.append("file", file);
      return request<{ content: string; title: string; mime_type: string; source: string; metadata: Record<string, string>; pages: number }>(
        "/documents/extract",
        { method: "POST", body: formData }
      );
    },
  },
  graph: {
    traverse: (entityId: string, depth?: number) =>
      request<{ nodes: unknown[]; edges: unknown[] }>(`/graph/traverse/${entityId}${depth ? `?depth=${depth}` : ""}`),
    query: (cypher: string, params?: Record<string, unknown>) =>
      request<{ results: unknown[] }>("/graph/query", { method: "POST", body: JSON.stringify({ cypher, params: params || {} }) }),
  },
  search: {
    hybrid: (req: { query: string; semantic_weight?: number; keyword_weight?: number; limit?: number }) =>
      request<{ results: unknown[]; count: number }>("/search/hybrid", { method: "POST", body: JSON.stringify(req) }),
    advanced: (req: { query: string; filters?: Record<string, unknown>; limit?: number }) =>
      request<{ results: unknown[]; count: number }>("/search/advanced", { method: "POST", body: JSON.stringify(req) }),
    enhanced: (query: string, mode: string) =>
      request<{ results: EnhancedSearchResult[]; mode: string }>("/search/enhanced", { params: { query, mode } }),
  },
  feedback: {
    create: (data: { memory_id?: string; feedback_type: string; content: string }) =>
      request<void>("/feedback", { method: "POST", body: JSON.stringify(data) }),
    list: (params?: Record<string, string | number | boolean | undefined>) =>
      request<unknown[]>("/feedback", { params }),
  },
  metrics: {
    compression: () => request<{
      ExtractionsTotal: number;
      ExtractionsByProvider: Record<string, number>;
      SpreadingActivationsTotal: number;
      CompressionLatencyMs: number;
      TokensSavedTotal: number;
      AccuracyRetention: number;
      CacheHits: number;
      CacheMisses: number;
      TierHits: Record<string, number>;
    }>("/metrics/compression"),
  },
};

export interface AuditEvent {
  id: string;
  tenant_id: string;
  timestamp: string;
  type: string;
  actor_id: string;
  actor_type: string;
  resource_type: string;
  resource_id: string;
  action: string;
  status: string;
  ip_address?: string;
  user_agent?: string;
  metadata?: Record<string, unknown>;
  error?: string;
  duration_ms?: number;
}

export const auditApi = {
  query: (params?: {
    types?: string;
    actor_id?: string;
    start_time?: string;
    end_time?: string;
    limit?: number;
    offset?: number;
  }) =>
    request<AuditEvent[]>("/audit/events", { params: params as Record<string, string | number | boolean | undefined> }),
  export: (params?: { start_time?: string; end_time?: string; format?: string }) =>
    request<Blob>("/audit/export", { params: params as Record<string, string | number | boolean | undefined> }),
};

export const sourcesApi = {
  list: () => request<{ sources: Array<{ id: string; name: string; type: string; status: string; created_at: string }> }>("/sources"),
  get: (id: string) => request<{ id: string; name: string; type: string; content: string }>(`/sources/${id}`),
  delete: (id: string) => request<void>(`/sources/${id}`, { method: "DELETE" }),
};
