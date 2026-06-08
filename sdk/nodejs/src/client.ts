/**
 * Hystersis Async Client
 * 
 * Persistent memory infrastructure for AI agents.
 */

import {
  HystersisConfig, RetryConfig, RateLimitConfig, TimeoutConfig,
  RequestInterceptor, ResponseInterceptor, Source, SourceIngestRequest,
  SourceIngestResult, SourceListResponse, SourceUploadOptions,
  MemoryEvent, V3AddMemoryOptions, V3AddMemoryResponse, V3ListOptions,
  V3SearchOptions
} from './types';
import {
  HystersisError, AuthenticationError, NotFoundError, ValidationError, RateLimitError, ServerError
} from './errors';

// ==================== Default Configs ====================

const DEFAULT_TIMEOUT: TimeoutConfig = {
  connect: 10,
  read: 30,
  write: 30,
  pool: 5,
};

const DEFAULT_RETRY: RetryConfig = {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 60000,
  exponentialBase: 2,
  retryOnStatusCodes: [429, 500, 502, 503, 504],
};

const DEFAULT_RATE_LIMIT: RateLimitConfig = {
  requestsPerSecond: 10,
  burstSize: 20,
};

// ==================== Rate Limiter ====================

class TokenBucketRateLimiter {
  private tokens: number;
  private lastUpdate: number;
  private readonly rate: number;
  private readonly burst: number;
  private mutex: Promise<void> = Promise.resolve();

  constructor(requestsPerSecond: number, burstSize: number) {
    this.rate = requestsPerSecond;
    this.burst = burstSize;
    this.tokens = burstSize;
    this.lastUpdate = Date.now();
  }

