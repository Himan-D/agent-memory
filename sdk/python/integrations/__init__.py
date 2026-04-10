"""
Agent Memory Integrations

This package provides integrations with popular AI frameworks:
- LangChain: Memory components and retrievers
- LlamaIndex: Reader, index, and query engine components
- CrewAI: Shared memory for multi-agent crews

Example:
    from agentmemory.integrations.langchain import AgentMemoryMemory
    from agentmemory.integrations.llamaindex import AgentMemoryIndex
    from agentmemory.integrations.crewai import CrewMemory

    # LangChain
    memory = AgentMemoryMemory(session_id="user-123")

    # LlamaIndex
    reader = AgentMemoryReader(user_id="user-123")
    documents = reader.load_data(query="AI projects")

    # CrewAI
    crew_memory = CrewMemory(crew_id="research-crew", user_id="user-123")
"""

from agentmemory.integrations.langchain import (
    AgentMemoryMemory,
    AgentMemoryRetriever,
    AgentMemoryVectorStore,
)

from agentmemory.integrations.llamaindex import (
    AgentMemoryReader,
    AgentMemoryIndex,
    AgentMemoryRetriever,
    AgentMemoryQueryEngine,
    AgentMemoryMemoryStore,
)

from agentmemory.integrations.crewai import (
    CrewMemory,
    AgentMemory,
)

__all__ = [
    "AgentMemoryMemory",
    "AgentMemoryRetriever",
    "AgentMemoryVectorStore",
    "AgentMemoryReader",
    "AgentMemoryIndex",
    "AgentMemoryQueryEngine",
    "AgentMemoryMemoryStore",
    "CrewMemory",
]
