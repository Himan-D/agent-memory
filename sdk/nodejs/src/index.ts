/**
 * Agent Memory System - Node.js SDK
 * 
 * Persistent memory for AI agents with graph relationships and semantic search.
 * 
 * @example
 * ```typescript
 * import { AgentMemory } from 'agent-memory';
 * 
 * const client = new AgentMemory({
 *   baseUrl: 'http://localhost:8080',
 *   apiKey: 'your-api-key'
 * });
 * 
 * // Create a session
 * const session = await client.sessions.create({ agentId: 'my-agent' });
 * 
 * // Add messages
 * await client.messages.add(session.id, 'user', 'Hello!');
 * 
 * // Search memories
 * const results = await client.memories.search({
 *   query: 'previous conversations',
 *   limit: 10
 * });
 * ```
 */

export type MemoryType = 'conversation' | 'session' | 'user' | 'org';
export type FeedbackType = 'positive' | 'negative' | 'very_negative';
export type MemoryStatus = 'active' | 'archived' | 'deleted';
export type ImportanceLevel = 'critical' | 'high' | 'medium' | 'low';
export type MemoryLinkType = 'parent' | 'related' | 'reply' | 'cite';
export type MemberRole = 'admin' | 'contributor' | 'reader';
export type ReviewStatus = 'pending' | 'approved' | 'rejected';
export type AgentStatus = 'active' | 'inactive' | 'suspended';

export interface Memory {
  id: string;
  tenantId?: string;
  userId?: string;
  orgId?: string;
  agentId?: string;
  sessionId?: string;
  type: MemoryType;
  content: string;
  category?: string;
  entityId?: string;
  metadata?: Record<string, unknown>;
  status: MemoryStatus;
  immutable: boolean;
  expirationDate?: string;
  feedbackScore?: FeedbackType;
  createdAt: string;
  updatedAt: string;
  lastAccessed?: string;
  tags?: string[];
  importance?: ImportanceLevel;
  accessCount?: number;
  links?: MemoryLink[];
  versions?: MemoryVersion[];
}

export interface MemoryLink {
  id: string;
  fromId: string;
  toId: string;
  type: MemoryLinkType;
  weight: number;
  metadata?: Record<string, unknown>;
  createdAt: string;
}

export interface MemoryVersion {
  id: string;
  memoryId: string;
  content: string;
  version: number;
  metadata?: Record<string, unknown>;
  createdAt: string;
  createdBy?: string;
}

export interface MemoryStats {
  totalMemories: number;
  activeMemories: number;
  archivedMemories: number;
  expiredMemories: number;
  byCategory: Record<string, number>;
  byType: Record<string, number>;
  byImportance: Record<string, number>;
  avgAccessCount: number;
  totalLinks: number;
}

export interface MemoryInsights {
  insight: string;
  category: string;
  evidenceCount: number;
  relatedMemories: number;
}

export interface MemorySummary {
  summary: string;
  keyPoints: string[];
  memoryCount: number;
  tokenSavings: number;
  compressedMemories: number;
}

export interface CompactionStatus {
  status: 'idle' | 'running' | 'completed' | 'failed';
  action?: string;
  startedAt?: string;
  completedAt?: string;
  memoriesProcessed?: number;
  tokensSaved?: number;
  error?: string;
}

export interface Message {
  id: string;
  tenantId?: string;
  sessionId: string;
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
  timestamp: string;
}

