"""
Example: Basic Usage

Simple example of using Agent Memory to store and retrieve conversations.
"""

from agentmemory import AgentMemory

# Initialize client
client = AgentMemory(base_url="http://localhost:8080", api_key="test-key")

# Check health
print("Health:", client.health())

# Create a session for an agent
session = client.create_session(
    agent_id="my-assistant", metadata={"user_id": "user-123"}
)
print(f"Created session: {session['id']}")

# Add conversation messages
client.add_message(session["id"], "user", "Hello! I'm interested in machine learning.")
client.add_message(session["id"], "assistant", "Great! What aspect interests you most?")
client.add_message(session["id"], "user", "Neural networks and deep learning")

# Get conversation history
messages = client.get_messages(session["id"])
print("\nConversation:")
for msg in messages:
    print(f"  {msg['role']}: {msg['content']}")

# Semantic search
results = client.search("deep learning neural networks")
print("\nSearch results:")
for r in results:
    print(f"  Score: {r.get('score', 0):.2f}")
