/**
 * OpenClaw Integration for Hystersis
 *
 * Provides memory tools for OpenClaw personal AI assistants.
 * OpenClaw uses a skill-based system where each tool is a
 * function-calling endpoint backed by the Hystersis API.
 *
 * @example
 * ```typescript
 * import { OpenClawMemoryTools } from 'hystersis/integrations/openclaw';
 *
 * const tools = new OpenClawMemoryTools({
 *   baseUrl: 'http://localhost:8080',
 *   apiKey: 'your-key'
 * });
 *
 * // Get all tool definitions for OpenClaw config
 * const toolDefs = tools.getToolDefinitions();
 *
 * // Execute a tool by name
 * const result = await tools.executeTool('semantic_search', { query: 'machine learning' });
 * ```
 */

import { HystersisClient } from '../index.js';

export interface OpenClawConfig {
  baseUrl: string;
  apiKey?: string;
}

export interface ToolDefinition {
  name: string;
  description: string;
  inputSchema: Record<string, unknown>;
}

export interface ToolResult {
  success: boolean;
  data?: unknown;
  error?: string;
}

/**
 * Memory tools for OpenClaw integration
 */
export class OpenClawMemoryTools {
  private client: HystersisClient;

  constructor(config: OpenClawConfig) {
    this.client = new HystersisClient({
      baseUrl: config.baseUrl,
      apiKey: config.apiKey,
    });
  }

  /**
   * Get all tool definitions for OpenClaw configuration
   */
  getToolDefinitions(): ToolDefinition[] {
    return [
      {
        name: 'create_session',
        description: 'Create a new agent session',
        inputSchema: {
          type: 'object',
          properties: {
            agent_id: { type: 'string', description: 'Unique agent identifier' },
            metadata: { type: 'object', description: 'Optional metadata' },
          },
          required: ['agent_id'],
        },
      },
      {
        name: 'add_message',
        description: 'Add a message to session context',
        inputSchema: {
          type: 'object',
          properties: {
            session_id: { type: 'string' },
            role: { type: 'string', enum: ['user', 'assistant', 'system', 'tool'] },
            content: { type: 'string' },
          },
          required: ['session_id', 'role', 'content'],
        },
      },
      {
        name: 'get_messages',
        description: 'Get messages from a session',
        inputSchema: {
          type: 'object',
          properties: {
            session_id: { type: 'string' },
            limit: { type: 'integer', default: 50 },
          },
          required: ['session_id'],
        },
      },
      {
        name: 'create_entity',
        description: 'Create a knowledge graph entity',
        inputSchema: {
          type: 'object',
          properties: {
            name: { type: 'string' },
            entity_type: { type: 'string' },
            properties: { type: 'object' },
          },
          required: ['name', 'entity_type'],
        },
      },
      {
        name: 'get_entity',
        description: 'Get an entity and its relations',
        inputSchema: {
          type: 'object',
          properties: {
            entity_id: { type: 'string' },
          },
          required: ['entity_id'],
        },
      },
      {
        name: 'create_relation',
        description: 'Create entity relationship',
        inputSchema: {
          type: 'object',
          properties: {
            from_id: { type: 'string' },
            to_id: { type: 'string' },
            relation_type: { type: 'string' },
          },
          required: ['from_id', 'to_id', 'relation_type'],
        },
      },
      {
        name: 'semantic_search',
        description: 'Search similar content',
        inputSchema: {
          type: 'object',
          properties: {
            query: { type: 'string' },
            limit: { type: 'integer', default: 10 },
            threshold: { type: 'number', default: 0.5 },
          },
          required: ['query'],
        },
      },
      {
        name: 'health_check',
        description: 'Check memory service health',
        inputSchema: { type: 'object', properties: {} },
      },
    ];
  }

  /**
   * Execute a tool by name with given args
   */
  async executeTool(name: string, args: Record<string, unknown>): Promise<ToolResult> {
    try {
      switch (name) {
        case 'create_session': {
          const session = await this.client.createSession(
            args.agent_id as string,
            args.metadata as Record<string, unknown> | undefined,
          );
          return { success: true, data: session };
        }

        case 'add_message': {
          const message = await this.client.addMessage(
            args.session_id as string,
            args.role as string,
            args.content as string,
          );
          return { success: true, data: message };
        }

        case 'get_messages': {
          const messages = await this.client.getMessages(
            args.session_id as string,
            (args.limit as number) || 50,
          );
          return { success: true, data: messages };
        }

        case 'create_entity': {
          const entity = await this.client.createEntity(
            args.name as string,
            args.entity_type as string,
            args.properties as Record<string, unknown> | undefined,
          );
          return { success: true, data: entity };
        }

        case 'get_entity': {
          const entity = await this.client.getEntity(args.entity_id as string);
          return { success: true, data: entity };
        }

        case 'create_relation': {
          const relation = await this.client.createRelation(
            args.from_id as string,
            args.to_id as string,
            args.relation_type as string,
          );
          return { success: true, data: relation };
        }

        case 'semantic_search': {
          const results = await this.client.search(
            args.query as string,
            {
              limit: (args.limit as number) || 10,
              threshold: (args.threshold as number) || 0.5,
            },
          );
          return { success: true, data: results };
        }

        case 'health_check': {
          const health = await this.client.health();
          return { success: true, data: health };
        }

        default:
          return { success: false, error: `Unknown tool: ${name}` };
      }
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : String(err),
      };
    }
  }
}

export default OpenClawMemoryTools;