  async acquire(): Promise<void> {
    const prev = this.mutex;
    let release: () => void;
    this.mutex = new Promise<void>(r => release = r);
    
    await prev;
    
    try {
      const now = Date.now();
      const elapsed = (now - this.lastUpdate) / 1000;
      this.tokens = Math.min(this.burst, this.tokens + elapsed * this.rate);
      this.lastUpdate = now;

      if (this.tokens < 1) {
        const waitTime = (1 - this.tokens) / this.rate;
        await this.sleep(waitTime * 1000);
        this.tokens = 0;
      } else {
        this.tokens -= 1;
      }
    } finally {
      release!();
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// ==================== Main Client ====================

export class HystersisClient {
  private baseUrl: string;
  private apiKey?: string;
  private timeout: TimeoutConfig;
  private retry: RetryConfig;
  private rateLimiter: TokenBucketRateLimiter;
  private requestInterceptors: RequestInterceptor[] = [];
  private responseInterceptors: ResponseInterceptor[] = [];
  private maxConnections: number;
  private maxKeepaliveConnections: number;
  private closed = false;

  constructor(config: HystersisConfig) {
    this.baseUrl = config.baseUrl.replace(/\/$/, '');
    this.apiKey = config.apiKey;
    this.timeout = config.timeout ?? DEFAULT_TIMEOUT;
    this.retry = config.retry ?? DEFAULT_RETRY;
    this.rateLimiter = new TokenBucketRateLimiter(
      config.rateLimit?.requestsPerSecond ?? DEFAULT_RATE_LIMIT.requestsPerSecond,
      config.rateLimit?.burstSize ?? DEFAULT_RATE_LIMIT.burstSize
    );
    this.maxConnections = config.maxConnections ?? 100;
    this.maxKeepaliveConnections = config.maxKeepaliveConnections ?? 20;
  }

  // ==================== Interceptors ====================

  addRequestInterceptor(interceptor: RequestInterceptor): void {
    this.requestInterceptors.push(interceptor);
  }

  addResponseInterceptor(interceptor: ResponseInterceptor): void {
    this.responseInterceptors.push(interceptor);
  }

  // ==================== Request Building ====================

  private buildRequest(
    method: string,
    endpoint: string,
    options?: { params?: Record<string, unknown>; data?: unknown }
  ): Request {
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

    return new Request(url, {
      method,
      headers,
      body: options?.data ? JSON.stringify(options.data) : undefined,
    });
  }

  private buildMultipartRequest(endpoint: string, form: FormData): Request {
    const headers: Record<string, string> = {};
    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }
    return new Request(`${this.baseUrl}${endpoint}`, {
      method: 'POST',
      headers,
      body: form,
    });
  }

  // ==================== Request Sending ====================

  private async sendRequest<T>(
    request: Request,
    retryCount = 0
  ): Promise<T> {
    // Apply request interceptors
    let finalRequest = request;
    for (const interceptor of this.requestInterceptors) {
      finalRequest = await interceptor(finalRequest);
    }

    try {
      const response = await fetch(finalRequest, {
        signal: AbortSignal.timeout(this.timeout.read * 1000),
      });

      // Apply response interceptors
      let finalResponse = response;
      for (const interceptor of this.responseInterceptors) {
        finalResponse = await interceptor(finalResponse, finalRequest);
      }

      // Handle errors
      if (finalResponse.status === 401) {
        throw new AuthenticationError('Invalid or missing API key', 401, finalResponse);
      }
      if (finalResponse.status === 403) {
        throw new AuthenticationError('Forbidden: Admin access required', 403, finalResponse);
      }
      if (finalResponse.status === 404) {
        throw new NotFoundError(`Resource not found: ${finalRequest.url}`, 404, finalResponse);
      }
      if (finalResponse.status === 429) {
        if (retryCount < this.retry.maxRetries) {
          const delay = this.calculateRetryDelay(retryCount);
          await this.sleep(delay);
          return this.sendRequest<T>(request, retryCount + 1);
        }
        throw new RateLimitError('Rate limit exceeded', 429, finalResponse);
      }
      if (finalResponse.status >= 500 && this.retry.retryOnStatusCodes.includes(finalResponse.status)) {
        if (retryCount < this.retry.maxRetries) {
          const delay = this.calculateRetryDelay(retryCount);
          await this.sleep(delay);
          return this.sendRequest<T>(request, retryCount + 1);
        }
        throw new ServerError(`Server error: ${finalResponse.status}`, finalResponse.status, finalResponse);
      }
      if (finalResponse.status === 400) {
        const text = await finalResponse.text();
        throw new ValidationError(text, 400, finalResponse);
      }

      if (!finalResponse.ok) {
        throw new HystersisError(
          `Request failed: ${finalResponse.statusText}`,
          'REQUEST_ERROR',
          finalResponse.status,
          finalResponse
        );
      }

      return finalResponse.json() as Promise<T>;
    } catch (error) {
      if (error instanceof HystersisError) {
        throw error;
      }
      if (error instanceof Error) {
        if (error.name === 'TimeoutError' || error.name === 'AbortError') {
          if (retryCount < this.retry.maxRetries) {
            const delay = this.calculateRetryDelay(retryCount);
            await this.sleep(delay);
            return this.sendRequest<T>(request, retryCount + 1);
          }
          throw new HystersisError('Request timeout', 'TIMEOUT');
        }
        throw new HystersisError(error.message, 'REQUEST_ERROR');
      }
      throw new HystersisError('Unknown error', 'UNKNOWN');
    }
  }

  private calculateRetryDelay(attempt: number): number {
    const delay = this.retry.baseDelay * Math.pow(this.retry.exponentialBase, attempt);
    return Math.min(delay, this.retry.maxDelay);
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // ==================== Public API ====================

  async request<T>(
    method: string,
    endpoint: string,
    options?: { params?: Record<string, unknown>; data?: unknown }
  ): Promise<T> {
    if (this.closed) {
      throw new HystersisError('Client is closed');
    }
    
    await this.rateLimiter.acquire();
    const request = this.buildRequest(method, endpoint, options);
    return this.sendRequest<T>(request);
  }

  async close(): Promise<void> {
    this.closed = true;
  }

  // ==================== Health ====================

  async health(): Promise<{ status: string }> {
    return this.request<{ status: string }>('GET', '/health');
  }

  async ready(): Promise<{ status: string; neo4j?: string; qdrant?: string }> {
    return this.request<{ status: string; neo4j?: string; qdrant?: string }>('GET', '/ready');
  }

  // ==================== Sessions ====================

  async createSession(agentId: string, metadata?: Record<string, unknown>): Promise<{ id: string; agent_id: string; created_at: string }> {
    return this.request<{ id: string; agent_id: string; created_at: string }>('POST', '/sessions', {
      data: { agent_id: agentId, metadata },
    });
  }

  async getSession(sessionId: string): Promise<{ id: string; agent_id: string; messages: any[]; created_at: string }> {
    return this.request<{ id: string; agent_id: string; messages: any[]; created_at: string }>('GET', `/sessions/${sessionId}`);
  }

  async deleteSession(sessionId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/sessions/${sessionId}`);
  }

  async listSessions(agentId?: string, limit = 50, offset = 0): Promise<{ sessions: any[]; count: number }> {
    const params: Record<string, unknown> = { limit, offset };
    if (agentId) params.agent_id = agentId;
    return this.request<{ sessions: any[]; count: number }>('GET', '/sessions', { params });
  }

  async addMessage(sessionId: string, role: string, content: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('POST', `/sessions/${sessionId}/messages`, {
      data: { role, content },
    });
  }

  async getMessages(sessionId: string, limit = 50): Promise<any[]> {
    return this.request<any[]>('GET', `/sessions/${sessionId}/messages`, {
      params: { limit },
    });
  }

  // ==================== Memories ====================

  async createMemory(options: {
    content: string;
    type?: string;
    user_id?: string;
    org_id?: string;
    agent_id?: string;
    session_id?: string;
    category?: string;
    metadata?: Record<string, unknown>;
    immutable?: boolean;
    expiration_date?: string;
    tags?: string[];
    importance?: string;
  }): Promise<any> {
    return this.request<any>('POST', '/memories', { data: options });
  }

  async getMemory(memoryId: string): Promise<any> {
    return this.request<any>('GET', `/memories/${memoryId}`);
  }

  async updateMemory(memoryId: string, content: string, metadata?: Record<string, unknown>): Promise<any> {
    const data: Record<string, unknown> = { content };
    if (metadata) data.metadata = metadata;
    return this.request<any>('PUT', `/memories/${memoryId}`, { data });
  }

  async deleteMemory(memoryId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/memories/${memoryId}`);
  }

  async listMemories(options?: {
    user_id?: string;
    org_id?: string;
    agent_id?: string;
    category?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ memories: any[]; count: number }> {
    const params: Record<string, unknown> = { limit: options?.limit ?? 100, offset: options?.offset ?? 0 };
    if (options?.user_id) params.user_id = options.user_id;
    if (options?.org_id) params.org_id = options.org_id;
    if (options?.agent_id) params.agent_id = options.agent_id;
    if (options?.category) params.category = options.category;
    return this.request<{ memories: any[]; count: number }>('GET', '/memories', { params });
  }

  async search(query: string, options?: {
    limit?: number;
    threshold?: number;
    user_id?: string;
    org_id?: string;
    agent_id?: string;
    category?: string;
    memory_type?: string;
    rerank?: boolean;
    mode?: string;
  }): Promise<any[]> {
    const params: Record<string, unknown> = { q: query, limit: options?.limit ?? 10 };
    if (options?.threshold) params.threshold = options.threshold;
    if (options?.user_id) params.user_id = options.user_id;
    if (options?.org_id) params.org_id = options.org_id;
    if (options?.agent_id) params.agent_id = options.agent_id;
    if (options?.category) params.category = options.category;
    if (options?.memory_type) params.memory_type = options.memory_type;
    if (options?.rerank) params.rerank = options.rerank;
    if (options?.mode) params.mode = options.mode;
    return this.request<any[]>('GET', '/search', { params });
  }

  async getMemoryHistory(memoryId: string): Promise<any[]> {
    return this.request<any[]>('GET', `/memories/${memoryId}/history`);
  }

  async setMemoryExpiration(memoryId: string, expirationDate: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('POST', `/memories/${memoryId}/expire`, {
      data: { expiration_date: expirationDate },
    });
  }

  async linkMemoryToEntity(memoryId: string, entityId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('POST', `/memories/${memoryId}/link/${entityId}`);
  }

  async batchCreateMemories(memories: any[]): Promise<{ created: any[]; count: number }> {
    return this.request<{ created: any[]; count: number }>('POST', '/memories/batch', {
      data: { memories },
    });
  }

  async batchUpdateMemories(memoryIds: string[], action: string, content?: string, metadata?: Record<string, unknown>): Promise<{ status: string }> {
    const data: Record<string, unknown> = { ids: memoryIds, action };
    if (content) data.content = content;
    if (metadata) data.metadata = metadata;
    return this.request<{ status: string }>('PUT', '/memories/batch-update', { data });
  }

  async bulkDelete(options: { user_id?: string; org_id?: string; category?: string }): Promise<{ status: string; count: number }> {
    return this.request<{ status: string; count: number }>('DELETE', '/memories/bulk-delete', { data: options });
  }

  async v3AddMemory(options: V3AddMemoryOptions): Promise<V3AddMemoryResponse> {
    return this.request<V3AddMemoryResponse>('POST', '/v3/memories/add', { data: options });
  }

  async v3SearchMemories(options: V3SearchOptions): Promise<{ results: any[]; count: number; query: string; mode: string }> {
    return this.request<{ results: any[]; count: number; query: string; mode: string }>('POST', '/v3/memories/search', { data: options });
  }

  async v3ListMemories(options: V3ListOptions): Promise<{ count: number; next: string | null; previous: string | null; results: any[] }> {
    return this.request<{ count: number; next: string | null; previous: string | null; results: any[] }>('POST', '/v3/memories', { data: options });
  }

  async getEventStatus(eventId: string): Promise<MemoryEvent> {
    return this.request<MemoryEvent>('GET', `/events/${eventId}`);
  }

  async exportMemories(options: { user_id?: string; org_id?: string; format?: 'json' | 'jsonl' }): Promise<any> {
    return this.request<any>('POST', '/exports', { data: options });
  }

  async importMemories(options: { memories?: any[]; entities?: any[]; relations?: any[] }): Promise<{ event_id: string; status: string; imported: number }> {
    return this.request<{ event_id: string; status: string; imported: number }>('POST', '/imports', { data: options });
  }

  // ==================== Sources ====================

  async ingestSource(options: SourceIngestRequest): Promise<SourceIngestResult> {
    return this.request<SourceIngestResult>('POST', '/sources/ingest', { data: options });
  }

  async uploadSource(options: SourceUploadOptions): Promise<SourceIngestResult> {
    if (this.closed) {
      throw new HystersisError('Client is closed');
    }
    await this.rateLimiter.acquire();

    const form = new FormData();
    const contentType = options.contentType ?? 'application/octet-stream';
    const blob = options.file instanceof Blob
      ? options.file
      : new Blob([options.file], { type: contentType });
    form.append('file', blob, options.filename);
    if (options.title) form.append('title', options.title);
    if (options.user_id) form.append('user_id', options.user_id);
    if (options.org_id) form.append('org_id', options.org_id);
    if (options.agent_id) form.append('agent_id', options.agent_id);
    if (options.metadata) form.append('metadata', JSON.stringify(options.metadata));

    return this.sendRequest<SourceIngestResult>(this.buildMultipartRequest('/sources/upload', form));
  }

  async listSources(options?: {
    user_id?: string;
    org_id?: string;
    limit?: number;
    offset?: number;
  }): Promise<SourceListResponse> {
    const params: Record<string, unknown> = { limit: options?.limit ?? 50, offset: options?.offset ?? 0 };
    if (options?.user_id) params.user_id = options.user_id;
    if (options?.org_id) params.org_id = options.org_id;
    return this.request<SourceListResponse>('GET', '/sources', { params });
  }

  async getSource(sourceId: string): Promise<Source> {
    return this.request<Source>('GET', `/sources/${sourceId}`);
  }

  async deleteSource(sourceId: string): Promise<{ status: string; source_id: string }> {
    return this.request<{ status: string; source_id: string }>('DELETE', `/sources/${sourceId}`);
  }

  // ==================== Feedback ====================

  async addFeedback(memoryId: string, feedbackType: string, comment?: string, userId?: string): Promise<any> {
    const data: Record<string, unknown> = { memory_id: memoryId, type: feedbackType };
    if (comment) data.comment = comment;
    if (userId) data.user_id = userId;
    return this.request<any>('POST', '/feedback', { data });
  }

  async getMemoriesByFeedback(feedbackType: string, limit = 50): Promise<any[]> {
    return this.request<any[]>('GET', '/feedback/memories', {
      params: { type: feedbackType, limit },
    });
  }

  // ==================== Entities ====================

  async createEntity(name: string, entityType: string, properties?: Record<string, unknown>): Promise<any> {
    const data: Record<string, unknown> = { name, type: entityType };
    if (properties) data.properties = properties;
    return this.request<any>('POST', '/entities', { data });
  }

  async getEntity(entityId: string): Promise<any> {
    return this.request<any>('GET', `/entities/${entityId}`);
  }

  async listEntities(entityType?: string, limit = 100, offset = 0): Promise<{ entities: any[]; count: number }> {
    const params: Record<string, unknown> = { limit, offset };
    if (entityType) params.entity_type = entityType;
    return this.request<{ entities: any[]; count: number }>('GET', '/entities', { params });
  }

  async updateEntity(entityId: string, updates: { name?: string; type?: string; properties?: Record<string, unknown> }): Promise<any> {
    return this.request<any>('PUT', `/entities/${entityId}`, { data: updates });
  }

  async deleteEntity(entityId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/entities/${entityId}`);
  }

  async getEntityMemories(entityId: string, limit = 50): Promise<any[]> {
    return this.request<any[]>('GET', `/entities/${entityId}/memories`, {
      params: { limit },
    });
  }

  async getEntityRelations(entityId: string, relationType?: string): Promise<any[]> {
    const params: Record<string, unknown> = {};
    if (relationType) params.type = relationType;
    return this.request<any[]>('GET', `/entities/${entityId}/relations`, { params });
  }

  // ==================== Relations ====================

  async createRelation(fromId: string, toId: string, relationType: string, metadata?: Record<string, unknown>): Promise<{ status: string }> {
    const data: Record<string, unknown> = { from_id: fromId, to_id: toId, type: relationType };
    if (metadata) data.metadata = metadata;
    return this.request<{ status: string }>('POST', '/relations', { data });
  }

  async deleteRelation(fromId: string, toId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/relations/${fromId}/${toId}`);
  }

  // ==================== Graph ====================

  async graphQuery(cypher: string, params?: Record<string, unknown>): Promise<any[]> {
    const data: Record<string, unknown> = { cypher };
    if (params) data.params = params;
    return this.request<any[]>('POST', '/graph/query', { data });
  }

  async graphTraverse(entityId: string, depth = 3): Promise<{ nodes: any[]; edges: any[] }> {
    return this.request<{ nodes: any[]; edges: any[] }>('GET', `/graph/traverse/${entityId}`, {
      params: { depth },
    });
  }

  // ==================== Projects ====================

  async createProject(name: string, description?: string, userId?: string, orgId?: string, settings?: any): Promise<any> {
    const data: Record<string, unknown> = { name };
    if (description) data.description = description;
    if (userId) data.user_id = userId;
    if (orgId) data.org_id = orgId;
    if (settings) data.settings = settings;
    return this.request<any>('POST', '/projects', { data });
  }

  async getProject(projectId: string): Promise<any> {
    return this.request<any>('GET', `/projects/${projectId}`);
  }

  async listProjects(userId?: string, orgId?: string, limit = 50, offset = 0): Promise<any[]> {
    const params: Record<string, unknown> = { limit, offset };
    if (userId) params.user_id = userId;
    if (orgId) params.org_id = orgId;
    return this.request<any[]>('GET', '/projects', { params });
  }

  async updateProject(projectId: string, updates: any): Promise<any> {
    return this.request<any>('PUT', `/projects/${projectId}`, { data: updates });
  }

  async deleteProject(projectId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/projects/${projectId}`);
  }

  // ==================== Webhooks ====================

  async createWebhook(url: string, events: string[], projectId?: string, secret?: string, active = true): Promise<any> {
    const data: Record<string, unknown> = { url, events, active };
    if (projectId) data.project_id = projectId;
    if (secret) data.secret = secret;
    return this.request<any>('POST', '/webhooks', { data });
  }

  async getWebhook(webhookId: string): Promise<any> {
    return this.request<any>('GET', `/webhooks/${webhookId}`);
  }

  async listWebhooks(projectId?: string, limit = 50, offset = 0): Promise<any[]> {
    const params: Record<string, unknown> = { limit, offset };
    if (projectId) params.project_id = projectId;
    return this.request<any[]>('GET', '/webhooks', { params });
  }

  async updateWebhook(webhookId: string, updates: any): Promise<any> {
    return this.request<any>('PUT', `/webhooks/${webhookId}`, { data: updates });
  }

  async deleteWebhook(webhookId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/webhooks/${webhookId}`);
  }

  async testWebhook(webhookId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('POST', `/webhooks/${webhookId}/test`);
  }

  // ==================== Skills ====================

  async createSkill(options: {
    name: string;
    trigger: string;
    action: string;
    domain?: string;
    confidence?: number;
    tags?: string[];
    examples?: string[];
    metadata?: Record<string, unknown>;
  }): Promise<any> {
    return this.request<any>('POST', '/skills', { data: options });
  }

  async getSkill(skillId: string): Promise<any> {
    return this.request<any>('GET', `/skills/${skillId}`);
  }

  async listSkills(domain?: string, limit = 50, offset = 0): Promise<{ skills: any[]; count: number }> {
    const params: Record<string, unknown> = { limit, offset };
    if (domain) params.domain = domain;
    return this.request<{ skills: any[]; count: number }>('GET', '/skills', { params });
  }

  async searchSkills(trigger?: string, domain?: string, limit = 20): Promise<{ skills: any[]; count: number }> {
    const params: Record<string, unknown> = { limit };
    if (trigger) params.trigger = trigger;
    if (domain) params.domain = domain;
    return this.request<{ skills: any[]; count: number }>('GET', '/skills/search', { params });
  }

  async updateSkill(skillId: string, updates: any): Promise<any> {
    return this.request<any>('PUT', `/skills/${skillId}`, { data: updates });
  }

  async deleteSkill(skillId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/skills/${skillId}`);
  }

  async useSkill(skillId: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>('POST', `/skills/${skillId}/use`);
  }

  async suggestSkills(trigger: string, context?: string, limit = 5): Promise<{ suggestions: any[] }> {
    const data: Record<string, unknown> = { trigger, limit };
    if (context) data.context = context;
    return this.request<{ suggestions: any[] }>('POST', '/skills/suggest', { data });
  }

  async synthesizeSkills(skillIds: string[]): Promise<any> {
    return this.request<any>('POST', '/skills/synthesize', { data: { skill_ids: skillIds } });
  }

  async extractSkills(content: string, userId?: string, agentId?: string): Promise<{ skills: any[] }> {
    const data: Record<string, unknown> = { content };
    if (userId) data.user_id = userId;
    if (agentId) data.agent_id = agentId;
    return this.request<{ skills: any[] }>('POST', '/skills/extract', { data });
  }

  // ==================== Skill Chains ====================

  async createChain(name: string, trigger: string, steps: any[], conditions?: any[]): Promise<any> {
    const data: Record<string, unknown> = { name, trigger, steps };
    if (conditions) data.conditions = conditions;
    return this.request<any>('POST', '/chains', { data });
  }

  async getChain(chainId: string): Promise<any> {
    return this.request<any>('GET', `/chains/${chainId}`);
  }

  async listChains(status?: string, limit = 50, offset = 0): Promise<{ chains: any[]; count: number }> {
    const params: Record<string, unknown> = { limit, offset };
    if (status) params.status = status;
    return this.request<{ chains: any[]; count: number }>('GET', '/chains', { params });
  }

  async updateChain(chainId: string, updates: any): Promise<any> {
    return this.request<any>('PUT', `/chains/${chainId}`, { data: updates });
  }

  async deleteChain(chainId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/chains/${chainId}`);
  }

