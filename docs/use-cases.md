# Agent Memory Use Cases

Real-world examples of how to use Agent Memory in your products.

## 1. Customer Support Bot

**Problem**: Support bots repeat themselves and don't remember customer history.

**Solution**: Store every interaction and retrieve relevant context.

```python
from agentmemory import AgentMemory

client = AgentMemory("https://api.yourserver.com", api_key="support-key")

# When customer starts a new conversation
session = client.create_session(
    agent_id="support-bot",
    metadata={"customer_id": "CUST-123", "tier": "premium"}
)

# Store each interaction
client.add_message(session["id"], "user", "I can't login to my account")
client.add_message(session["id"], "assistant", "I'll help you. What error do you see?")
client.add_message(session["id"], "user", "It says 'invalid password'")

# Later - when customer returns, find similar issues
past_issues = client.semantic_search("can't login invalid password", limit=5)
# Returns similar issues from other customers
```

**Result**: Bot can say "I see similar login issues were resolved by resetting passwords..."

---

## 2. Code Assistant / Developer Tool

**Problem**: Developer agents don't understand your codebase's patterns.

**Solution**: Index code, docs, and past solutions.

```python
# Index codebase entities
client.create_entity(
    name="auth-service",
    type="Service",
    properties={"language": "python", "port": 8080}
)

client.create_entity(
    name="UserService",
    type="Class",
    properties={"file": "services/user.py", "methods": ["login", "logout"]}
)

# Create relationships
client.create_relation("auth-service", "UserService", "USES")

# Later - when developer asks about auth
results = client.semantic_search("how does user authentication work")
# Returns semantically similar code/docs
```

---

## 3. Research & Analysis Agent

**Problem**: Research agents can't connect ideas across papers and notes.

**Solution**: Build a knowledge graph of research.

```python
# Add paper as entity
paper = client.create_entity(
    name="Attention Is All You Need",
    type="Paper",
    properties={
        "authors": ["Vaswani", "Shazeer", " Parmar"],
        "year": 2017,
        "abstract": "..."
    }
)

# Add concepts
client.create_entity(name="Transformer", type="Concept")
client.create_entity(name="Self-Attention", type="Concept")
client.create_entity(name="Seq2Seq", type="Concept")

# Connect them
client.create_relation("Transformer", "Self-Attention", "USES")
client.create_relation("Transformer", "Seq2Seq", "IMPROVES")

# Find related work
related = client.semantic_search("attention mechanism neural network")
```

---

## 4. Personal AI Assistant

**Problem**: Personal assistants forget preferences and important events.

**Solution**: Store preferences, memories, and context.

```python
# Remember preferences
client.create_entity(
    name="user_preferences",
    type="Preference",
    properties={
        "coffee": "black, no sugar",
        "meetings": " mornings preferred",
        "diet": "vegetarian"
    }
)

# Remember important dates
client.create_entity(
    name="anniversary",
    type="Event",
    properties={"date": "2025-06-15", "description": "Wedding anniversary"}
)

# Store conversations about topics
session = client.create_session(agent_id="personal-assistant")
client.add_message(session["id"], "user", "I want to learn Spanish")
client.add_message(session["id"], "assistant", "Great! How about we start with basics?")

# Later - search preferences
prefs = client.get_entity("user_preferences")
# Returns: coffee preferences, meeting times, etc.
```

---

## 5. Multi-Tenant SaaS Application

**Problem**: Need to separate data between customers in multi-tenant app.

**Solution**: Use tenant IDs in API keys.

```python
# Server config: API_KEYS="key1:tenant1,key2:tenant2"
# Keys: "key1" maps to tenant1, "key2" maps to tenant2

# When customer A makes requests with their key
client_a = AgentMemory("https://api.com", api_key="key1")
session_a = client_a.create_session(agent_id="my-agent")
# Session is automatically tagged with tenant1

# Customer B with different key
client_b = AgentMemory("https://api.com", api_key="key2")
# Their data is completely isolated from customer A
```

---

## 6. Sales Intelligence

**Problem**: Sales bots don't remember deal history or customer relationships.

**Solution**: Build a relationship graph of accounts.

```python
# Add companies
acme = client.create_entity(name="Acme Corp", type="Company")
competitor = client.create_entity(name="Globex", type="Company")

# Add contacts
alice = client.create_entity(name="Alice (Acme CTO)", type="Person")
bob = client.create_entity(name="Bob (Acme VP)", type="Person")

# Create relationships
client.create_relation("acme", "alice", "HAS_EMPLOYEE")
client.create_relation("acme", "bob", "HAS_EMPLOYEE")
client.create_relation("alice", "bob", "REPORTS_TO")
client.create_relation("acme", "competitor", "COMPETES_WITH")

# Query: Who do we know at Acme?
contacts = client.get_entity_relations("acme", "HAS_EMPLOYEE")
```

---

## 7. Educational Tutoring

**Problem**: Tutors don't adapt to student progress or remember past lessons.

**Solution**: Track learning progress and adapt.

```python
# Store student profile
student = client.create_entity(
    name="student-123",
    type="Student",
    properties={"level": "intermediate", "topics_covered": ["algebra", "geometry"]}
)

# Add lesson notes
client.create_entity(
    name="lesson-2024-01-15",
    type="Lesson",
    properties={"topic": "calculus", "duration": "60min", "difficulty": "hard"}
)

# Connect student to lesson
client.create_relation("student-123", "lesson-2024-01-15", "COMPLETED")

# Query: What topics should we review?
# Semantic search for topics marked as "difficult" in past lessons
```

---

## 8. Game AI Companion

**Problem**: Game NPCs don't remember player interactions or learn from them.

**Solution**: Store player history and preferences.

```python
# Remember player decisions
session = client.create_session(agent_id="npc-companion")
client.add_message(session["id"], "player", "I'll take the healing potion")
client.add_message(session["id"], "npc", "Good choice! It could save you later")

# Store character preferences
client.create_entity(
    name="player-quest-log",
    type="QuestLog",
    properties={"completed": ["dragon_quest", "forest_quest"], "active": "castle_quest"}
)

# Remember player behavior patterns
results = client.semantic_search("attacked enemies without caution")
# NPC can adapt behavior based on past patterns
```

---

## Summary

| Use Case | Memory Type | Key Benefit |
|----------|-------------|--------------|
| Support Bot | Messages + Semantic | Remember past issues |
| Code Assistant | Entities + Relations | Understand codebase |
| Research Agent | Graph + Vectors | Connect ideas |
| Personal AI | All types | Remember preferences |
| SaaS | Tenant isolation | Multi-tenant security |
| Sales | Graph relationships | Account mapping |
| Tutoring | Session + Entities | Track progress |
| Gaming | Messages | Learn player behavior |

**Next Steps**: Check out the [Quick Start Guide](./QUICKSTART.md) to begin building.