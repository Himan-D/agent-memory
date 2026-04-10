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
    const url = `${this.baseUrl}${endpoint}`;
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
        params: options?.params,
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

      return response.json();
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

  async sessions = {
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

  async memories = {
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
  };

  // ==================== Feedback ====================

  async feedback = {
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

  async entities = {
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

  async relations = {
    create: async (options: CreateRelationOptions): Promise<{ status: string }> => {
      return this.request<{ status: string }>('POST', '/relations', { data: options });
    },

    delete: async (relationId: string): Promise<{ status: string }> => {
      return this.request<{ status: string }>('DELETE', `/relations/${relationId}`);
    },
  };

  // ==================== Graph ====================

  async graph = {
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

  async projects = {
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

  async webhooks = {
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

  async admin = {
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
}

export default AgentMemory;
