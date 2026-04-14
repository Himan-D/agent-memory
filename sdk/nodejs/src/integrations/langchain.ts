/**
 * LangChain Integration for Hystersis - Node.js SDK
 * 
 * Provides LangChain memory components and retrievers for Hystersis.
 * 
 * @example
 * ```typescript
 * import { HystersisMemory } from 'hystersis/integrations/langchain';
 * import { ConversationChain } from 'langchain/chains';
 * import { ChatOpenAI } from 'langchain/chat_models';
 * 
 * const memory = new HystersisMemory({
 *   sessionId: 'user-123',
 *   baseUrl: 'http://localhost:8080'
 * });
 * 
 * const chain = new ConversationChain({
 *   llm: new ChatOpenAI({ temperature: 0 }),
 *   memory,
 * });
 * ```
 */

import { Hystersis, type Memory, type Message } from '../index.js';

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

export interface MemoryVariables {
  history: string | Message[];
}

/**
 * LangChain compatible memory component using Hystersis backend.
 */
export class HystersisMemory {
  private client: Hystersis;
  private sessionId: string;
  private memoryType: 'user' | 'session' | 'conversation' | 'org';
  private userId?: string;
  private agentId?: string;
  private returnMessages: boolean;
  private inputKey: string;
  private outputKey: string;

  constructor(config: LangChainMemoryConfig) {
    this.client = new Hystersis({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
    this.sessionId = config.sessionId;
    this.memoryType = config.memoryType ?? 'session';
    this.userId = config.userId;
    this.agentId = config.agentId;
    this.returnMessages = config.returnMessages ?? false;
    this.inputKey = config.inputKey ?? 'input';
    this.outputKey = config.outputKey ?? 'output';
  }

  /**
   * Get all messages from the session
   */
  async getMessages(): Promise<Message[]> {
    try {
      const result = await this.client.sessions.messages.list(this.sessionId);
      return result;
    } catch {
      return [];
    }
  }

  /**
   * Load memory variables for LangChain
   */
  async loadMemoryVariables(_inputs: Record<string, unknown>): Promise<MemoryVariables> {
    const messages = await this.getMessages();

    if (this.returnMessages) {
      return { history: messages };
    }

    const history = this.formatHistory(messages);
    return { history };
  }

  /**
   * Format messages into conversation string
   */
  private formatHistory(messages: Message[]): string {
    const formatted: string[] = [];
    for (const msg of messages) {
      if (msg.role === 'user') {
        formatted.push(`Human: ${msg.content}`);
      } else if (msg.role === 'assistant') {
        formatted.push(`AI: ${msg.content}`);
      } else {
        formatted.push(`${msg.role}: ${msg.content}`);
      }
    }
    return formatted.join('\n');
  }

  /**
   * Save context from the current conversation turn
   */
  async saveContext(
    inputs: Record<string, unknown>,
    outputs: Record<string, unknown>
  ): Promise<void> {
    const inputText = inputs[this.inputKey] as string;
    const outputText = outputs[this.outputKey] as string;

    if (inputText) {
      await this.addMessage('user', inputText);
    }
    if (outputText) {
      await this.addMessage('assistant', outputText);
    }
  }

  /**
   * Add a message to the session
   */
  async addMessage(role: 'user' | 'assistant' | 'system' | 'tool', content: string): Promise<void> {
    try {
      await this.client.sessions.messages.add(this.sessionId, role, content);
    } catch (error) {
      console.warn('Failed to save message:', error);
    }
  }

  /**
   * Clear all messages from the session
   */
  async clear(): Promise<void> {
    try {
      await this.client.sessions.delete(this.sessionId);
    } catch (error) {
      console.warn('Failed to clear session:', error);
    }
  }

  /**
   * Search past memories semantically
   */
  async searchMemories(query: string, limit = 5): Promise<Memory[]> {
    try {
      const results = await this.client.memories.search(query, { limit });
      return results.map((r) => r.metadata).filter((m): m is Memory => m !== undefined);
    } catch {
      return [];
    }
  }

  /**
   * Get relevant memories as formatted strings
   */
  async getRelevantMemories(query: string, limit = 5, threshold = 0.5): Promise<string[]> {
    const results = await this.searchMemories(query, limit);
    const memories: string[] = [];

    for (const mem of results) {
      if (mem && (mem as Memory).content) {
        const score = (mem as unknown as { score?: number }).score ?? 1;
        if (score >= threshold) {
          memories.push((mem as Memory).content);
        }
      }
    }

    return memories;
  }

  /**
   * LangChain BaseMemory compatibility
   */
  get memoryKeys(): string[] {
    return ['history'];
  }
}

/**
 * LangChain Retriever for Hystersis
 */
export class HystersisRetriever {
  private client: Hystersis;
  private userId?: string;
  private orgId?: string;
  private agentId?: string;
  private memoryType?: 'user' | 'session' | 'conversation' | 'org';
  private topK: number;
  private scoreThreshold: number;

  constructor(config: {
    baseUrl: string;
    apiKey?: string;
    userId?: string;
    orgId?: string;
    agentId?: string;
    memoryType?: 'user' | 'session' | 'conversation' | 'org';
    topK?: number;
    scoreThreshold?: number;
  }) {
    this.client = new Hystersis({ baseUrl: config.baseUrl, apiKey: config.apiKey });
    this.userId = config.userId;
    this.orgId = config.orgId;
    this.agentId = config.agentId;
    this.memoryType = config.memoryType;
    this.topK = config.topK ?? 5;
    this.scoreThreshold = config.scoreThreshold ?? 0.5;
  }

  /**
   * Get relevant documents for a query
   */
  async getRelevantDocuments(query: string): Promise<Memory[]> {
    const results = await this.client.memories.search(query, {
      limit: this.topK,
      threshold: this.scoreThreshold,
      userId: this.userId,
      orgId: this.orgId,
      agentId: this.agentId,
      memoryType: this.memoryType,
    });

    return results.map((r) => r.metadata).filter((m): m is Memory => m !== undefined);
  }

  /**
   * LangChain Retriever compatibility
   */
  async getRelevantDocs(query: string): Promise<Memory[]> {
    return this.getRelevantDocuments(query);
  }
}
