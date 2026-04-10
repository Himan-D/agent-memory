/**
 * Agent Memory SDK - Shared Types
 */

export interface LangChainMemoryConfig {
  sessionId: string;
  memoryType?: 'user' | 'session' | 'conversation' | 'org';
  userId?: string;
  agentId?: string;
  baseUrl: string;
  apiKey?: string;
  returnMessages?: boolean;
  inputKey?: string;
  outputKey?: string;
}

export interface MastraMemoryConfig {
  userId?: string;
  orgId?: string;
  agentId?: string;
  baseUrl: string;
  apiKey?: string;
  name?: string;
  description?: string;
}

export interface AgnoMemoryConfig {
  userId?: string;
  orgId?: string;
  agentId?: string;
  sessionId?: string;
  baseUrl: string;
  apiKey?: string;
}

export interface AutoGenMemoryConfig {
  groupId: string;
  userId?: string;
  orgId?: string;
  baseUrl: string;
  apiKey?: string;
}

export interface LlamaIndexReaderConfig {
  baseUrl: string;
  apiKey?: string;
  userId?: string;
  orgId?: string;
  agentId?: string;
}

export interface CrewMemoryConfig {
  crewId: string;
  userId?: string;
  orgId?: string;
  baseUrl: string;
  apiKey?: string;
}

export interface CrewAgentConfig {
  agentId: string;
  role?: string;
  goal?: string;
}
