"""
Agent Memory - Without vs With Memory Comparison
==============================================

This demo shows the fundamental difference between AI agents
with and without persistent memory.

REQUIREMENTS:
- Python 3.8+
- openai package (pip install openai)
- agentmemory package (pip install agentmemory)

SETUP:
1. Start Agent Memory server: ./server
2. Set environment variables for your API keys
"""

print("=" * 60)
print("AGENT MEMORY DEMO: Without vs With Memory")
print("=" * 60)

# ============================================================
# PART 1: WITHOUT MEMORY - Traditional AI Conversation
# ============================================================

print("\n" + "=" * 60)
print("PART 1: WITHOUT MEMORY (Traditional AI)")
print("=" * 60)

# Simulated responses showing the problem
without_memory_conversation = """
User: Hi, my name is John and I'm a software engineer at Acme Inc.

Assistant: Hello John! It's nice to meet you. How can I help you today?

---

User: What is my name and where do I work?

Assistant: I don't have access to that information. You haven't told me 
your name or workplace in this conversation.

---

User: Remember, I mentioned I work at Acme Inc as a software engineer?

Assistant: Thank you for that information! I've noted that you work at 
Acme Inc as a software engineer.

---

User: What did I just tell you about my job?

Assistant: You told me that you work at Acme Inc as a software engineer.
(I can only remember within the current conversation)

---

[NEW CONVERSATION STARTS - ALL CONTEXT LOST]

User: Hi, do you remember what I do for a living?

Assistant: I don't have any information about your occupation. 
We haven't discussed that topic yet. 
(Starting completely fresh - forgot everything)
"""

print(without_memory_conversation)

print("\nKEY LIMITATIONS WITHOUT MEMORY:")
print("-" * 40)
print("❌ Forgets everything between conversations")
print("❌ Cannot share context between agents")
print("❌ No semantic search of past conversations")
print("❌ No knowledge graph of entities/facts")
print("❌ No skill extraction and reuse")
print("❌ Must re-explain context every time")


# ============================================================
# PART 2: WITH MEMORY - AI + Agent Memory
# ============================================================

print("\n\n" + "=" * 60)
print("PART 2: WITH MEMORY (Agent Memory)")
print("=" * 60)

with_memory_conversation = """
User: Hi, my name is John and I'm a software engineer at Acme Inc.

Assistant: Hello John! Nice to meet you. I'll remember that you're 
a software engineer at Acme Inc.

[MEMORY STORED: user:John, company:Acme Inc, role:Software Engineer]

---

User: What is my name and where do I work?

Assistant: Your name is John and you work at Acme Inc as a software engineer.
[RECALLED FROM MEMORY]

---

User: What did I tell you about my job?

Assistant: You told me you're a software engineer at Acme Inc.
[LOOKED UP IN KNOWLEDGE GRAPH]

---

[NEW CONVERSATION STARTS - MEMORY PERSISTS]

User: Hi, do you remember what I do for a living?

Assistant: Yes! You're John, a software engineer at Acme Inc.
[CONTEXT RECALLED FROM PREVIOUS SESSIONS]

---

User: Can you search for all conversations about Acme Inc?

Assistant: Found 3 relevant memories:
1. "John works at Acme Inc as a software engineer"
2. "John mentioned Acme Inc has 50 employees"
3. "John is interested in AI/ML projects at Acme"
[SEMANTIC SEARCH ACROSS ALL HISTORY]
"""

print(with_memory_conversation)

print("\nCAPABILITIES WITH MEMORY:")
print("-" * 40)
print("✓ Persistent memory across conversations")
print("✓ Semantic search of all past interactions")
print("✓ Knowledge graph with entities and relationships")
print("✓ Multi-agent shared memory")
print("✓ Skill extraction and reuse")
print("✓ Context compression (85% storage reduction)")


# ============================================================
# PART 3: CODE COMPARISON
# ============================================================

print("\n\n" + "=" * 60)
print("PART 3: CODE COMPARISON")
print("=" * 60)

print("""
WITHOUT MEMORY:
--------------
messages = [{"role": "user", "content": "Hi, I'm John..."}]
response = openai.chat.completions.create(model="gpt-4", messages=messages)
# That's it - no persistence, no search, no context

WITH MEMORY:
------------
# 1. Store conversation
client.add_message(session_id, "user", "Hi, I'm John at Acme...")

# 2. Query anytime
results = client.semantic_search("What does John do?")
# Returns: [{"text": "John works at Acme Inc...", "score": 0.95}]

# 3. Get facts
entities = client.get_entities(type="person", name="John")
# Returns: {"name": "John", "role": "Engineer", "company": "Acme"}

# 4. Multi-agent sharing
client.share_memory_to_group(group_id, memory_id)
# Other agents can access the context
""")


# ============================================================
# PART 4: LIVE TEST (requires server)
# ============================================================

print("\n\n" + "=" * 60)
print("PART 4: HOW TO TEST WITH LIVE SERVER")
print("=" * 60)

print("""
To run the live demo:

1. START SERVICES (Docker):
   docker run -d --name neo4j -p 7474:7474 -p 7687:7687 neo4j
   docker run -d --name qdrant -p 6333:6333 qdrant/qdrant

2. SET ENVIRONMENT:
   export OPENAI_API_KEY="sk-..."
   export NEO4J_PASSWORD="your-password"

3. START SERVER:
   ./server

4. RUN PYTHON CLIENT:
   python examples/with_memory.py
""")


# ============================================================
# PART 5: RUN THE TESTS
# ============================================================

print("\n\n" + "=" * 60)
print("RUNNING COMPARISON TEST")
print("=" * 60)

# Check if we can import the libraries
try:
    import openai

    print("\n✓ OpenAI library available")
except ImportError:
    print("\n✗ OpenAI not installed: pip install openai")

try:
    # Try to import agentmemory
    import agentmemory

    print("✓ AgentMemory library available")
    HAS_AGENTMEMORY = True
except ImportError:
    print("✗ AgentMemory not installed: pip install agentmemory")
    HAS_AGENTMEMORY = False

print("\n" + "=" * 60)
print("TEST SUMMARY")
print("=" * 60)

if HAS_AGENTMEMORY:
    print("""
You have AgentMemory installed. To test:

1. Start Neo4j:    docker run -d -p 7474:7474 -p 7687:7687 neo4j
2. Start Qdrant:   docker run -d -p 6333:6333 qdrant/qdrant  
3. Start Server:    ./server
4. Run Example:     python sdk/python/examples/with_memory.py
""")
else:
    print("""
To test with live AgentMemory:

1. Install dependencies:
   pip install openai agentmemory

2. Start required services:
   - Neo4j (graph database)
   - Qdrant (vector database)
   - OpenAI API key

3. Start the server:
   ./server

4. Run the examples:
   python examples/without_memory.py
   python examples/with_memory.py
""")

print("\nKEY DIFFERENCE:")
print("-" * 40)
print("WITHOUT MEMORY:  Each conversation starts fresh")
print("WITH MEMORY:     Conversations remember everything")