export interface Session {
  id: string;
  tenantId?: string;
  agentId: string;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface Entity {
  id: string;
  tenantId?: string;
  type: string;
  name: string;
  properties?: Record<string, unknown>;
  embedding?: number[];
  createdAt: string;
  updatedAt: string;
  lastSynced?: string;
}

export interface Relation {
  id: string;
  tenantId?: string;
  fromId: string;
  toId: string;
  type: string;
  weight: number;
  metadata?: Record<string, unknown>;
}

export interface MemoryResult {
  entity: Entity;
  score: number;
  text: string;
  source: string;
  memoryId?: string;
  metadata?: Memory;
}

export interface SearchRequest {
  query: string;
  limit?: number;
  offset?: number;
  threshold?: number;
  filters?: SearchFilters;
  memoryType?: MemoryType;
  userId?: string;
  orgId?: string;
  agentId?: string;
  category?: string;
  rerank?: boolean;
  rerankTopK?: number;
}

export interface HybridSearchRequest {
  query: string;
  semanticLimit?: number;
  keywordLimit?: number;
  boost?: number;
  threshold?: number;
  filters?: SearchFilters;
  userId?: string;
  orgId?: string;
}

export interface SearchFilters {
  logic: 'AND' | 'OR' | 'NOT';
  rules: SearchFilter[];
  nested?: SearchFilters[];
}

export interface SearchFilter {
  field: string;
  operator: 'eq' | 'ne' | 'gt' | 'gte' | 'lt' | 'lte' | 'contains' | 'icontains' | 'in';
  value: unknown;
}

export interface BatchUpdateRequest {
  ids: string[];
  action: 'update' | 'archive' | 'delete';
  content?: string;
  metadata?: Record<string, unknown>;
}

export interface BatchDeleteRequest {
  userId?: string;
  orgId?: string;
  category?: string;
}

export interface Feedback {
  id: string;
  memoryId: string;
  type: FeedbackType;
  comment?: string;
  sessionId?: string;
  userId?: string;
  createdAt: string;
}

export interface MemoryHistory {
  id: string;
  memoryId: string;
  action: 'create' | 'update' | 'delete' | 'archive' | 'feedback';
  oldValue?: string;
  newValue?: string;
  changedBy?: string;
  reason?: string;
  metadata?: Record<string, unknown>;
  createdAt: string;
}

export interface Project {
  id: string;
  name: string;
  description?: string;
  userId?: string;
  orgId?: string;
  customInstructions?: string;
  settings: ProjectSettings;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface ProjectSettings {
  memoryTypes?: MemoryType[];
  categories?: string[];
  embeddingModel?: string;
  rerankingEnabled: boolean;
  conflictResolution: boolean;
  autoExpiration?: number;
  maxMemoriesPerUser?: number;
}

export interface Webhook {
  id: string;
  projectId: string;
  url: string;
  events: string[];
  secret?: string;
  active: boolean;
  metadata?: Record<string, unknown>;
  createdAt: string;
}

export interface MemoryAnalytics {
  totalMemories: number;
  activeMemories: number;
  archivedMemories: number;
  expiredMemories: number;
  byCategory: Record<string, number>;
  byType: Record<string, number>;
  byFeedbackScore: Record<string, number>;
  avgFeedbackScore: number;
  memoriesWithFeedback: number;
  totalFeedback: number;
  positiveFeedback: number;
  negativeFeedback: number;
}

export interface APIKey {
  id: string;
  key?: string;
  label: string;
  createdAt: string;
  expiresAt?: string;
  tenantId?: string;
}

export interface HealthStatus {
  status: 'ok' | 'ready';
  neo4j?: string;
  qdrant?: string;
}

export interface Skill {
  id: string;
  tenantId?: string;
  groupId?: string;
  name: string;
  domain: string;
  trigger: string;
  action: string;
  confidence: number;
  usageCount: number;
  sourceMemory?: string;
  createdBy?: string;
  verified: boolean;
  humanReviewed: boolean;
  version: number;
  tags?: string[];
  examples?: string[];
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  lastUsed?: string;
}

export interface SkillReview {
  id: string;
  tenantId?: string;
  skillId: string;
  status: ReviewStatus;
  reviewedBy?: string;
  notes?: string;
  decision?: string;
  createdAt: string;
  reviewedAt?: string;
}

export interface SkillSynthesis {
  id: string;
  tenantId?: string;
  groupId?: string;
  sourceSkillIds: string[];
  resultSkill: Skill;
  status: string;
  reason: string;
  createdAt: string;
}

export interface Agent {
  id: string;
  tenantId?: string;
  name: string;
  description?: string;
  config?: AgentConfig;
  status: AgentStatus;
  groups?: string[];
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  lastActive?: string;
}

export interface AgentConfig {
  maxMemories?: number;
  autoExtract?: boolean;
  sharingPolicy?: string;
  skillDomains?: string[];
}

export interface AgentGroup {
  id: string;
  tenantId?: string;
  name: string;
  description?: string;
  domain?: string;
  members: AgentMember[];
  policy?: GroupPolicy;
  memoryPoolId?: string;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface AgentMember {
  agentId: string;
  role: MemberRole;
  joinedAt: string;
}

export interface GroupPolicy {
  allowCrossAgentMemory?: boolean;
  requireHumanReview?: boolean;
  autoSyncEnabled?: boolean;
  syncIntervalSeconds?: number;
  maxSharedMemories?: number;
  skillSharingEnabled?: boolean;
}

export interface SharedMemory {
  id: string;
  groupId: string;
  memoryId: string;
  sharedBy: string;
  sharedAt: string;
  expiresAt?: string;
}

export interface CreateSkillOptions {
  name: string;
  trigger: string;
  action: string;
  domain?: string;
  confidence?: number;
  tags?: string[];
  examples?: string[];
  metadata?: Record<string, unknown>;
}

export interface CreateAgentOptions {
  name: string;
  description?: string;
  config?: AgentConfig;
  metadata?: Record<string, unknown>;
}

export interface CreateAgentGroupOptions {
  name: string;
  description?: string;
  domain?: string;
  policy?: GroupPolicy;
  metadata?: Record<string, unknown>;
}

export interface SuggestSkillsOptions {
  trigger: string;
  context?: string;
  limit?: number;
}

export interface SynthesizeSkillsOptions {
  skillIds: string[];
}

export interface ExtractSkillsOptions {
  content: string;
  userId?: string;
  agentId?: string;
}

export interface AddAgentToGroupOptions {
  agentId: string;
  role?: MemberRole;
}

export interface ProcessReviewOptions {
  approved: boolean;
  notes?: string;
}

export interface CreateMemoryOptions {
  content: string;
  memoryType?: MemoryType;
  userId?: string;
  orgId?: string;
  agentId?: string;
  sessionId?: string;
  category?: string;
  metadata?: Record<string, unknown>;
  immutable?: boolean;
  expirationDate?: Date;
  tags?: string[];
  importance?: ImportanceLevel;
}

export interface UpdateMemoryOptions {
  content: string;
  metadata?: Record<string, unknown>;
}

export interface CreateEntityOptions {
  name: string;
  entityType: string;
  properties?: Record<string, unknown>;
}

export interface CreateRelationOptions {
  fromId: string;
  toId: string;
  relationType: string;
  metadata?: Record<string, unknown>;
}

export interface SearchOptions {
  limit?: number;
  threshold?: number;
  userId?: string;
  orgId?: string;
  agentId?: string;
  category?: string;
  memoryType?: MemoryType;
  rerank?: boolean;
  rerankTopK?: number;
}

export interface ListMemoriesOptions {
  userId?: string;
  orgId?: string;
  agentId?: string;
  category?: string;
}

export interface BatchCreateMemoriesOptions {
  memories: CreateMemoryOptions[];
}

export interface BatchUpdateMemoriesOptions {
  memoryIds: string[];
  action: 'update' | 'archive' | 'delete';
  content?: string;
  metadata?: Record<string, unknown>;
}

export interface AddFeedbackOptions {
  memoryId: string;
  feedbackType: FeedbackType;
  comment?: string;
  userId?: string;
}

export interface CreateMemoryLinkOptions {
  fromId: string;
  toId: string;
  linkType: MemoryLinkType;
  weight?: number;
  metadata?: Record<string, unknown>;
}

export interface HybridSearchOptions {
  semanticLimit?: number;
  keywordLimit?: number;
  boost?: number;
  threshold?: number;
  filters?: SearchFilters;
  userId?: string;
  orgId?: string;
}

export interface CompactionOptions {
  userId?: string;
  orgId?: string;
  action?: 'full' | 'summarize' | 'archive' | 'delete';
}

export interface ExportMemoriesOptions {
  userId?: string;
  orgId?: string;
  agentId?: string;
  category?: string;
  format?: 'json' | 'jsonl';
}

export interface ImportMemoriesOptions {
  memories: CreateMemoryOptions[];
  relations?: CreateRelationOptions[];
}

export interface GetRelatedMemoriesOptions {
  linkType?: MemoryLinkType;
  limit?: number;
}

export class AgentMemoryError extends Error {
  constructor(
    message: string,
    public code?: string,
    public statusCode?: number
  ) {
    super(message);
    this.name = 'AgentMemoryError';
  }
}

export class AuthenticationError extends AgentMemoryError {
  constructor(message: string) {
    super(message, 'AUTHENTICATION_ERROR', 401);
    this.name = 'AuthenticationError';
  }
}

export class NotFoundError extends AgentMemoryError {
  constructor(message: string) {
    super(message, 'NOT_FOUND', 404);
    this.name = 'NotFoundError';
  }
}

export class ValidationError extends AgentMemoryError {
  constructor(message: string) {
    super(message, 'VALIDATION_ERROR', 400);
    this.name = 'ValidationError';
  }
}

export class RateLimitError extends AgentMemoryError {
  constructor(message: string) {
    super(message, 'RATE_LIMIT', 429);
    this.name = 'RateLimitError';
  }
}

export interface AgentMemoryConfig {
  baseUrl: string;
  apiKey?: string;
  timeout?: number;
}

export class AgentMemory {
  private baseUrl: string;
  private apiKey?: string;
  private timeout: number;

