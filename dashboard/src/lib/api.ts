const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.ai";
const PROXY_URL = "/api/proxy";

let currentApiKey: string | null = null;
const ADMIN_API_KEY = process.env.ADMIN_API_KEY || "";

export function setApiKey(key: string) {
  currentApiKey = key;
}

export function getApiKey(): string | null {
  return currentApiKey;
}

export function clearApiKey() {
  currentApiKey = null;
}

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
  useAdminKey?: boolean;
}

async function request<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const { params, useAdminKey, ...fetchOptions } = options;

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

  const apiKey = useAdminKey ? ADMIN_API_KEY : currentApiKey;

  if (typeof window !== "undefined") {
    let url = `${PROXY_URL}?endpoint=${encodeURIComponent(endpoint)}${searchParams}`;
    
    const response = await fetch(url, {
      method: fetchOptions.method || "GET",
      headers: {
        "Content-Type": "application/json",
        ...(apiKey && { "x-api-key": apiKey }),
        ...fetchOptions.headers,
      },
      body: fetchOptions.body,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: response.statusText }));
      throw new Error(error.message || `HTTP error! status: ${response.status}`);
    }

    return response.json();
  } else {
    let url = `${API_BASE_URL}${endpoint}${searchParams}`;
    
    const headers: HeadersInit = {
      "Content-Type": "application/json",
      ...(apiKey && { "X-API-Key": apiKey }),
      ...fetchOptions.headers,
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
  scope: "read" | "write" | "admin";
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
    top_queries: string[] | null;
    zero_result_queries: number;
    top_recall_memories: string[] | null;
  };
  skill_metrics: {
    total_skills: number;
    active_skills: number;
    top_skills: string[] | null;
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
  list: (params?: { user_id?: string; org_id?: string; agent_id?: string; category?: string; limit?: number }) =>
    request<{ memories: Memory[]; count: number }>("/memories", { params }),
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
  suggest: (params: { trigger: string; context?: string; limit?: number }) =>
    request<{ skills: Skill[] }>("/skills/suggest", { params }),
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
  list: () => request<APIKey[]>("/admin/api-keys", { useAdminKey: true }),
  create: (data: { label?: string; scope?: "read" | "write" | "admin"; expires_in_hours?: number }) =>
    request<{ id: string; key: string; label: string; tenant: string; expires?: string }>(
      "/admin/api-keys",
      { method: "POST", body: JSON.stringify(data), useAdminKey: true }
    ),
  delete: (id: string) => request<void>(`/admin/api-keys/${id}`, { method: "DELETE", useAdminKey: true }),
};

export const userApiKeysApi = {
  list: () => request<APIKey[]>("/api-keys"),
  create: (data: { label?: string; scope?: "read" | "write"; expires_in_hours?: number }) =>
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
  markAllRead: (userId?: string) =>
    request<{ success: boolean }>(`/notifications/read-all${userId ? `?user_id=${userId}` : ""}`, { method: "POST" }),
  archive: (id: string) =>
    request<{ success: boolean }>(`/notifications/${id}/archive`, { method: "POST" }),
  archiveAll: (userId?: string) =>
    request<{ success: boolean }>(`/notifications/archive-all${userId ? `?user_id=${userId}` : ""}`, { method: "POST" }),
  delete: (id: string) => request<void>(`/notifications/${id}`, { method: "DELETE" }),
  summary: (userId?: string) =>
    request<NotificationSummary>(`/notifications/summary${userId ? `?user_id=${userId}` : ""}`),
  getPreferences: (userId?: string) =>
    request<NotificationPreferences>(`/notifications/preferences${userId ? `?user_id=${userId}` : ""}`),
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
  url: string;
  events: string[];
  active: boolean;
  secret?: string;
  last_triggered?: string;
  created_at: string;
  updated_at: string;
}

export const webhooksApi = {
  list: () => request<{ webhooks: Webhook[] }>("/webhooks"),
  get: (id: string) => request<Webhook>(`/webhooks/${id}`),
  create: (data: Partial<Webhook>) =>
    request<Webhook>("/webhooks", { method: "POST", body: JSON.stringify(data) }),
  update: (id: string, data: Partial<Webhook>) =>
    request<Webhook>(`/webhooks/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/webhooks/${id}`, { method: "DELETE" }),
  test: (id: string) => request<{ success: boolean; message?: string }>(`/webhooks/${id}/test`, { method: "POST" }),
};

export interface AgentGroup {
  id: string;
  name: string;
  description?: string;
  members?: Agent[];
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
  list: () => request<{ users: User[]; total: number }>("/admin/users", { useAdminKey: true }),
  create: (data: { email: string; name: string; role: string }) =>
    request<User>("/admin/users", { method: "POST", body: JSON.stringify(data), useAdminKey: true }),
  update: (id: string, data: { name?: string; role?: string; status?: string }) =>
    request<User>(`/admin/users/${id}`, { method: "PUT", body: JSON.stringify(data), useAdminKey: true }),
  delete: (id: string) =>
    request<{ status: string }>(`/admin/users/${id}`, { method: "DELETE", useAdminKey: true }),
  listInvites: () => request<{ invites: Invite[]; total: number }>("/admin/invites", { useAdminKey: true }),
  createInvite: (data: { email: string; role: string }) =>
    request<Invite>("/admin/invites", { method: "POST", body: JSON.stringify(data), useAdminKey: true }),
  acceptInvite: (id: string) =>
    request<{ success: boolean }>(`/admin/invites/${id}/accept`, { method: "POST", useAdminKey: true }),
  cancelInvite: (id: string) =>
    request<{ status: string }>(`/admin/invites/${id}`, { method: "DELETE", useAdminKey: true }),
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
    searchEnhanced: (query: string, mode: string = "spreading") => 
      request<{ results: EnhancedSearchResult[]; mode: string }>(`/search/enhanced?query=${encodeURIComponent(query)}&mode=${mode}`),
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

export interface Entity {
  name: string;
  type: string;
  role?: string;
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
      useAdminKey: true,
    });
  },

  async testSearch(req: PlaygroundSearchRequest): Promise<PlaygroundSearchResult> {
    return request<PlaygroundSearchResult>("/playground/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
      useAdminKey: true,
    });
  },

  async getStats(): Promise<PlaygroundStats> {
    return request<PlaygroundStats>("/playground/stats", {
      useAdminKey: true,
    });
  },
};