  async executeChain(chainId: string, context: Record<string, unknown>, timeoutMs?: number): Promise<any> {
    const data: Record<string, unknown> = { context };
    if (timeoutMs) data.timeout_ms = timeoutMs;
    return this.request<any>('POST', `/chains/${chainId}/execute`, { data });
  }

  // ==================== Reviews ====================

  async listReviews(status?: string, limit = 50, offset = 0): Promise<{ reviews: any[]; count: number }> {
    const params: Record<string, unknown> = { limit, offset };
    if (status) params.status = status;
    return this.request<{ reviews: any[]; count: number }>('GET', '/reviews', { params });
  }

  async getReview(reviewId: string): Promise<any> {
    return this.request<any>('GET', `/reviews/${reviewId}`);
  }

  async processReview(reviewId: string, approved: boolean, notes?: string): Promise<{ success: boolean }> {
    const data: Record<string, unknown> = { approved };
    if (notes) data.notes = notes;
    return this.request<{ success: boolean }>('POST', `/reviews/${reviewId}`, { data });
  }

  // ==================== Agents ====================

  async createAgent(name: string, description?: string, config?: any): Promise<any> {
    const data: Record<string, unknown> = { name };
    if (description) data.description = description;
    if (config) data.config = config;
    return this.request<any>('POST', '/agents', { data });
  }

