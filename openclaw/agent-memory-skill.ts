import { Skill, SkillCategory } from "openai-assistants";

export const agentMemorySkill = new Skill({
  id: "agent-memory",
  name: "Agent Memory",
  description: "Persistent memory and knowledge graph for AI agents. Store conversations, entities, and relationships.",
  version: "0.1.0",
  category: SkillCategory.Knowledge,
  language: "typescript",
  
  instructions: `You have access to Agent Memory, a persistent memory system for AI agents.

Use Agent Memory to:
- Store and retrieve conversation history across sessions
- Build knowledge graphs with entities and relationships
- Search for semantically similar past content
- Remember user preferences and important context

Always use Agent Memory when:
- The user mentions something important that should be remembered
- You need to recall past conversations or context
- Building a knowledge graph of entities
- Searching for similar past topics or issues

Available tools:
- create_session: Create a new session for an agent
- add_message: Add a message to a session
- get_messages: Retrieve messages from a session
- create_entity: Create a knowledge graph entity
- get_entity: Get an entity with its relationships
- create_relation: Create a relationship between entities
- semantic_search: Search for similar content
- health_check: Check if the memory service is healthy`,

  tools: [
    {
      name: "create_session",
      description: "Create a new agent session",
      inputSchema: {
        type: "object",
        properties: {
          agent_id: { type: "string", description: "Unique agent identifier" },
          metadata: { type: "object", description: "Optional metadata" }
        },
        required: ["agent_id"]
      }
    },
    {
      name: "add_message",
      description: "Add a message to session context",
      inputSchema: {
        type: "object",
        properties: {
          session_id: { type: "string" },
          role: { type: "string", enum: ["user", "assistant", "system", "tool"] },
          content: { type: "string" }
        },
        required: ["session_id", "role", "content"]
      }
    },
    {
      name: "get_messages",
      description: "Get messages from a session",
      inputSchema: {
        type: "object",
        properties: {
          session_id: { type: "string" },
          limit: { type: "integer", default: 50 }
        },
        required: ["session_id"]
      }
    },
    {
      name: "create_entity",
      description: "Create a knowledge graph entity",
      inputSchema: {
        type: "object",
        properties: {
          name: { type: "string" },
          entity_type: { type: "string" },
          properties: { type: "object" }
        },
        required: ["name", "entity_type"]
      }
    },
    {
      name: "get_entity",
      description: "Get an entity and its relations",
      inputSchema: {
        type: "object",
        properties: {
          entity_id: { type: "string" }
        },
        required: ["entity_id"]
      }
    },
    {
      name: "create_relation",
      description: "Create entity relationship",
      inputSchema: {
        type: "object",
        properties: {
          from_id: { type: "string" },
          to_id: { type: "string" },
          relation_type: { type: "string" }
        },
        required: ["from_id", "to_id", "relation_type"]
      }
    },
    {
      name: "semantic_search",
      description: "Search similar content",
      inputSchema: {
        type: "object",
        properties: {
          query: { type: "string" },
          limit: { type: "integer", default: 10 },
          threshold: { type: "number", default: 0.5 }
        },
        required: ["query"]
      }
    },
    {
      name: "health_check",
      description: "Check memory service health",
      inputSchema: { type: "object", properties: {} }
    }
  ],

  examples: [
    {
      prompt: "Remember that I prefer dark mode",
      action: "create_entity",
      args: { name: "user_preferences", entity_type: "Preference", properties: { theme: "dark" } }
    },
    {
      prompt: "What did we discuss about machine learning?",
      action: "semantic_search",
      args: { query: "machine learning" }
    },
    {
      prompt: "Create a session for our conversation",
      action: "create_session",
      args: { agent_id: "main-assistant" }
    }
  ]
});

export default agentMemorySkill;