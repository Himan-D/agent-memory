/**
 * Flowise Integration for Hystersis - Node.js SDK
 *
 * Provides a custom Flowise node (INode interface) for integrating
 * Hystersis memory into Flowise visual workflows.
 *
 * @example
 * ```typescript
 * import { HystersisFlowiseNode } from 'hystersis/integrations/flowise';
 *
 * // Register as a custom Flowise node
 * const node = new HystersisFlowiseNode();
 * const instance = await node.init({ apiKey: 'key', baseUrl: 'https://api.hystersis.ai' });
 * const result = await node.run(instance, 'What do you know about the user?');
 * ```
 */

import { HystersisClient as Hystersis } from '../index.js';

export interface FlowiseNodeInput {
  apiKey: string;
  baseUrl: string;
  userId?: string;
  limit?: number;
}

export interface FlowiseNodeInstance {
  client: Hystersis;
  userId: string;
  limit: number;
}

/**
 * Custom Flowise node for Hystersis memory.
 *
 * Implements the Flowise INode interface pattern with type, label,
 * name, description, inputs, init(), and run() methods.
 */
export class HystersisFlowiseNode {
  readonly type = 'HystersisMemory';
  readonly label = 'Hystersis Memory';
  readonly name = 'hystersisMemory';
  readonly description = 'Long-term memory for AI agents powered by Hystersis';
  readonly category = 'Memory';
  readonly version = 1.0;

  readonly inputs = [
    {
      label: 'API Key',
      name: 'apiKey',
      type: 'password',
      description: 'Hystersis API key',
    },
    {
      label: 'Base URL',
      name: 'baseUrl',
      type: 'string',
      default: 'https://api.hystersis.ai',
      description: 'Hystersis API base URL',
    },
    {
      label: 'User ID',
      name: 'userId',
      type: 'string',
      optional: true,
      description: 'User identifier for memory operations',
    },
    {
      label: 'Result Limit',
      name: 'limit',
      type: 'number',
      default: 5,
      optional: true,
      description: 'Maximum number of memories to retrieve',
    },
  ];

  /**
   * Initialize the Flowise node instance.
   */
  async init(nodeData: FlowiseNodeInput): Promise<FlowiseNodeInstance> {
    const client = new Hystersis({
      baseUrl: nodeData.baseUrl,
      apiKey: nodeData.apiKey,
    });

    return {
      client,
      userId: nodeData.userId ?? 'default',
      limit: nodeData.limit ?? 5,
    };
  }

  /**
   * Run the memory node with a query.
   *
   * When used in a Flowise chain, this searches memory for relevant
   * context and returns it as a formatted string.
   */
  async run(instance: FlowiseNodeInstance, query: string): Promise<string> {
    try {
      const results = await instance.client.memories.search(query, {
        user_id: instance.userId,
        limit: instance.limit,
      });

      const items = Array.isArray(results) ? results : [];
      const memories = items
        .map((r: Record<string, unknown>) => (r.text as string) || (r.content as string) || '')
        .filter(Boolean);

      if (memories.length === 0) {
        return 'No relevant memories found.';
      }

      return memories.map((m) => `- ${m}`).join('\n');
    } catch {
      return 'Error retrieving memories.';
    }
  }

  /**
   * Store content in memory (for use in Flowise tool nodes).
   */
  async store(instance: FlowiseNodeInstance, content: string): Promise<string> {
    try {
      await instance.client.memories.create({
        content,
        user_id: instance.userId,
        type: 'user',
        metadata: { source: 'flowise' },
      });
      return 'Memory stored successfully.';
    } catch (error) {
      return `Failed to store memory: ${error}`;
    }
  }

  /**
   * Clear all memories for the user (for use in Flowise tool nodes).
   */
  async clear(instance: FlowiseNodeInstance): Promise<string> {
    try {
      await instance.client.memories.deleteAll({ user_id: instance.userId });
      return 'All memories cleared.';
    } catch (error) {
      return `Failed to clear memories: ${error}`;
    }
  }
}
