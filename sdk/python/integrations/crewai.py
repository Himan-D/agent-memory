"""
CrewAI Integration for Agent Memory

This module provides a CrewAI-compatible memory component that enables
shared memory across multiple agents in a crew.

Usage:
    from agentmemory.integrations.crewai import CrewMemory

    # Create shared memory for the crew
    crew_memory = CrewMemory(
        crew_id="research-crew",
        user_id="user-123",
        base_url="http://localhost:8080"
    )

    # Each agent can access and contribute to shared memory
    agent = Agent(
        role="Researcher",
        goal="Find and summarize AI papers",
        memory=crew_memory.get_agent_memory("researcher-agent")
    )
"""

from typing import Any, Dict, List, Optional, Union
from datetime import datetime


class CrewMemory:
    """
    Shared memory for CrewAI crews.

    Enables multiple agents to share memories across a crew, with support
    for both individual agent memories and crew-wide shared memories.

    Attributes:
        crew_id: Unique identifier for the crew
        user_id: User identifier for the memory namespace
        base_url: Base URL of the Agent Memory API
        api_key: API key for authentication

    Example:
        >>> crew_memory = CrewMemory(
        ...     crew_id="my-crew",
        ...     user_id="user-123"
        ... )
        >>> # Each agent gets their own memory view
        >>> agent_memory = crew_memory.get_agent_memory("agent-1")
        >>> # But can also access shared crew memories
        >>> crew_memory.add_shared_memory("Decision: Use RAG for this project")
    """

    def __init__(
        self,
        crew_id: str,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
    ):
        self.crew_id = crew_id
        self.user_id = user_id
        self.org_id = org_id
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key

        self._session = requests.Session()
        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make an HTTP request."""
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    def get_agent_memory(self, agent_id: str) -> "AgentMemory":
        """
        Get a memory instance for a specific agent.

        Args:
            agent_id: Unique identifier for the agent

        Returns:
            AgentMemory instance configured for this agent
        """
        return AgentMemory(
            agent_id=agent_id,
            crew_id=self.crew_id,
            user_id=self.user_id,
            org_id=self.org_id,
            base_url=self.base_url,
            api_key=self.api_key,
        )

    def add_shared_memory(
        self,
        content: str,
        category: Optional[str] = "crew-shared",
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Add a memory shared across the entire crew.

        Args:
            content: Memory content
            category: Memory category
            metadata: Additional metadata

        Returns:
            Created memory dict
        """
        payload = {
            "content": content,
            "type": "org",
            "category": category,
            "metadata": metadata or {},
        }

        if self.user_id:
            payload["user_id"] = self.user_id
        if self.org_id:
            payload["org_id"] = self.org_id

        payload["metadata"]["crew_id"] = self.crew_id
        payload["metadata"]["shared"] = True

        resp = self._request("POST", "/memories", json=payload)
        return resp.json()

    def get_shared_memories(
        self,
        category: Optional[str] = None,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """
        Get all memories shared across the crew.

        Args:
            category: Optional category filter
            limit: Maximum results

        Returns:
            List of shared memory dicts
        """
        params = {"limit": limit}

        if self.org_id:
            params["org_id"] = self.org_id
        elif self.user_id:
            params["user_id"] = self.user_id

        resp = self._request("GET", "/memories", params=params)
        result = resp.json()

        memories = result.get("memories", [])
        shared = [
            m
            for m in memories
            if m.get("metadata", {}).get("crew_id") == self.crew_id
            and m.get("metadata", {}).get("shared", False)
        ]

        if category:
            shared = [m for m in shared if m.get("category") == category]

        return shared

    def search_shared(
        self,
        query: str,
        limit: int = 10,
        threshold: float = 0.5,
    ) -> List[Dict[str, Any]]:
        """
        Search across shared crew memories.

        Args:
            query: Search query
            limit: Maximum results
            threshold: Minimum similarity score

        Returns:
            List of matching memory dicts
        """
        params = {
            "q": query,
            "limit": limit,
            "threshold": threshold,
        }

        if self.org_id:
            params["org_id"] = self.org_id
        elif self.user_id:
            params["user_id"] = self.user_id

        resp = self._request("GET", "/search", params=params)
        results = resp.json()

        shared = [
            r
            for r in results
            if r.get("metadata", {}).get("crew_id") == self.crew_id
            and r.get("metadata", {}).get("shared", False)
        ]

        return shared

    def add_feedback_to_shared(
        self,
        memory_id: str,
        feedback_type: str,
        comment: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Add feedback to a shared memory."""
        payload = {
            "memory_id": memory_id,
            "type": feedback_type,
        }
        if comment:
            payload["comment"] = comment

        resp = self._request("POST", "/feedback", json=payload)
        return resp.json()


class AgentMemory:
    """
    Agent-specific memory for CrewAI agents.

    Each agent in a crew gets their own memory namespace but can
    also access shared crew memories.

    Attributes:
        agent_id: Unique identifier for the agent
        crew_id: Crew identifier for shared memory access
        user_id: User identifier
        org_id: Organization identifier
        base_url: Base URL of the Agent Memory API
        api_key: API key for authentication

    Example:
        >>> agent_memory = AgentMemory(
        ...     agent_id="researcher-1",
        ...     crew_id="my-crew",
        ...     user_id="user-123"
        ... )
        >>> # Store agent-specific memory
        >>> agent_memory.add_memory("Found a great paper on transformers")
        >>> # Access shared crew memories
        >>> shared = agent_memory.get_shared_memories()
    """

    def __init__(
        self,
        agent_id: str,
        crew_id: Optional[str] = None,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
    ):
        self.agent_id = agent_id
        self.crew_id = crew_id
        self.user_id = user_id
        self.org_id = org_id
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key

        self._session = requests.Session()
        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    def add_memory(
        self,
        content: str,
        memory_type: str = "user",
        category: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Add an agent-specific memory.

        Args:
            content: Memory content
            memory_type: Type of memory
            category: Optional category
            metadata: Additional metadata

        Returns:
            Created memory dict
        """
        payload = {
            "content": content,
            "type": memory_type,
            "agent_id": self.agent_id,
            "metadata": metadata or {},
        }

        if self.user_id:
            payload["user_id"] = self.user_id
        if self.org_id:
            payload["org_id"] = self.org_id
        if category:
            payload["category"] = category
        if self.crew_id:
            payload["metadata"]["crew_id"] = self.crew_id

        resp = self._request("POST", "/memories", json=payload)
        return resp.json()

    def get_memories(
        self,
        limit: int = 50,
        category: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """
        Get all memories for this agent.

        Args:
            limit: Maximum results
            category: Optional category filter

        Returns:
            List of memory dicts
        """
        params = {"limit": limit}

        if self.user_id:
            params["user_id"] = self.user_id
        elif self.org_id:
            params["org_id"] = self.org_id

        resp = self._request("GET", "/memories", params=params)
        result = resp.json()

        memories = result.get("memories", [])
        agent_memories = [m for m in memories if m.get("agent_id") == self.agent_id]

        if category:
            agent_memories = [
                m for m in agent_memories if m.get("category") == category
            ]

        return agent_memories

    def search(
        self,
        query: str,
        limit: int = 10,
        threshold: float = 0.5,
    ) -> List[Dict[str, Any]]:
        """
        Search agent memories.

        Args:
            query: Search query
            limit: Maximum results
            threshold: Minimum score

        Returns:
            List of matching memories
        """
        params = {
            "q": query,
            "limit": limit,
            "threshold": threshold,
            "agent_id": self.agent_id,
        }

        if self.user_id:
            params["user_id"] = self.user_id
        elif self.org_id:
            params["org_id"] = self.org_id

        resp = self._request("GET", "/search", params=params)
        return resp.json()

    def get_shared_memories(
        self,
        category: Optional[str] = None,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """
        Get shared crew memories accessible to this agent.

        Args:
            category: Optional category filter
            limit: Maximum results

        Returns:
            List of shared memory dicts
        """
        if not self.crew_id:
            return []

        params = {"limit": limit}

        if self.org_id:
            params["org_id"] = self.org_id
        elif self.user_id:
            params["user_id"] = self.user_id

        resp = self._request("GET", "/memories", params=params)
        result = resp.json()

        memories = result.get("memories", [])
        shared = [
            m
            for m in memories
            if m.get("metadata", {}).get("crew_id") == self.crew_id
            and m.get("metadata", {}).get("shared", False)
        ]

        if category:
            shared = [m for m in shared if m.get("category") == category]

        return shared

    def add_feedback(
        self,
        memory_id: str,
        feedback_type: str,
        comment: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Add feedback to a memory."""
        payload = {
            "memory_id": memory_id,
            "type": feedback_type,
        }
        if comment:
            payload["comment"] = comment

        resp = self._request("POST", "/feedback", json=payload)
        return resp.json()


import requests


__all__ = [
    "CrewMemory",
    "AgentMemory",
]
