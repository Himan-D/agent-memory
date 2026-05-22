/**
 * Core Types
 */

// Config Types
export interface TimeoutConfig {
  connect: number;
  read: number;
  write: number;
  pool: number;
}

export interface RetryConfig {
  maxRetries: number;
  baseDelay: number;
  maxDelay: number;
  exponentialBase: number;
  retryOnStatusCodes: number[];
}

export interface RateLimitConfig {
  requestsPerSecond: number;
  burstSize: number;
}

export interface HystersisConfig {
  baseUrl: string;
  apiKey?: string;
  timeout?: TimeoutConfig;
  retry?: RetryConfig;
  rateLimit?: RateLimitConfig;
  maxConnections?: number;
  maxKeepaliveConnections?: number;
}

export type RequestInterceptor = (request: Request) => Request | Promise<Request>;
export type ResponseInterceptor = (response: Response, request: Request) => Response | Promise<Response>;

// Enums
export type MemoryType = 'conversation' | 'session' | 'user' | 'org';
export type FeedbackType = 'positive' | 'negative' | 'very_negative';
export type MemoryStatus = 'active' | 'archived' | 'deleted';
export type ImportanceLevel = 'critical' | 'high' | 'medium' | 'low';
export type MemoryLinkType = 'parent' | 'related' | 'reply' | 'cite';
export type MemberRole = 'admin' | 'contributor' | 'reader';
export type ReviewStatus = 'pending' | 'approved' | 'rejected';
export type AgentStatus = 'active' | 'inactive' | 'suspended';
export type CompressionMode = 'extract' | 'balanced' | 'aggressive';
export type TierPolicy = 'aggressive' | 'balanced' | 'conservative';
export type SearchMode = 'vector' | 'spreading' | 'hybrid';
export type ChainStatus = 'active' | 'paused' | 'completed' | 'failed';

// Interface Types
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
  worth_score?: number;
  validity_status?: string;
  volatility_score?: number;
  phase_angle?: number;
  provenance_edges?: string[];
  q_value?: number;
  pool_type?: string;
  retrieval_count?: number;
}

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
  expirationDate?: string;
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
  mode?: SearchMode;
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