"""
AutoGen Integration for Agent Memory - Python SDK

Provides shared memory capabilities for AutoGen multi-agent systems.

Example:
    >>> from agentmemory.integrations.autogen import AutoGenMemory
    >>>
    >>> memory = AutoGenMemory(
    ...     group_id='research-team',
    ...     user_id='user-123',
    ...     base_url='http://localhost:8080'
    ... )
    >>>
    >>> # All agents can share memories
    >>> memory.add_shared_memory('Research shows AI will transform healthcare')
    >>> memories = memory.get_shared_memories()
"""

from typing import Any, Dict, List, Optional
from agentmemory import AgentMemory


class AutoGenMemoryConfig:
    """Configuration for AutoGen memory."""

    def __init__(
        self,
        group_id: str,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
    ):
        """
        Initialize AutoGen memory config.

        Args:
            group_id: Group ID for shared memories
            user_id: Optional user ID
            org_id: Optional organization ID
            base_url: Base URL of the Agent Memory API
            api_key: Optional API key
        """
        self.group_id = group_id
        self.user_id = user_id
        self.org_id = org_id
        self.base_url = base_url
        self.api_key = api_key


class AgentContext:
    """Context for an AutoGen agent."""

    def __init__(
        self,
        agent_id: str,
        role: Optional[str] = None,
        goal: Optional[str] = None,
    ):
        """
        Initialize agent context.

        Args:
            agent_id: Unique agent identifier
            role: Optional role description
            goal: Optional goal description
        """
        self.agent_id = agent_id
        self.role = role
        self.goal = goal

    def to_dict(self) -> Dict[str, Any]:
        return {
            "agent_id": self.agent_id,
            "role": self.role,
            "goal": self.goal,
        }


