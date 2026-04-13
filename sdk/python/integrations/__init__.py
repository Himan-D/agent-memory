"""
Agent Memory Integrations

This package provides integrations with popular AI frameworks:
- LangChain: Memory components and retrievers
- LangGraph: Memory nodes for LangGraph workflows
- LlamaIndex: Reader, index, and query engine components
- CrewAI: Shared memory for multi-agent crews
- AutoGen: Shared memory for AutoGen multi-agent systems

Example:
    from agentmemory.integrations.langchain import AgentMemoryMemory
    from agentmemory.integrations.llamaindex import AgentMemoryIndex
    from agentmemory.integrations.crewai import CrewMemory
    from agentmemory.integrations.langgraph import AgentMemoryChecker
    from agentmemory.integrations.autogen import AutoGenMemory

    # LangChain
    memory = AgentMemoryMemory(session_id="user-123")

    # LlamaIndex
    reader = AgentMemoryReader(user_id="user-123")
    documents = reader.load_data(query="AI projects")

    # CrewAI
    crew_memory = CrewMemory(crew_id="research-crew", user_id="user-123")

    # LangGraph
    checker = AgentMemoryChecker(user_id="user-123")

    # AutoGen
    autogen_memory = AutoGenMemory(group_id="research-team", user_id="user-123")
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

from agentmemory.integrations.langgraph import (
    AgentMemoryChecker,
    AgentMemoryUpdater,
    AgentMemoryNode,
)

from agentmemory.integrations.autogen import (
    AutoGenMemory,
    AutoGenAgentMemory,
)

__all__ = [
    # LangChain
    "AgentMemoryMemory",
    "AgentMemoryRetriever",
    "AgentMemoryVectorStore",
    # LlamaIndex
    "AgentMemoryReader",
    "AgentMemoryIndex",
    "AgentMemoryQueryEngine",
    "AgentMemoryMemoryStore",
    # CrewAI
    "CrewMemory",
    # LangGraph
    "AgentMemoryChecker",
    "AgentMemoryUpdater",
    "AgentMemoryNode",
    # AutoGen
    "AutoGenMemory",
    "AutoGenAgentMemory",
]
