"""
Hystersis Integrations

This package provides integrations with popular AI frameworks:
- LangChain: Memory components and retrievers
- LangGraph: Memory nodes for LangGraph workflows
- LlamaIndex: Reader, index, and query engine components
- CrewAI: Shared memory for multi-agent crews
- AutoGen: Shared memory for AutoGen multi-agent systems
- OpenAI Agents: Memory tools for OpenAI Agents SDK
- Google ADK: Memory tools for Google Agent Development Kit
- Pydantic AI: Type-safe memory dependency injection

Example:
    from hystersis.integrations.langchain import HystersisMemory
    from hystersis.integrations.llamaindex import HystersisIndex
    from hystersis.integrations.crewai import CrewMemory
    from hystersis.integrations.langgraph import HystersisChecker
    from hystersis.integrations.autogen import AutoGenMemory
    from hystersis.integrations.openai_agents import HystersisOpenAITools
    from hystersis.integrations.google_adk import HystersisGoogleADKTool
    from hystersis.integrations.pydantic_ai import HystersisMemoryDeps

    # LangChain
    memory = HystersisMemory(session_id="user-123")

    # LlamaIndex
    reader = HystersisReader(user_id="user-123")
    documents = reader.load_data(query="AI projects")

    # CrewAI
    crew_memory = CrewMemory(crew_id="research-crew", user_id="user-123")

    # LangGraph
    checker = HystersisChecker(user_id="user-123")

    # AutoGen
    autogen_memory = AutoGenMemory(group_id="research-team", user_id="user-123")

    # OpenAI Agents
    tools = HystersisOpenAITools(base_url="http://localhost:8080", api_key="key")

    # Google ADK
    adk_tool = HystersisGoogleADKTool(base_url="http://localhost:8080", api_key="key")

    # Pydantic AI
    deps = HystersisMemoryDeps(base_url="http://localhost:8080", api_key="key")
"""

from hystersis.integrations.langchain import (
    HystersisMemory,
    HystersisRetriever,
    HystersisVectorStore,
)

from hystersis.integrations.llamaindex import (
    HystersisReader,
    HystersisIndex,
    HystersisQueryEngine,
    HystersisMemoryStore,
)

from hystersis.integrations.crewai import (
    CrewMemory,
)

from hystersis.integrations.langgraph import (
    HystersisChecker,
    HystersisUpdater,
    HystersisNode,
)

from hystersis.integrations.autogen import (
    AutoGenMemory,
    AutoGenHystersis,
)

from hystersis.integrations.openai_agents import (
    HystersisOpenAITools,
)

from hystersis.integrations.google_adk import (
    HystersisGoogleADKTool,
)

from hystersis.integrations.pydantic_ai import (
    HystersisMemoryDeps,
)

__all__ = [
    # LangChain
    "HystersisMemory",
    "HystersisRetriever",
    "HystersisVectorStore",
    # LlamaIndex
    "HystersisReader",
    "HystersisIndex",
    "HystersisQueryEngine",
    "HystersisMemoryStore",
    # CrewAI
    "CrewMemory",
    # LangGraph
    "HystersisChecker",
    "HystersisUpdater",
    "HystersisNode",
    # AutoGen
    "AutoGenMemory",
    "AutoGenHystersis",
    # OpenAI Agents
    "HystersisOpenAITools",
    # Google ADK
    "HystersisGoogleADKTool",
    # Pydantic AI
    "HystersisMemoryDeps",
]
