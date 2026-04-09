"""
Example: Knowledge Graph

Build a knowledge graph with entities and relationships.
"""

from agentmemory import AgentMemory

client = AgentMemory(base_url="http://localhost:8080", api_key="test-key")

# Create entities for a research knowledge graph
print("Creating entities...")

paper = client.create_entity(
    name="Attention Is All You Need",
    type="Paper",
    properties={"year": 2017, "authors": "Vaswani et al.", "citations": 50000},
)
print(f"Created paper: {paper['id']}")

transformer = client.create_entity(
    name="Transformer",
    type="Architecture",
    properties={"key_innovation": "Self-attention mechanism"},
)
print(f"Created architecture: {transformer['id']}")

attention = client.create_entity(
    name="Self-Attention",
    type="Concept",
    properties={"definition": "Mechanism to relate different positions of a sequence"},
)
print(f"Created concept: {attention['id']}")

bert = client.create_entity(
    name="BERT",
    type="Model",
    properties={"year": 2018, "framework": "transformer-based"},
)
print(f"Created model: {bert['id']}")

# Create relationships
print("\nCreating relationships...")
client.create_relation(paper["id"], transformer["id"], "INTRODUCES")
client.create_relation(transformer["id"], attention["id"], "USES")
client.create_relation(bert["id"], transformer["id"], "BASED_ON")
client.create_relation(bert["id"], attention["id"], "USES")
print("Created 4 relationships")

# Query the graph
print("\nRelationships for Transformer:")
relations = client.get_entity_relations(transformer["id"])
for rel in relations:
    print(f"  {rel['type']} -> {rel['to_id']}")

# Semantic search for related concepts
print("\nSearching for attention-related concepts:")
results = client.search("self-attention mechanism", limit=3)
for r in results:
    print(f"  {r.get('score', 0):.2f} - {r.get('entity', {}).get('name', 'N/A')}")