  async getAgent(agentId: string): Promise<any> {
    return this.request<any>('GET', `/agents/${agentId}`);
  }

  async listAgents(status?: string, limit = 50, offset = 0): Promise<{ agents: any[]; total: number }> {
    const params: Record<string, unknown> = { limit, offset };
    if (status) params.status = status;
    return this.request<{ agents: any[]; total: number }>('GET', '/agents', { params });
  }

  async updateAgent(agentId: string, updates: any): Promise<any> {
    return this.request<any>('PUT', `/agents/${agentId}`, { data: updates });
  }

  async deleteAgent(agentId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/agents/${agentId}`);
  }

  // ==================== Groups ====================

  async createGroup(name: string, description?: string, domain?: string, policy?: any): Promise<any> {
    const data: Record<string, unknown> = { name };
    if (description) data.description = description;
    if (domain) data.domain = domain;
    if (policy) data.policy = policy;
    return this.request<any>('POST', '/groups', { data });
  }

  async getGroup(groupId: string): Promise<any> {
    return this.request<any>('GET', `/groups/${groupId}`);
  }

  async listGroups(domain?: string, limit = 50, offset = 0): Promise<{ groups: any[]; total: number }> {
    const params: Record<string, unknown> = { limit, offset };
    if (domain) params.domain = domain;
    return this.request<{ groups: any[]; total: number }>('GET', '/groups', { params });
  }

  async updateGroup(groupId: string, updates: any): Promise<any> {
    return this.request<any>('PUT', `/groups/${groupId}`, { data: updates });
  }

  async deleteGroup(groupId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/groups/${groupId}`);
  }

  async addMember(groupId: string, agentId: string, role = 'contributor'): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>('POST', `/groups/${groupId}/members`, {
      data: { agent_id: agentId, role },
    });
  }

  async removeMember(groupId: string, agentId: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>('DELETE', `/groups/${groupId}/members/${agentId}`);
  }

  async getGroupSkills(groupId: string, limit = 50): Promise<{ skills: any[]; count: number }> {
    return this.request<{ skills: any[]; count: number }>('GET', `/groups/${groupId}/skills`, {
      params: { limit },
    });
  }

  async getGroupMemories(groupId: string): Promise<{ memories: any[]; count: number }> {
    return this.request<{ memories: any[]; count: number }>('GET', `/groups/${groupId}/memories`);
  }

  async shareMemoryToGroup(groupId: string, memoryId: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>('POST', `/groups/${groupId}/memories`, {
      data: { memory_id: memoryId },
    });
  }

  // ==================== Notifications ====================

  async listNotifications(read?: boolean, limit = 50, offset = 0): Promise<{ notifications: any[]; count: number }> {
    const params: Record<string, unknown> = { limit, offset };
    if (read !== undefined) params.read = read;
    return this.request<{ notifications: any[]; count: number }>('GET', '/notifications', { params });
  }

  async markNotificationRead(notificationId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('POST', `/notifications/${notificationId}/read`);
  }

  async markAllNotificationsRead(): Promise<{ status: string }> {
    return this.request<{ status: string }>('POST', '/notifications/read-all');
  }

  // ==================== Admin ====================

  async adminCleanup(): Promise<{ cleaned_up: number }> {
    return this.request<{ cleaned_up: number }>('POST', '/admin/cleanup');
  }

  async adminSync(entityIds?: string[]): Promise<{ status: string }> {
    const data: Record<string, unknown> = {};
    if (entityIds) data.entity_ids = entityIds;
    return this.request<{ status: string }>('POST', '/admin/sync', { data });
  }

  async adminAnalytics(): Promise<any> {
    return this.request<any>('GET', '/analytics/dashboard');
  }

  async listApiKeys(): Promise<any[]> {
    return this.request<any[]>('GET', '/admin/api-keys');
  }

  async createApiKey(label: string, expiresInHours = 0, tenantId?: string): Promise<any> {
    const data: Record<string, unknown> = { label, expires_in_hours: expiresInHours };
    if (tenantId) data.tenant_id = tenantId;
    return this.request<any>('POST', '/admin/api-keys', { data });
  }

  async deleteApiKey(keyId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/admin/api-keys/${keyId}`);
  }

  // ==================== Users ====================

  async inviteUser(email: string, role = 'member'): Promise<any> {
    return this.request<any>('POST', '/admin/invites', { data: { email, role } });
  }

  async listInvitations(status?: string): Promise<{ invites: any[]; count: number }> {
    const params: Record<string, unknown> = {};
    if (status) params.status = status;
    return this.request<{ invites: any[]; count: number }>('GET', '/admin/invites', { params });
  }

  async cancelInvitation(inviteId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/admin/invites/${inviteId}`);
  }

  async acceptInvitation(inviteId: string, name: string, password: string): Promise<any> {
    return this.request<any>('POST', `/admin/invites/${inviteId}/accept`, {
      data: { name, password },
    });
  }

  // ==================== Compression Engine ====================

  async getCompressionStats(): Promise<{
    accuracy_retention: number;
    token_reduction: number;
    total_tokens_saved: number;
    extractions_performed: number;
    spreading_activations: number;
    avg_latency_ms: number;
    p95_latency_ms: number;
  }> {
    return this.request<{
      accuracy_retention: number;
      token_reduction: number;
      total_tokens_saved: number;
      extractions_performed: number;
      spreading_activations: number;
      avg_latency_ms: number;
      p95_latency_ms: number;
    }>('GET', '/compression/stats');
  }

  async getCompressionMode(): Promise<{ mode: string }> {
    return this.request<{ mode: string }>('GET', '/compression/mode');
  }

  async setCompressionMode(mode: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>('PUT', '/compression/mode', { data: { mode } });
  }

  async getTierPolicy(): Promise<{ policy: string }> {
    return this.request<{ policy: string }>('GET', '/tier/policy');
  }

  async setTierPolicy(policy: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>('PUT', '/tier/policy', { data: { policy } });
  }

  async searchEnhanced(query: string, mode = 'spreading', limit = 10): Promise<{ results: any[]; mode: string }> {
    return this.request<{ results: any[]; mode: string }>('GET', '/search/enhanced', {
      params: { query, mode, limit },
    });
  }

  async temporalSearch(query: string, options?: { time_start?: string; time_end?: string; limit?: number }): Promise<any[]> {
    const params: Record<string, unknown> = { q: query };
    if (options?.time_start) params.time_start = options.time_start;
    if (options?.time_end) params.time_end = options.time_end;
    if (options?.limit) params.limit = String(options.limit);
    return this.request<any[]>('GET', '/search', { params });
  }

  async getProvenanceChain(memoryId: string): Promise<any[]> {
    return this.request<any[]>('GET', `/memories/${memoryId}/versions`);
  }

  // ==================== Memory Links & Versions ====================

  async createMemoryLink(fromId: string, toId: string, linkType: string, weight = 0.5, metadata?: Record<string, unknown>): Promise<any> {
    const data: Record<string, unknown> = { from_id: fromId, to_id: toId, type: linkType, weight };
    if (metadata) data.metadata = metadata;
    return this.request<any>('POST', '/memories/links', { data });
  }

  async getMemoryLinks(memoryId: string): Promise<any[]> {
    return this.request<any[]>('GET', `/memories/${memoryId}/links`);
  }

  async deleteMemoryLink(linkId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('DELETE', `/memories/links/${linkId}`);
  }

  async getMemoryVersions(memoryId: string): Promise<any[]> {
    return this.request<any[]>('GET', `/memories/${memoryId}/versions`);
  }

  async restoreMemoryVersion(memoryId: string, versionId: string): Promise<{ status: string }> {
    return this.request<{ status: string }>('POST', `/memories/${memoryId}/restore`, {
      data: { version_id: versionId },
    });
  }

  // ==================== Analytics ====================

  async getMemoryStats(userId?: string, orgId?: string): Promise<any> {
    const params: Record<string, unknown> = {};
    if (userId) params.user_id = userId;
    if (orgId) params.org_id = orgId;
    return this.request<any>('GET', '/memories/stats', { params });
  }

  async getMemoryInsights(userId?: string, orgId?: string): Promise<any[]> {
    const params: Record<string, unknown> = {};
    if (userId) params.user_id = userId;
    if (orgId) params.org_id = orgId;
    return this.request<any[]>('GET', '/memories/insights', { params });
  }

  async getMemorySummary(userId?: string, orgId?: string): Promise<any> {
    const params: Record<string, unknown> = {};
    if (userId) params.user_id = userId;
    if (orgId) params.org_id = orgId;
    return this.request<any>('GET', '/memories/summary', { params });
  }

  // ==================== Concepts (GAAMA paper) ====================

  async createConcept(options: { name: string; description?: string }) {
    return this.request<any>('POST', '/concepts', { data: options });
  }

  async listConcepts() {
    return this.request<any>('GET', '/concepts');
  }

  async getConceptMemories(conceptId: string, limit?: number) {
    const params = limit ? `?limit=${limit}` : '';
    return this.request<any>('GET', `/concepts/${conceptId}/memories${params}`);
  }

  async linkToConcept(conceptId: string, nodeId: string, relType?: string) {
    return this.request<any>('POST', `/concepts/${conceptId}/link`, {
      data: { node_id: nodeId, rel_type: relType || 'BELONGS_TO' },
    });
  }

  // ==================== Reminders (prospective memory) ====================

  async setReminder(memoryId: string, remindAt: string, condition?: string) {
    return this.request<any>('POST', `/memories/${memoryId}/remind`, {
      data: { remind_at: remindAt, condition: condition || '' },
    });
  }

  async listReminders() {
    return this.request<any>('GET', '/reminders');
  }

  // ==================== Safety ====================

  async checkSafety(content: string) {
    return this.request<any>('POST', '/safety/check', { data: { content } });
  }

  // ==================== Aliases for Backward Compatibility ====================

  // Sessions
  sessions = {
    create: this.createSession.bind(this),
    get: this.getSession.bind(this),
    delete: this.deleteSession.bind(this),
    list: this.listSessions.bind(this),
    messages: {
      add: this.addMessage.bind(this),
      list: this.getMessages.bind(this),
    },
  };

  // Memories
  memories = {
    create: this.createMemory.bind(this),
    get: this.getMemory.bind(this),
    update: this.updateMemory.bind(this),
    delete: this.deleteMemory.bind(this),
    list: this.listMemories.bind(this),
    search: this.search.bind(this),
    history: this.getMemoryHistory.bind(this),
    setExpiration: this.setMemoryExpiration.bind(this),
    linkToEntity: this.linkMemoryToEntity.bind(this),
    batch: {
      create: this.batchCreateMemories.bind(this),
      update: this.batchUpdateMemories.bind(this),
    },
    bulkDelete: this.bulkDelete.bind(this),
    createLink: this.createMemoryLink.bind(this),
    getLinks: this.getMemoryLinks.bind(this),
    deleteLink: this.deleteMemoryLink.bind(this),
    getVersions: this.getMemoryVersions.bind(this),
    restoreVersion: this.restoreMemoryVersion.bind(this),
    getStats: this.getMemoryStats.bind(this),
    getInsights: this.getMemoryInsights.bind(this),
    getSummary: this.getMemorySummary.bind(this),
  };

  // Mem0/Supermemory-compatible v3 surface
  v3 = {
    add: this.v3AddMemory.bind(this),
    search: this.v3SearchMemories.bind(this),
    list: this.v3ListMemories.bind(this),
  };

  // Async operation events
  events = {
    get: this.getEventStatus.bind(this),
  };

  // Import/export
  transfer = {
    export: this.exportMemories.bind(this),
    import: this.importMemories.bind(this),
  };

  // Sources
  sources = {
    ingest: this.ingestSource.bind(this),
    upload: this.uploadSource.bind(this),
    list: this.listSources.bind(this),
    get: this.getSource.bind(this),
    delete: this.deleteSource.bind(this),
  };

  // Feedback
  feedback = {
    add: this.addFeedback.bind(this),
    getByType: this.getMemoriesByFeedback.bind(this),
  };

  // Entities
  entities = {
    create: this.createEntity.bind(this),
    get: this.getEntity.bind(this),
    list: this.listEntities.bind(this),
    update: this.updateEntity.bind(this),
    delete: this.deleteEntity.bind(this),
    getMemories: this.getEntityMemories.bind(this),
    relations: {
      get: this.getEntityRelations.bind(this),
    },
  };

  // Relations
  relations = {
    create: this.createRelation.bind(this),
    delete: this.deleteRelation.bind(this),
  };

  // Graph
  graph = {
    query: this.graphQuery.bind(this),
    traverse: this.graphTraverse.bind(this),
  };

  // Projects
  projects = {
    create: this.createProject.bind(this),
    get: this.getProject.bind(this),
    list: this.listProjects.bind(this),
    update: this.updateProject.bind(this),
    delete: this.deleteProject.bind(this),
  };

  // Webhooks
  webhooks = {
    create: this.createWebhook.bind(this),
    get: this.getWebhook.bind(this),
    list: this.listWebhooks.bind(this),
    update: this.updateWebhook.bind(this),
    delete: this.deleteWebhook.bind(this),
    test: this.testWebhook.bind(this),
  };

  // Admin
  admin = {
    cleanup: this.adminCleanup.bind(this),
    sync: this.adminSync.bind(this),
    analytics: this.adminAnalytics.bind(this),
    apiKeys: {
      list: this.listApiKeys.bind(this),
      create: this.createApiKey.bind(this),
      delete: this.deleteApiKey.bind(this),
    },
  };

  // Skills
  skills = {
    create: this.createSkill.bind(this),
    get: this.getSkill.bind(this),
    list: this.listSkills.bind(this),
    search: this.searchSkills.bind(this),
    update: this.updateSkill.bind(this),
    delete: this.deleteSkill.bind(this),
    use: this.useSkill.bind(this),
    suggest: this.suggestSkills.bind(this),
    synthesize: this.synthesizeSkills.bind(this),
    extract: this.extractSkills.bind(this),
  };

  // Chains
  chains = {
    create: this.createChain.bind(this),
    get: this.getChain.bind(this),
    list: this.listChains.bind(this),
    update: this.updateChain.bind(this),
    delete: this.deleteChain.bind(this),
    execute: this.executeChain.bind(this),
  };

  // Agents
  agents = {
    create: this.createAgent.bind(this),
    get: this.getAgent.bind(this),
    list: this.listAgents.bind(this),
    update: this.updateAgent.bind(this),
    delete: this.deleteAgent.bind(this),
  };

  // Groups
  groups = {
    create: this.createGroup.bind(this),
    get: this.getGroup.bind(this),
    list: this.listGroups.bind(this),
    update: this.updateGroup.bind(this),
    delete: this.deleteGroup.bind(this),
    addMember: this.addMember.bind(this),
    removeMember: this.removeMember.bind(this),
    getSkills: this.getGroupSkills.bind(this),
    getMemories: this.getGroupMemories.bind(this),
    shareMemory: this.shareMemoryToGroup.bind(this),
  };

  // Reviews
  reviews = {
    listPending: this.listReviews.bind(this),
    get: this.getReview.bind(this),
    process: this.processReview.bind(this),
  };

  // Notifications
  notifications = {
    list: this.listNotifications.bind(this),
    markRead: this.markNotificationRead.bind(this),
    markAllRead: this.markAllNotificationsRead.bind(this),
  };

  // Concepts
  concepts = {
    create: this.createConcept.bind(this),
    list: this.listConcepts.bind(this),
    getMemories: this.getConceptMemories.bind(this),
    link: this.linkToConcept.bind(this),
  };

  // Reminders
  reminders = {
    set: this.setReminder.bind(this),
    list: this.listReminders.bind(this),
  };

  // Safety
  safety = {
    check: this.checkSafety.bind(this),
  };

  // Compression
  compression = {
    getStats: this.getCompressionStats.bind(this),
    getMode: this.getCompressionMode.bind(this),
    setMode: this.setCompressionMode.bind(this),
    getTierPolicy: this.getTierPolicy.bind(this),
    setTierPolicy: this.setTierPolicy.bind(this),
    searchEnhanced: this.searchEnhanced.bind(this),
  };

  // Temporal search
  temporal = {
    search: this.temporalSearch.bind(this),
  };

  // Provenance tracking
  provenance = {
    getChain: this.getProvenanceChain.bind(this),
  };

  // Legacy aliases
  async create_session(agentId: string, metadata?: Record<string, unknown>) {
    return this.createSession(agentId, metadata);
  }
  async get_session(sessionId: string) {
    return this.getSession(sessionId);
  }
  async delete_session(sessionId: string) {
    return this.deleteSession(sessionId);
  }
  async create_memory(options: any) {
    return this.createMemory(options);
  }
  async get_memory(memoryId: string) {
    return this.getMemory(memoryId);
  }
  async list_memories(options?: any) {
    return this.listMemories(options);
  }
  async semantic_search(query: string, options?: any) {
    return this.search(query, options);
  }
  async add_feedback(memoryId: string, feedbackType: string, comment?: string) {
    return this.addFeedback(memoryId, feedbackType, comment);
  }
  async create_entity(name: string, entityType: string, properties?: Record<string, unknown>) {
    return this.createEntity(name, entityType, properties);
  }
  async list_entities(entityType?: string, limit = 100) {
    return this.listEntities(entityType, limit);
  }
  async create_skill(options: any) {
    return this.createSkill(options);
  }
  async list_skills(domain?: string, limit = 50) {
    return this.listSkills(domain, limit);
  }
  async create_agent(name: string, description?: string, config?: any) {
    return this.createAgent(name, description, config);
  }
  async create_group(name: string, description?: string, domain?: string) {
    return this.createGroup(name, description, domain);
  }
}

// For backward compatibility, also export as default
export default HystersisClient;