  constructor(config: AgentMemoryConfig) {
    this.baseUrl = config.baseUrl.replace(/\/$/, '');
    this.apiKey = config.apiKey;
    this.timeout = config.timeout ?? 30000;
  }

  private async request<T>(
    method: string,
    endpoint: string,
    options?: {
      params?: Record<string, unknown>;
      data?: unknown;
    }
  ): Promise<T> {
    let url = `${this.baseUrl}${endpoint}`;
    if (options?.params) {
      const searchParams = new URLSearchParams();
      for (const [key, value] of Object.entries(options.params)) {
        if (value !== undefined) {
          searchParams.append(key, String(value));
        }
      }
      const queryString = searchParams.toString();
      if (queryString) {
        url += `?${queryString}`;
      }
    }
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        method,
        headers,
        body: options?.data ? JSON.stringify(options.data) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeout);

      if (response.status === 401) {
        throw new AuthenticationError('Invalid or missing API key');
      }
      if (response.status === 403) {
        throw new AuthenticationError('Admin access required');
      }
      if (response.status === 404) {
        throw new NotFoundError(`Resource not found: ${endpoint}`);
      }
      if (response.status === 429) {
        throw new RateLimitError('Rate limit exceeded');
      }
      if (response.status === 400) {
        const text = await response.text();
        throw new ValidationError(text);
      }

