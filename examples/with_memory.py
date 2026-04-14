# With Agent Memory - AI Agent with Persistent Memory

from agentmemory import AgentMemory

client = AgentMemory("https://api.yourserver.com", api_key="your-key")

# 1. Create agent and session
agent = client.create_agent(name="assistant", config={})
session = client.create_session(agent_id=agent["id"])

# 2. Add conversation with context
client.add_message(
    session["id"],
    "user",
    "My name is John and I work at Acme Corp as a senior engineer.",
)
client.add_message(
    session["id"],
    "assistant",
    "Nice to meet you, John! A senior engineer at Acme Corp - that's great.",
)

# 3. Later conversation - AGENT REMEMBERS
client.add_message(
    session["id"], "user", "What company do I work for and what's my role?"
)

# 4. Semantic search to recall relevant context
results = client.semantic_search("John work Acme engineer")
print(results)
# Output: Found relevant memory about John working at Acme Corp as senior engineer

# 5. Query knowledge graph for facts
facts = client.get_entities(entity_type="person", name="John")
print(facts)
# Output: {"name": "John", "company": "Acme Corp", "role": "Senior Engineer"}

# 6. Multi-agent: Share memory with team
group = client.create_agent_group(name="engineering-team")
client.share_memory_to_group(group_id=group["id"], memory_id=session["id"])

# 7. Extract and reuse skills
skill = client.extract_skills(
    agent_id=agent["id"],
    interaction="User asked about company policy, bot explained PTO rules",
)
# Now other agents can use this skill