class AutoGenMemory:
    """Shared memory for AutoGen multi-agent systems."""

    def __init__(self, config: AutoGenMemoryConfig):
        """
        Initialize the shared memory.

        Args:
            config: AutoGenMemoryConfig with group_id and optional user/org IDs
        """
        self.client = AgentMemory(
            base_url=config.base_url,
            api_key=config.api_key,
        )
        self.group_id = config.group_id
        self.user_id = config.user_id
        self.org_id = config.org_id

    def add_shared_memory(
        self,
        content: str,
        category: str = "agent-shared",
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Add a shared memory visible to all agents in the group.

        Args:
            content: Memory content to store
            category: Optional category (default: 'agent-shared')
            metadata: Optional additional metadata

        Returns:
            Created memory dict
        """
        meta = metadata or {}
        meta["group_id"] = self.group_id
        meta["shared"] = True

        return self.client.create_memory(
            content=content,
            memory_type="org",
            category=category,
            user_id=self.user_id,
            org_id=self.org_id,
            metadata=meta,
        )

    def get_shared_memories(
        self,
        category: Optional[str] = None,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """
        Get all shared memories for this group.

        Args:
            category: Optional category filter
            limit: Maximum number of results

        Returns:
            List of shared memories
        """
        result = self.client.list_memories(
            user_id=self.user_id,
            org_id=self.org_id,
        )

        memories = result.get("memories", [])

        filtered = []
        for m in memories:
            meta = m.get("metadata") or {}
            if meta.get("group_id") == self.group_id and meta.get("shared") is True:
                if category is None or m.get("category") == category:
                    filtered.append(m)

        return filtered[:limit]

    def search_shared_memories(
        self,
        query: str,
        limit: int = 10,
        threshold: float = 0.5,
    ) -> List[Dict[str, Any]]:
        """
        Search shared memories across the group.

        Args:
            query: Search query
            limit: Maximum number of results
            threshold: Similarity threshold

        Returns:
            List of matching memory results
        """
        results = self.client.search(
            query=query,
            limit=limit,
            threshold=threshold,
            user_id=self.user_id,
            org_id=self.org_id,
        )

        filtered = []
        for r in results:
            meta = r.get("metadata", {}).get("metadata") if r.get("metadata") else {}
            if meta.get("group_id") == self.group_id and meta.get("shared") is True:
                filtered.append(r)

        return filtered

    def get_agent_memory(
        self, agent_id: str, agent_context: Optional[AgentContext] = None
    ) -> "AutoGenAgentMemory":
        """
        Get a memory agent for a specific AutoGen agent.

        Args:
            agent_id: Agent identifier
            agent_context: Optional agent context

        Returns:
            AutoGenAgentMemory instance
        """
        return AutoGenAgentMemory(
            client=self.client,
            agent_id=agent_id,
            group_id=self.group_id,
            agent_context=agent_context,
            user_id=self.user_id,
            org_id=self.org_id,
        )


class AutoGenAgentMemory:
    """Agent-specific memory for AutoGen."""

    def __init__(
        self,
        client: AgentMemory,
        agent_id: str,
        group_id: str,
        agent_context: Optional[AgentContext] = None,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
    ):
        """
        Initialize agent-specific memory.

        Args:
            client: AgentMemory client instance
            agent_id: Agent identifier
            group_id: Group ID
            agent_context: Optional agent context
            user_id: Optional user ID
            org_id: Optional organization ID
        """
        self.client = client
        self.agent_id = agent_id
        self.group_id = group_id
        self.agent_context = agent_context
        self.user_id = user_id
        self.org_id = org_id

    def add_memory(
        self,
        content: str,
        memory_type: str = "user",
        category: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        Add an agent-specific memory.

        Args:
            content: Memory content
            memory_type: Memory type (default: 'user')
            category: Optional category

        Returns:
            Created memory dict
        """
        meta = {
            "group_id": self.group_id,
        }
        if self.agent_context:
            meta["agent_context"] = self.agent_context.to_dict()

        return self.client.create_memory(
            content=content,
            memory_type=memory_type,
            category=category,
            user_id=self.user_id,
            org_id=self.org_id,
            agent_id=self.agent_id,
            metadata=meta,
        )

    def get_memories(self, limit: int = 50) -> List[Dict[str, Any]]:
        """
        Get all memories for this agent.

        Args:
            limit: Maximum number of results

        Returns:
            List of agent memories
        """
        result = self.client.list_memories(
            user_id=self.user_id,
            org_id=self.org_id,
        )

        memories = result.get("memories", [])
        return [m for m in memories if m.get("agent_id") == self.agent_id][:limit]

    def search(
        self, query: str, limit: int = 10, threshold: float = 0.5
    ) -> List[Dict[str, Any]]:
        """
        Search agent memories.

        Args:
            query: Search query
            limit: Maximum number of results
            threshold: Similarity threshold

        Returns:
            List of matching memory results
        """
        return self.client.search(
            query=query,
            limit=limit,
            threshold=threshold,
            user_id=self.user_id,
            org_id=self.org_id,
            agent_id=self.agent_id,
        )

    def get_shared_memories(self, limit: int = 50) -> List[Dict[str, Any]]:
        """
        Get shared memories visible to this agent.

        Args:
            limit: Maximum number of results

        Returns:
            List of shared memories
        """
        result = self.client.list_memories(
            user_id=self.user_id,
            org_id=self.org_id,
        )

        memories = result.get("memories", [])
        filtered = []
        for m in memories:
            meta = m.get("metadata") or {}
            if meta.get("group_id") == self.group_id and meta.get("shared") is True:
                filtered.append(m)

        return filtered[:limit]

    def add_feedback(
        self,
        memory_id: str,
        feedback_type: str = "positive",
        comment: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        Add feedback to improve future searches.

        Args:
            memory_id: Memory ID to feedback on
            feedback_type: Feedback type (positive, negative, very_negative)
            comment: Optional comment

        Returns:
            Feedback result dict
        """
        return self.client.add_feedback(
            memory_id=memory_id,
            feedback_type=feedback_type,
            comment=comment,
            user_id=self.user_id,
        )


__all__ = [
    "AutoGenMemory",
    "AutoGenAgentMemory",
    "AutoGenMemoryConfig",
    "AgentContext",
]