      if (!response.ok) {
        throw new AgentMemoryError(
          `Request failed: ${response.statusText}`,
          'REQUEST_ERROR',
          response.status
        );
      }

      return response.json() as T;
    } catch (error) {
      clearTimeout(timeout);
      if (error instanceof AgentMemoryError) {
        throw error;
      }
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new AgentMemoryError('Request timeout', 'TIMEOUT');
        }
        throw new AgentMemoryError(error.message, 'REQUEST_ERROR');
      }
      throw new AgentMemoryError('Unknown error', 'UNKNOWN');
    }
  }

  // ==================== Health ====================

  async health(): Promise<{ status: string }> {
    return this.request<{ status: string }>('GET', '/health');
  }

  async ready(): Promise<HealthStatus> {
    return this.request<HealthStatus>('GET', '/ready');
  }

  // ==================== Sessions ====================

  sessions = {
    create: async (agentId: string, metadata?: Record<string, unknown>): Promise<Session> => {
      return this.request<Session>('POST', '/sessions', {
        data: { agent_id: agentId, metadata },
      });
    },

    get: async (sessionId: string): Promise<Session & { messages: Message[] }> => {
      return this.request<Session & { messages: Message[] }>('GET', `/sessions/${sessionId}`);
    },

    delete: async (sessionId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/sessions/${sessionId}`);
    },

    messages: {
      add: async (
        sessionId: string,
        role: 'user' | 'assistant' | 'system' | 'tool',
        content: string
      ): Promise<{ status: string }> => {
        return this.request<{ status: string }>('POST', `/sessions/${sessionId}/messages`, {
          data: { role, content },
        });
      },

      list: async (sessionId: string, limit = 50): Promise<Message[]> => {
        return this.request<Message[]>('GET', `/sessions/${sessionId}/messages`, {
          params: { limit },
        });
      },
    },
  };

  // ==================== Memories ====================

  memories = {
    create: async (options: CreateMemoryOptions): Promise<Memory> => {
      const data: Record<string, unknown> = {
        content: options.content,
        type: options.memoryType ?? 'user',
      };

      if (options.userId) data.user_id = options.userId;
      if (options.orgId) data.org_id = options.orgId;
      if (options.agentId) data.agent_id = options.agentId;
      if (options.sessionId) data.session_id = options.sessionId;
      if (options.category) data.category = options.category;
      if (options.metadata) data.metadata = options.metadata;
      if (options.immutable) data.immutable = options.immutable;
      if (options.expirationDate) {
        data.expiration_date = options.expirationDate.toISOString();
      }
      if (options.tags) data.tags = options.tags;
      if (options.importance) data.importance = options.importance;

      return this.request<Memory>('POST', '/memories', { data });
    },

    get: async (memoryId: string): Promise<Memory> => {
      return this.request<Memory>('GET', `/memories/${memoryId}`);
    },

    update: async (memoryId: string, options: UpdateMemoryOptions): Promise<Memory> => {
      const data: Record<string, unknown> = {
        content: options.content,
      };
      if (options.metadata) data.metadata = options.metadata;

      return this.request<Memory>('PUT', `/memories/${memoryId}`, { data });
    },

    delete: async (memoryId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/memories/${memoryId}`);
    },

    list: async (options?: ListMemoriesOptions): Promise<{ memories: Memory[]; count: number }> => {
      const params: Record<string, unknown> = {};
      if (options?.userId) params.user_id = options.userId;
      if (options?.orgId) params.org_id = options.orgId;
      if (options?.agentId) params.agent_id = options.agentId;
      if (options?.category) params.category = options.category;

      return this.request<{ memories: Memory[]; count: number }>('GET', '/memories', { params });
    },

    search: async (query: string, options?: SearchOptions): Promise<MemoryResult[]> => {
      const params: Record<string, unknown> = { q: query };
      if (options?.limit) params.limit = options.limit;
      if (options?.threshold) params.threshold = options.threshold;
      if (options?.userId) params.user_id = options.userId;
      if (options?.orgId) params.org_id = options.orgId;
      if (options?.agentId) params.agent_id = options.agentId;
      if (options?.category) params.category = options.category;
      if (options?.memoryType) params.memory_type = options.memoryType;
      if (options?.rerank) params.rerank = options.rerank;
      if (options?.rerankTopK) params.rerank_top_k = options.rerankTopK;

      return this.request<MemoryResult[]>('GET', '/search', { params });
    },

    advancedSearch: async (filters: SearchFilters, limit = 100): Promise<MemoryResult[]> => {
      return this.request<MemoryResult[]>('POST', '/search/advanced', {
        data: { filters, limit },
      });
    },

    history: async (memoryId: string): Promise<MemoryHistory[]> => {
      return this.request<MemoryHistory[]>('GET', `/memories/${memoryId}/history`);
    },

    setExpiration: async (memoryId: string, expirationDate: Date): Promise<{ status: string }> => {
      return this.request<{ status: string }>('POST', `/memories/${memoryId}/expire`, {
        data: { expiration_date: expirationDate.toISOString() },
      });
    },

    linkToEntity: async (memoryId: string, entityId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('POST', `/memories/${memoryId}/link/${entityId}`);
    },

    batch: {
      create: async (memories: CreateMemoryOptions[]): Promise<{ created: Memory[]; count: number }> => {
        return this.request<{ created: Memory[]; count: number }>('POST', '/memories/batch', {
          data: { memories },
        });
      },

      update: async (options: BatchUpdateMemoriesOptions): Promise<{ status: string }> => {
        const data: Record<string, unknown> = {
          ids: options.memoryIds,
          action: options.action,
        };
        if (options.content) data.content = options.content;
        if (options.metadata) data.metadata = options.metadata;

        return this.request<{ status: string }>('PUT', '/memories/batch-update', { data });
      },

      delete: async (memoryIds: string[]): Promise<{ status: string; count: string }> => {
        return this.request<{ status: string; count: string }>('DELETE', '/memories/batch-delete', {
          data: { ids: memoryIds },
        });
      },
    },

    bulkDelete: async (options: BatchDeleteRequest): Promise<{ status: string; count: number }> => {
      return this.request<{ status: string; count: number }>('DELETE', '/memories/bulk-delete', {
        data: options,
      });
    },

    createLink: async (options: CreateMemoryLinkOptions): Promise<MemoryLink> => {
      const data: Record<string, unknown> = {
        from_id: options.fromId,
        to_id: options.toId,
        type: options.linkType,
        weight: options.weight ?? 0.5,
      };
      if (options.metadata) data.metadata = options.metadata;

      return this.request<MemoryLink>('POST', '/memories/links', { data });
    },

    getLinks: async (memoryId: string): Promise<MemoryLink[]> => {
      return this.request<MemoryLink[]>('GET', `/memories/${memoryId}/links`);
    },

    deleteLink: async (linkId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/memories/links/${linkId}`);
    },

    getVersions: async (memoryId: string): Promise<MemoryVersion[]> => {
      return this.request<MemoryVersion[]>('GET', `/memories/${memoryId}/versions`);
    },

    restoreVersion: async (memoryId: string, versionId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('POST', `/memories/${memoryId}/restore`, {
        data: { version_id: versionId },
      });
    },

    getStats: async (userId?: string, orgId?: string): Promise<MemoryStats> => {
      const params: Record<string, unknown> = {};
      if (userId) params.user_id = userId;
      if (orgId) params.org_id = orgId;

      return this.request<MemoryStats>('GET', '/memories/stats', { params });
    },

    getInsights: async (userId?: string, orgId?: string): Promise<MemoryInsights[]> => {
      const params: Record<string, unknown> = {};
      if (userId) params.user_id = userId;
      if (orgId) params.org_id = orgId;

      return this.request<MemoryInsights[]>('GET', '/memories/insights', { params });
    },

    getSummary: async (userId?: string, orgId?: string): Promise<MemorySummary> => {
      const params: Record<string, unknown> = {};
      if (userId) params.user_id = userId;
      if (orgId) params.org_id = orgId;

      return this.request<MemorySummary>('GET', '/memories/summary', { params });
    },

    export: async (options?: ExportMemoriesOptions): Promise<string> => {
      const params: Record<string, unknown> = {};
      if (options?.userId) params.user_id = options.userId;
      if (options?.orgId) params.org_id = options.orgId;
      if (options?.agentId) params.agent_id = options.agentId;
      if (options?.category) params.category = options.category;
      if (options?.format) params.format = options.format;

      return this.request<string>('GET', '/memories/export', { params });
    },

    import: async (options: ImportMemoriesOptions): Promise<{ imported: number; failed: number }> => {
      const data: Record<string, unknown> = {
        memories: options.memories,
      };
      if (options.relations) data.relations = options.relations;

      return this.request<{ imported: number; failed: number }>('POST', '/memories/import', { data });
    },

    hybridSearch: async (query: string, options?: HybridSearchOptions): Promise<MemoryResult[]> => {
      const data: Record<string, unknown> = {
        query,
        semantic_limit: options?.semanticLimit ?? 10,
        keyword_limit: options?.keywordLimit ?? 10,
        boost: options?.boost ?? 1.5,
        threshold: options?.threshold ?? 0.6,
      };
      if (options?.userId) data.user_id = options.userId;
      if (options?.orgId) data.org_id = options.orgId;
      if (options?.filters) data.filters = options.filters;

      return this.request<MemoryResult[]>('POST', '/search/hybrid', { data });
    },

    runCompaction: async (options?: CompactionOptions): Promise<{ status: string; memoriesProcessed?: number; tokensSaved?: number }> => {
      const data: Record<string, unknown> = {
        action: options?.action ?? 'full',
      };
      if (options?.userId) data.user_id = options.userId;
      if (options?.orgId) data.org_id = options.orgId;

      return this.request<{ status: string; memoriesProcessed?: number; tokensSaved?: number }>('POST', '/compact', { data });
    },

    getCompactionStatus: async (): Promise<CompactionStatus> => {
      return this.request<CompactionStatus>('GET', '/compact/status');
    },
  };

  // ==================== Feedback ====================

  feedback = {
    add: async (options: AddFeedbackOptions): Promise<Feedback> => {
      const data: Record<string, unknown> = {
        memory_id: options.memoryId,
        type: options.feedbackType,
      };
      if (options.comment) data.comment = options.comment;
      if (options.userId) data.user_id = options.userId;

      return this.request<Feedback>('POST', '/feedback', { data });
    },

    getByType: async (
      feedbackType: FeedbackType,
      limit = 50
    ): Promise<Memory[]> => {
      return this.request<Memory[]>('GET', '/feedback/memories', {
        params: { type: feedbackType, limit },
      });
    },
  };

  // ==================== Entities ====================

  entities = {
    create: async (options: CreateEntityOptions): Promise<Entity> => {
      const data: Record<string, unknown> = {
        name: options.name,
        type: options.entityType,
      };
      if (options.properties) data.properties = options.properties;

      return this.request<Entity>('POST', '/entities', { data });
    },

    get: async (entityId: string): Promise<Entity> => {
      return this.request<Entity>('GET', `/entities/${entityId}`);
    },

    list: async (entityType?: string, limit = 100): Promise<{ entities: Entity[]; count: number }> => {
      const params: Record<string, unknown> = { limit };
      if (entityType) params.entity_type = entityType;

      return this.request<{ entities: Entity[]; count: number }>('GET', '/entities', { params });
    },

    update: async (
      entityId: string,
      updates: Partial<CreateEntityOptions>
    ): Promise<Entity> => {
      return this.request<Entity>('PUT', `/entities/${entityId}`, { data: updates });
    },

    delete: async (entityId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/entities/${entityId}`);
    },

    getMemories: async (entityId: string, limit = 50): Promise<MemoryResult[]> => {
      return this.request<MemoryResult[]>('GET', `/entities/${entityId}/memories`, {
        params: { limit },
      });
    },

    relations: {
      get: async (entityId: string, relationType?: string): Promise<Relation[]> => {
        const params: Record<string, unknown> = {};
        if (relationType) params.type = relationType;

        return this.request<Relation[]>('GET', `/entities/${entityId}/relations`, { params });
      },
    },
  };

  // ==================== Relations ====================

  relations = {
    create: async (options: CreateRelationOptions): Promise<{ status: string }> => {
      return this.request<{ status: string }>('POST', '/relations', { data: options });
    },

    delete: async (relationId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/relations/${relationId}`);
    },
  };

  // ==================== Graph ====================

  graph = {
    query: async (
      cypher: string,
      params?: Record<string, unknown>
    ): Promise<Record<string, unknown>[]> => {
      return this.request<Record<string, unknown>[]>('POST', '/graph/query', {
        data: { cypher, params },
      });
    },

    traverse: async (entityId: string, depth = 3): Promise<{ nodes: Entity[]; edges: Relation[] }> => {
      return this.request<{ nodes: Entity[]; edges: Relation[] }>('GET', `/graph/traverse/${entityId}`, {
        params: { depth },
      });
    },
  };

  // ==================== Projects ====================

  projects = {
    create: async (project: Partial<Project>): Promise<Project> => {
      return this.request<Project>('POST', '/projects', { data: project });
    },

    get: async (projectId: string): Promise<Project> => {
      return this.request<Project>('GET', `/projects/${projectId}`);
    },

    update: async (projectId: string, updates: Partial<Project>): Promise<Project> => {
      return this.request<Project>('PUT', `/projects/${projectId}`, { data: updates });
    },

    delete: async (projectId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/projects/${projectId}`);
    },

    list: async (userId?: string, orgId?: string): Promise<Project[]> => {
      const params: Record<string, unknown> = {};
      if (userId) params.user_id = userId;
      if (orgId) params.org_id = orgId;

      return this.request<Project[]>('GET', '/projects', { params });
    },
  };

  // ==================== Webhooks ====================

  webhooks = {
    create: async (webhook: Partial<Webhook>): Promise<Webhook> => {
      return this.request<Webhook>('POST', '/webhooks', { data: webhook });
    },

    get: async (webhookId: string): Promise<Webhook> => {
      return this.request<Webhook>('GET', `/webhooks/${webhookId}`);
    },

    update: async (webhookId: string, updates: Partial<Webhook>): Promise<Webhook> => {
      return this.request<Webhook>('PUT', `/webhooks/${webhookId}`, { data: updates });
    },

    delete: async (webhookId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/webhooks/${webhookId}`);
    },

    list: async (projectId?: string): Promise<Webhook[]> => {
      const params: Record<string, unknown> = {};
      if (projectId) params.project_id = projectId;

      return this.request<Webhook[]>('GET', '/webhooks', { params });
    },

    test: async (webhookId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('POST', `/webhooks/${webhookId}/test`);
    },
  };

  // ==================== Admin ====================

  admin = {
    cleanup: async (): Promise<{ cleaned_up: number }> => {
      return this.request<{ cleaned_up: number }>('POST', '/admin/cleanup');
    },

    sync: async (entityIds?: string[]): Promise<{ status: string }> => {
      return this.request<{ status: string }>('POST', '/admin/sync', {
        data: entityIds ? { entity_ids: entityIds } : {},
      });
    },

    analytics: async (): Promise<MemoryAnalytics> => {
      return this.request<MemoryAnalytics>('GET', '/admin/analytics');
    },

    apiKeys: {
      list: async (): Promise<APIKey[]> => {
        return this.request<APIKey[]>('GET', '/admin/api-keys');
      },

      create: async (
        label: string,
        expiresInHours = 0,
        tenantId?: string
      ): Promise<APIKey> => {
        const data: Record<string, unknown> = { label, expires_in_hours: expiresInHours };
        if (tenantId) data.tenant_id = tenantId;

        return this.request<APIKey>('POST', '/admin/api-keys', { data });
      },

      delete: async (keyId: string): Promise<{ status: string }> => {
        return this.request<{ status: string }>('DELETE', `/admin/api-keys/${keyId}`);
      },
    },
  };

  // ==================== Skills ====================

  skills = {
    create: async (options: CreateSkillOptions): Promise<Skill> => {
      const data: Record<string, unknown> = {
        name: options.name,
        trigger: options.trigger,
        action: options.action,
        domain: options.domain ?? 'general',
        confidence: options.confidence ?? 0.8,
      };
      if (options.tags) data.tags = options.tags;
      if (options.examples) data.examples = options.examples;
      if (options.metadata) data.metadata = options.metadata;

      return this.request<Skill>('POST', '/skills', { data });
    },

    get: async (skillId: string): Promise<Skill> => {
      return this.request<Skill>('GET', `/skills/${skillId}`);
    },

    list: async (domain?: string, limit = 50): Promise<{ skills: Skill[]; count: number }> => {
      const params: Record<string, unknown> = { limit };
      if (domain) params.domain = domain;

      return this.request<{ skills: Skill[]; count: number }>('GET', '/skills', { params });
    },

    search: async (trigger?: string, domain?: string, limit = 20): Promise<{ skills: Skill[]; count: number }> => {
      const params: Record<string, unknown> = { limit };
      if (trigger) params.trigger = trigger;
      if (domain) params.domain = domain;

      return this.request<{ skills: Skill[]; count: number }>('GET', '/skills/search', { params });
    },

    update: async (skillId: string, updates: Partial<CreateSkillOptions>): Promise<Skill> => {
      return this.request<Skill>('PUT', `/skills/${skillId}`, { data: updates });
    },

    delete: async (skillId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/skills/${skillId}`);
    },

    use: async (skillId: string): Promise<{ success: boolean }> => {
      return this.request<{ success: boolean }>('POST', `/skills/${skillId}/use`);
    },

    suggest: async (options: SuggestSkillsOptions): Promise<{ suggestions: Skill[] }> => {
      const data: Record<string, unknown> = {
        trigger: options.trigger,
        limit: options.limit ?? 5,
      };
      if (options.context) data.context = options.context;

      return this.request<{ suggestions: Skill[] }>('POST', '/skills/suggest', { data });
    },

    synthesize: async (options: SynthesizeSkillsOptions): Promise<SkillSynthesis> => {
      return this.request<SkillSynthesis>('POST', '/skills/synthesize', {
        data: { skill_ids: options.skillIds },
      });
    },

    extract: async (options: ExtractSkillsOptions): Promise<{ skills: Skill[] }> => {
      const data: Record<string, unknown> = { content: options.content };
      if (options.userId) data.user_id = options.userId;
      if (options.agentId) data.agent_id = options.agentId;

      return this.request<{ skills: Skill[] }>('POST', '/skills/extract', { data });
    },
  };

  // ==================== Agents ====================

  agents = {
    create: async (options: CreateAgentOptions): Promise<Agent> => {
      const data: Record<string, unknown> = { name: options.name };
      if (options.description) data.description = options.description;
      if (options.config) data.config = options.config;
      if (options.metadata) data.metadata = options.metadata;

      return this.request<Agent>('POST', '/agents', { data });
    },

    get: async (agentId: string): Promise<Agent> => {
      return this.request<Agent>('GET', `/agents/${agentId}`);
    },

    list: async (limit = 50): Promise<{ agents: Agent[]; total: number }> => {
      return this.request<{ agents: Agent[]; total: number }>('GET', '/agents', {
        params: { limit },
      });
    },

    update: async (agentId: string, updates: Partial<CreateAgentOptions>): Promise<Agent> => {
      return this.request<Agent>('PUT', `/agents/${agentId}`, { data: updates });
    },

    delete: async (agentId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/agents/${agentId}`);
    },
  };

  // ==================== Agent Groups ====================

  groups = {
    create: async (options: CreateAgentGroupOptions): Promise<AgentGroup> => {
      const data: Record<string, unknown> = { name: options.name };
      if (options.description) data.description = options.description;
      if (options.domain) data.domain = options.domain;
      if (options.policy) data.policy = options.policy;
      if (options.metadata) data.metadata = options.metadata;

      return this.request<AgentGroup>('POST', '/groups', { data });
    },

    get: async (groupId: string): Promise<AgentGroup> => {
      return this.request<AgentGroup>('GET', `/groups/${groupId}`);
    },

    list: async (limit = 50): Promise<{ groups: AgentGroup[]; total: number }> => {
      return this.request<{ groups: AgentGroup[]; total: number }>('GET', '/groups', {
        params: { limit },
      });
    },

    update: async (groupId: string, updates: Partial<CreateAgentGroupOptions>): Promise<AgentGroup> => {
      return this.request<AgentGroup>('PUT', `/groups/${groupId}`, { data: updates });
    },

    delete: async (groupId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/groups/${groupId}`);
    },

    addMember: async (groupId: string, options: AddAgentToGroupOptions): Promise<{ success: boolean }> => {
      const data: Record<string, unknown> = {
        agent_id: options.agentId,
        role: options.role ?? 'contributor',
      };
      return this.request<{ success: boolean }>('POST', `/groups/${groupId}/members`, { data });
    },

    removeMember: async (groupId: string, agentId: string): Promise<{ success: boolean }> => {
      return this.request<{ success: boolean }>('DELETE', `/groups/${groupId}/members/${agentId}`);
    },

    getSkills: async (groupId: string, limit = 50): Promise<{ skills: Skill[]; count: number }> => {
      return this.request<{ skills: Skill[]; count: number }>('GET', `/groups/${groupId}/skills`, {
        params: { limit },
      });
    },

    getMemories: async (groupId: string): Promise<{ memories: Memory[]; count: number }> => {
      return this.request<{ memories: Memory[]; count: number }>('GET', `/groups/${groupId}/memories`);
    },

    shareMemory: async (groupId: string, memoryId: string): Promise<{ success: boolean }> => {
      return this.request<{ success: boolean }>('POST', `/groups/${groupId}/memories`, {
        data: { memory_id: memoryId },
      });
    },
  };

  // ==================== Reviews ====================

  reviews = {
    listPending: async (): Promise<{ reviews: SkillReview[]; count: number }> => {
      return this.request<{ reviews: SkillReview[]; count: number }>('GET', '/reviews');
    },

    get: async (reviewId: string): Promise<SkillReview> => {
      return this.request<SkillReview>('GET', `/reviews/${reviewId}`);
    },

    process: async (reviewId: string, options: ProcessReviewOptions): Promise<{ success: boolean }> => {
      const data: Record<string, unknown> = { approved: options.approved };
      if (options.notes) data.notes = options.notes;

      return this.request<{ success: boolean }>('POST', `/reviews/${reviewId}`, { data });
    },
  };
}

export default AgentMemory;
