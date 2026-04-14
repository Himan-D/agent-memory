#!/usr/bin/env python3
"""
Agent Memory - Without vs With Memory Comparison
==============================================

This script demonstrates the difference between AI agents
with and without persistent memory.
"""

import requests
import json
import time

BASE_URL = "http://localhost:8080"


def print_section(title):
    print("\n" + "=" * 60)
    print(title)
    print("=" * 60)


def print_step(step, description):
    print(f"\n{step}. {description}")


# ============================================================
# PART 1: WITHOUT MEMORY (Traditional AI Conversation)
# ============================================================

print_section("PART 1: WITHOUT MEMORY (Traditional AI)")

print("""
In a traditional AI setup WITHOUT memory:

1. User: "Hi, my name is John and I work at Acme Corp."
   Assistant: "Nice to meet you, John! How can I help you today?"

2. User: "What company do I work for?"
   Assistant: "I don't have access to information about your 
              employment unless you tell me."

3. [NEW CONVERSATION STARTS - ALL CONTEXT LOST]

4. User: "Hi, do you remember what I do for a living?"
   Assistant: "I don't have any information about your occupation.
              We haven't discussed that topic yet."

LIMITATIONS:
❌ Forgets everything between conversations
❌ Cannot share context between agents
❌ No semantic search of past conversations
❌ No knowledge graph of entities/facts
❌ Must re-explain context every time
""")

time.sleep(1)

# ============================================================
# PART 2: WITH AGENT MEMORY
# ============================================================

print_section("PART 2: WITH AGENT MEMORY")

print("\nConnecting to Agent Memory server...")

# Test health
response = requests.get(f"{BASE_URL}/health")
print(f"Health check: {response.json()}")

# Step 1: Create Agent
print_step(1, "Create Agent")
response = requests.post(f"{BASE_URL}/agents", json={"name": "assistant"})
agent = response.json()
print(f"Agent created: {agent['name']} (ID: {agent['id'][:8]}...)")
AGENT_ID = agent["id"]

# Step 2: Create Session
print_step(2, "Create Session")
response = requests.post(f"{BASE_URL}/sessions", json={"agent_id": AGENT_ID})
session = response.json()
print(f"Session created (ID: {session['id'][:8]}...)")
SESSION_ID = session["id"]

# Step 3: Add conversation messages
print_step(3, "Add Conversation Messages")

messages = [
    ("user", "Hi, my name is John and I work at Acme Corp as a software engineer."),
    (
        "assistant",
        "Nice to meet you, John! I'll remember that you work at Acme Corp as a software engineer.",
    ),
    (
        "user",
        "I'm currently working on a machine learning project using Python and PyTorch.",
    ),
    (
        "assistant",
        "That sounds interesting! A ML project with PyTorch - I'll note that for future reference.",
    ),
    ("user", "What's my name and where do I work?"),
]

for role, content in messages:
    response = requests.post(
        f"{BASE_URL}/sessions/{SESSION_ID}/messages",
        json={"role": role, "content": content},
    )
    print(f"  Added {role}: {content[:50]}...")

time.sleep(0.5)

# Step 4: Retrieve context
print_step(4, "Retrieve Session Context (Memory)")
response = requests.get(f"{BASE_URL}/sessions/{SESSION_ID}/context")
context = response.json() or []
print(f"\nRetrieved {len(context)} messages from memory:")
for msg in context:
    print(f"  [{msg['role']}] {msg['content'][:60]}...")

# Step 5: Create entities
print_step(5, "Create Knowledge Graph Entities")
entities = [
    {
        "name": "John",
        "type": "person",
        "properties": {"role": "software engineer", "company": "Acme Corp"},
    },
    {"name": "Acme Corp", "type": "organization", "properties": {"industry": "tech"}},
]

for entity_data in entities:
    response = requests.post(f"{BASE_URL}/entities", json=entity_data)
    if response.status_code == 200:
        entity = response.json()
        print(
            f"  Created entity: {entity['name']} (type: {entity.get('type', entity.get('entity_type'))})"
        )

time.sleep(0.5)

# Step 6: List entities
print_step(6, "Query Knowledge Graph")
response = requests.get(f"{BASE_URL}/entities")
entities_list = response.json()
print(
    f"  Found {entities_list.get('total', len(entities_list.get('entities', [])))} entities"
)

# Step 7: Create Agent Group (Multi-agent sharing)
print_step(7, "Create Agent Group (Multi-Agent)")
response = requests.post(
    f"{BASE_URL}/groups",
    json={"name": "engineering-team", "policy": {"shared_memory": True}},
)
group = response.json()
print(f"  Created group: {group['name']} (ID: {group['id'][:8]}...)")

# Step 8: Search
print_step(8, "Semantic Search")
search_queries = ["John", "machine learning", "PyTorch"]

for query in search_queries:
    response = requests.post(f"{BASE_URL}/search", json={"query": query, "limit": 3})
    if response.status_code == 200:
        results = response.json()
        if results:
            print(f"  Query '{query}': Found {len(results)} results")
            for r in results[:2]:
                text = r.get("text", r.get("content", str(r)))[:50]
                print(f"    - {text}...")

# ============================================================
# SUMMARY
# ============================================================

print_section("SUMMARY: WHAT AGENT MEMORY ENABLES")

print("""
CAPABILITIES WITH MEMORY:
✓ Persistent memory across conversations
✓ Session-based message history
✓ Semantic/vector search of all past interactions
✓ Knowledge graph with entities and relationships
✓ Multi-agent shared memory pools
✓ Agent groups with pub/sub sync
✓ Skill extraction and synthesis
✓ Context compression (85% storage reduction)

HOW IT WORKS:
1. All messages → stored in session + indexed in vector DB
2. Entities → extracted and stored in knowledge graph (Neo4j)
3. Semantic search → queries vector DB (Qdrant)
4. Multi-agent → Redis pub/sub for real-time sync
5. Skills → extracted from interactions, reusable across agents
""")

print("\nTest complete!")
