"""
Agent Memory Python SDK

A Python client for the Agent Memory System API.
Give your AI agents persistent memory with graph relationships and semantic search.

Example:
    from agentmemory import AgentMemory

    # Connect to your agent memory server
    client = AgentMemory("https://api.yourserver.com", api_key="your-key")

    # Create a session for your agent
    session = client.create_session(agent_id="assistant-bot")

    # Add conversation messages
    client.add_message(session["id"], "user", "I love machine learning!")
    client.add_message(session["id"], "assistant", "That's great!")

    # Store a semantic memory
    memory = client.create_memory(
        content="User is interested in machine learning and AI",
        user_id="user-123",
        category="preferences"
    )

    # Later, search semantically
    results = client.search("deep learning")

    # Add feedback to improve future searches
    client.add_feedback(memory["id"], "positive")
"""

import os
from typing import Optional, List, Dict, Any, Iterator
from datetime import datetime, timedelta
import requests


class AgentMemoryError(Exception):
    """Base exception for Agent Memory errors."""

    pass


class AuthenticationError(AgentMemoryError):
    """Raised when authentication fails."""

    pass


class NotFoundError(AgentMemoryError):
    """Raised when a resource is not found."""

    pass


class ValidationError(AgentMemoryError):
    """Raised when input validation fails."""

    pass


class RateLimitError(AgentMemoryError):
    """Raised when rate limit is exceeded."""

    pass


class MemoryType:
    CONVERSATION = "conversation"
    SESSION = "session"
    USER = "user"
    ORG = "org"


class FeedbackType:
    POSITIVE = "positive"
    NEGATIVE = "negative"
    VERY_NEGATIVE = "very_negative"


class AgentMemory:
    """
    Python SDK for Agent Memory System.

    Provides a simple interface to store and retrieve agent memories,
    including conversation history, knowledge graph entities, and
    semantic search capabilities.

    Attributes:
        api_key: API key for authentication
        base_url: Base URL of the Agent Memory API

    Example:
        >>> client = AgentMemory("http://localhost:8080", api_key="test-key")
        >>> session = client.create_session(agent_id="my-bot")
        >>> client.add_message(session["id"], "user", "Hello!")
        >>> results = client.search("greetings")
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        timeout: int = 30,
    ):
        """
        Initialize the Agent Memory client.

        Args:
            base_url: Base URL of the Agent Memory API
            api_key: API key for authentication. Can also be set via
                     AGENT_MEMORY_API_KEY environment variable
            timeout: Request timeout in seconds
        """
        self.api_key = api_key or os.environ.get("AGENT_MEMORY_API_KEY", "")
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self._session = requests.Session()

        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make an HTTP request with error handling."""
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, timeout=self.timeout, **kwargs)

        if resp.status_code == 401:
            raise AuthenticationError("Invalid or missing API key")
        elif resp.status_code == 403:
            raise AuthenticationError("Admin access required")
        elif resp.status_code == 404:
            raise NotFoundError(f"Resource not found: {endpoint}")
        elif resp.status_code == 429:
            raise RateLimitError("Rate limit exceeded")
        elif resp.status_code == 400:
            raise ValidationError(resp.text)

        resp.raise_for_status()
        return resp

    def __repr__(self) -> str:
        return f"AgentMemory(base_url='{self.base_url}')"

    # ==================== Health ====================

    def health(self) -> Dict[str, str]:
        """Check API health status."""
        resp = self._request("GET", "/health")
        return resp.json()

    def ready(self) -> Dict[str, Any]:
        """Check API readiness including dependency health."""
        resp = self._request("GET", "/ready")
        return resp.json()

    # ==================== Sessions ====================

    def create_session(
        self,
        agent_id: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Create a new conversation session for an agent."""
        payload = {"agent_id": agent_id}
        if metadata:
            payload["metadata"] = metadata

        resp = self._request("POST", "/sessions", json=payload)
        return resp.json()

    def get_session(self, session_id: str) -> Dict[str, Any]:
        """Get session details and messages."""
        resp = self._request("GET", f"/sessions/{session_id}")
        return resp.json()

    def delete_session(self, session_id: str) -> Dict[str, str]:
        """Delete a session and all its messages."""
        resp = self._request("DELETE", f"/sessions/{session_id}")
        return resp.json()

    def get_messages(
        self,
        session_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """Get conversation messages for a session."""
        resp = self._request(
            "GET",
            f"/sessions/{session_id}/messages",
            params={"limit": limit},
        )
        return resp.json()

    def add_message(
        self,
        session_id: str,
        role: str,
        content: str,
    ) -> Dict[str, str]:
        """Add a message to a session conversation."""
        if role not in ("user", "assistant", "system", "tool"):
            raise ValidationError(
                f"Invalid role: {role}. Must be one of: user, assistant, system, tool"
            )

        payload = {"role": role, "content": content}
        resp = self._request(
            "POST",
            f"/sessions/{session_id}/messages",
            json=payload,
        )
        return resp.json()

    def get_context(
        self,
        session_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """Get conversation context for a session."""
        return self.get_messages(session_id, limit)

    # ==================== Memory CRUD ====================

    def create_memory(
        self,
        content: str,
        memory_type: str = MemoryType.USER,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        session_id: Optional[str] = None,
        category: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        immutable: bool = False,
        expiration_date: Optional[datetime] = None,
    ) -> Dict[str, Any]:
        """
        Create a new memory.

        Args:
            content: The memory content to store
            memory_type: Type of memory (conversation, session, user, org)
            user_id: Optional user identifier
            org_id: Optional organization identifier
            agent_id: Optional agent identifier
            session_id: Optional session identifier
            category: Optional category for organization
            metadata: Optional custom metadata
            immutable: If True, memory cannot be modified or deleted
            expiration_date: Optional expiration date for TTL

        Returns:
            Created memory dict with id, content, type, etc.

        Example:
            >>> memory = client.create_memory(
            ...     content="User prefers dark mode",
            ...     user_id="user-123",
            ...     category="preferences",
            ...     immutable=True
            ... )
        """
        if not content:
            raise ValidationError("content is required")

        payload = {
            "content": content,
            "type": memory_type,
        }

        if user_id:
            payload["user_id"] = user_id
        if org_id:
            payload["org_id"] = org_id
        if agent_id:
            payload["agent_id"] = agent_id
        if session_id:
            payload["session_id"] = session_id
        if category:
            payload["category"] = category
        if metadata:
            payload["metadata"] = metadata
        if immutable:
            payload["immutable"] = True
        if expiration_date:
            payload["expiration_date"] = expiration_date.isoformat()

        resp = self._request("POST", "/memories", json=payload)
        return resp.json()

    def get_memory(self, memory_id: str) -> Dict[str, Any]:
        """Get a specific memory by ID."""
        resp = self._request("GET", f"/memories/{memory_id}")
        return resp.json()

    def update_memory(
        self,
        memory_id: str,
        content: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Update a memory's content and/or metadata.

        Args:
            memory_id: Memory identifier
            content: New content
            metadata: Updated metadata to merge

        Returns:
            Updated memory dict
        """
        if not content:
            raise ValidationError("content is required")

        payload = {"content": content}
        if metadata:
            payload["metadata"] = metadata

        resp = self._request("PUT", f"/memories/{memory_id}", json=payload)
        return resp.json()

    def delete_memory(self, memory_id: str) -> Dict[str, str]:
        """Delete a specific memory."""
        resp = self._request("DELETE", f"/memories/{memory_id}")
        return resp.json()

    def list_memories(
        self,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        category: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        List memories with optional filters.

        Args:
            user_id: Filter by user
            org_id: Filter by organization
            agent_id: Filter by agent
            category: Filter by category

        Returns:
            Dict with memories list and count
        """
        params = {}
        if user_id:
            params["user_id"] = user_id
        if org_id:
            params["org_id"] = org_id
        if agent_id:
            params["agent_id"] = agent_id
        if category:
            params["category"] = category

        resp = self._request("GET", "/memories", params=params)
        return resp.json()

    # ==================== Batch Operations ====================

    def batch_create_memories(
        self,
        memories: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """
        Create multiple memories in one request.

        Args:
            memories: List of memory dicts (max 1000)

        Returns:
            Dict with created memories and count

        Example:
            >>> memories = [
            ...     {"content": "Fact 1", "user_id": "user-1"},
            ...     {"content": "Fact 2", "user_id": "user-1"},
            ... ]
            >>> result = client.batch_create_memories(memories)
        """
        if len(memories) > 1000:
            raise ValidationError("Maximum 1000 memories per batch")

        resp = self._request("POST", "/memories/batch", json={"memories": memories})
        return resp.json()

    def batch_update_memories(
        self,
        memory_ids: List[str],
        action: str = "update",
        content: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Batch update, archive, or delete memories.

        Args:
            memory_ids: List of memory IDs (max 1000)
            action: One of 'update', 'archive', 'delete'
            content: New content for update action
            metadata: Metadata to set for update action

        Returns:
            Status dict
        """
        if len(memory_ids) > 1000:
            raise ValidationError("Maximum 1000 IDs per batch")

        if action not in ("update", "archive", "delete"):
            raise ValidationError("action must be update, archive, or delete")

        payload = {
            "ids": memory_ids,
            "action": action,
        }
        if content:
            payload["content"] = content
        if metadata:
            payload["metadata"] = metadata

        resp = self._request("PUT", "/memories/batch-update", json=payload)
        return resp.json()

    def bulk_delete(
        self,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        category: Optional[str] = None,
    ) -> Dict[str, int]:
        """
        Delete all memories matching a filter.

        Args:
            user_id: Delete all memories for this user
            org_id: Delete all memories for this org
            category: Delete all memories in this category

        Returns:
            Dict with count of deleted memories
        """
        if not user_id and not org_id and not category:
            raise ValidationError(
                "At least one filter (user_id, org_id, category) is required"
            )

        payload = {}
        if user_id:
            payload["user_id"] = user_id
        if org_id:
            payload["org_id"] = org_id
        if category:
            payload["category"] = category

        resp = self._request("DELETE", "/memories/bulk-delete", json=payload)
        return resp.json()

    # ==================== Search ====================

    def search(
        self,
        query: str,
        limit: int = 10,
        threshold: float = 0.5,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        category: Optional[str] = None,
        memory_type: Optional[str] = None,
        rerank: bool = False,
        rerank_top_k: int = 20,
    ) -> List[Dict[str, Any]]:
        """
        Semantic search over stored memories.

        Uses vector embeddings to find semantically similar content.

        Args:
            query: Natural language search query (max 1000 chars)
            limit: Maximum results (1-100, default 10)
            threshold: Minimum similarity score (0.0-1.0, default 0.5)
            user_id: Filter by user
            org_id: Filter by organization
            agent_id: Filter by agent
            category: Filter by category
            memory_type: Filter by memory type
            rerank: Enable reranking for better results
            rerank_top_k: Number of results to rerank

        Returns:
            List of search results with score, content, metadata

        Example:
            >>> results = client.search(
            ...     "machine learning",
            ...     limit=5,
            ...     user_id="user-123",
            ...     rerank=True
            ... )
            >>> for r in results:
            ...     print(f"{r['score']:.2f}: {r['content']}")
        """
        params = {
            "q": query,
            "limit": min(max(limit, 1), 100),
            "threshold": min(max(threshold, 0.0), 1.0),
            "rerank": rerank,
            "rerank_top_k": rerank_top_k,
        }
        if user_id:
            params["user_id"] = user_id
        if org_id:
            params["org_id"] = org_id
        if agent_id:
            params["agent_id"] = agent_id
        if category:
            params["category"] = category
        if memory_type:
            params["memory_type"] = memory_type

        resp = self._request("GET", "/search", params=params)
        return resp.json()

    def search_post(
        self,
        query: str,
        filters: Optional[Dict[str, Any]] = None,
        limit: int = 10,
        threshold: float = 0.5,
    ) -> List[Dict[str, Any]]:
        """
        Advanced search with POST request and filter body.

        Supports more complex filter structures.
        """
        payload = {
            "query": query,
            "limit": limit,
            "threshold": threshold,
        }
        if filters:
            payload["filters"] = filters

        resp = self._request("POST", "/search", json=payload)
        return resp.json()

    def advanced_search(
        self,
        filters: Dict[str, Any],
        limit: int = 100,
    ) -> List[Dict[str, Any]]:
        """
        Advanced search with complex filter logic.

        Args:
            filters: Filter dict with rules and logic (AND/OR/NOT)
            limit: Maximum results

        Example:
            >>> filters = {
            ...     "logic": "AND",
            ...     "rules": [
            ...         {"field": "category", "operator": "eq", "value": "preferences"},
            ...         {"field": "user_id", "operator": "eq", "value": "user-123"}
            ...     ]
            ... }
            >>> results = client.advanced_search(filters)
        """
        payload = {
            "filters": filters,
            "limit": limit,
        }
        resp = self._request("POST", "/search/advanced", json=payload)
        return resp.json()

    # Alias for semantic_search
    semantic_search = search

    # ==================== Feedback ====================

    def add_feedback(
        self,
        memory_id: str,
        feedback_type: str,
        comment: Optional[str] = None,
        user_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        Provide feedback on a memory to improve future searches.

        Args:
            memory_id: Memory identifier
            feedback_type: One of positive, negative, very_negative
            comment: Optional feedback comment
            user_id: Optional user identifier

        Returns:
            Feedback dict with id

        Example:
            >>> client.add_feedback(
            ...     memory_id="mem-123",
            ...     feedback_type="positive",
            ...     comment="This is accurate"
            ... )
        """
        if feedback_type not in (
            FeedbackType.POSITIVE,
            FeedbackType.NEGATIVE,
            FeedbackType.VERY_NEGATIVE,
        ):
            raise ValidationError(
                f"Invalid feedback_type. Must be one of: "
                f"{FeedbackType.POSITIVE}, {FeedbackType.NEGATIVE}, {FeedbackType.VERY_NEGATIVE}"
            )

        payload = {
            "memory_id": memory_id,
            "type": feedback_type,
        }
        if comment:
            payload["comment"] = comment
        if user_id:
            payload["user_id"] = user_id

        resp = self._request("POST", "/feedback", json=payload)
        return resp.json()

    def get_memories_by_feedback(
        self,
        feedback_type: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """
        Get memories filtered by feedback type.

        Useful for finding high-quality (positive) or low-quality (negative) memories.
        """
        if feedback_type not in (
            FeedbackType.POSITIVE,
            FeedbackType.NEGATIVE,
            FeedbackType.VERY_NEGATIVE,
        ):
            raise ValidationError(f"Invalid feedback_type")

        resp = self._request(
            "GET",
            "/feedback/memories",
            params={"type": feedback_type, "limit": limit},
        )
        return resp.json()

    # ==================== History ====================

    def get_memory_history(self, memory_id: str) -> List[Dict[str, Any]]:
        """
        Get modification history for a memory.

        Returns:
            List of history entries (create, update, delete, feedback)
        """
        resp = self._request("GET", f"/memories/{memory_id}/history")
        return resp.json()

    # ==================== Memory Expiration ====================

    def set_memory_expiration(
        self,
        memory_id: str,
        expiration_date: datetime,
    ) -> Dict[str, str]:
        """
        Set an expiration date for a memory (TTL).

        Args:
            memory_id: Memory identifier
            expiration_date: When the memory should expire

        Returns:
            Status dict
        """
        payload = {
            "expiration_date": expiration_date.isoformat(),
        }
        resp = self._request("POST", f"/memories/{memory_id}/expire", json=payload)
        return resp.json()

    # ==================== Entity/Memory Linking ====================

    def link_memory_to_entity(
        self,
        memory_id: str,
        entity_id: str,
    ) -> Dict[str, str]:
        """Link a memory to an entity in the knowledge graph."""
        resp = self._request(
            "POST",
            f"/memories/{memory_id}/link/{entity_id}",
        )
        return resp.json()

    def get_entity_memories(
        self,
        entity_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """Get all memories linked to an entity."""
        resp = self._request(
            "GET",
            f"/entities/{entity_id}/memories",
            params={"limit": limit},
        )
        return resp.json()

    # ==================== Entities ====================

    def create_entity(
        self,
        name: str,
        entity_type: str,
        properties: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Create a knowledge graph entity."""
        payload = {"name": name, "type": entity_type}
        if properties:
            payload["properties"] = properties

        resp = self._request("POST", "/entities", json=payload)
        return resp.json()

    def get_entity(self, entity_id: str) -> Dict[str, Any]:
        """Get an entity by ID."""
        resp = self._request("GET", f"/entities/{entity_id}")
        return resp.json()

    def list_entities(
        self,
        entity_type: Optional[str] = None,
        limit: int = 100,
    ) -> Dict[str, Any]:
        """List entities with optional type filter."""
        params = {"limit": limit}
        if entity_type:
            params["entity_type"] = entity_type

        resp = self._request("GET", "/entities", params=params)
        return resp.json()

    def get_entity_relations(
        self,
        entity_id: str,
        relation_type: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """Get all relations for an entity."""
        params = {}
        if relation_type:
            params["type"] = relation_type

        resp = self._request("GET", f"/entities/{entity_id}/relations", params=params)
        return resp.json()

    # ==================== Relations ====================

    def create_relation(
        self,
        from_id: str,
        to_id: str,
        relation_type: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, str]:
        """Create a typed relationship between two entities."""
        allowed_types = (
            "KNOWS",
            "HAS",
            "RELATED_TO",
            "DEPENDS_ON",
            "USES",
            "CREATED_BY",
            "PART_OF",
            "IMPROVES",
            "CONFLICTS",
            "FOLLOWS",
            "LIKES",
            "DISLIKES",
            "SUBSCRIBED",
        )
        if relation_type not in allowed_types:
            raise ValidationError(
                f"Invalid relation type. Must be one of: {', '.join(allowed_types)}"
            )

        payload = {
            "from_id": from_id,
            "to_id": to_id,
            "type": relation_type,
        }
        if metadata:
            payload["metadata"] = metadata

        resp = self._request("POST", "/relations", json=payload)
        return resp.json()

    # ==================== Graph ====================

    def graph_query(
        self,
        cypher: str,
        params: Optional[Dict[str, Any]] = None,
    ) -> List[Dict[str, Any]]:
        """Execute a raw Cypher query on the knowledge graph."""
        payload = {"cypher": cypher}
        if params:
            payload["params"] = params

        resp = self._request("POST", "/graph/query", json=payload)
        return resp.json()

    def traverse(
        self,
        entity_id: str,
        depth: int = 3,
    ) -> List[Dict[str, Any]]:
        """Traverse graph from an entity up to a given depth."""
        resp = self._request(
            "GET",
            f"/graph/traverse/{entity_id}",
            params={"depth": depth},
        )
        return resp.json()

    # ==================== Admin ====================

    def cleanup_expired_memories(self) -> Dict[str, int]:
        """Delete all expired memories (admin only)."""
        resp = self._request("POST", "/admin/cleanup")
        return resp.json()

    def sync_entities(
        self,
        entity_ids: Optional[List[str]] = None,
    ) -> Dict[str, str]:
        """Sync entities to vector store (admin only)."""
        payload = {}
        if entity_ids:
            payload["entity_ids"] = entity_ids

        resp = self._request("POST", "/admin/sync", json=payload)
        return resp.json()

    def list_api_keys(self) -> List[Dict[str, Any]]:
        """List all API keys (admin only)."""
        resp = self._request("GET", "/admin/api-keys")
        return resp.json()

    def create_api_key(
        self,
        label: str,
        expires_in_hours: int = 0,
        tenant_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Create a new API key."""
        payload = {
            "label": label,
            "expires_in_hours": expires_in_hours,
        }
        if tenant_id:
            payload["tenant_id"] = tenant_id

        resp = self._request("POST", "/admin/api-keys", json=payload)
        return resp.json()

    def delete_api_key(self, key_id: str) -> Dict[str, str]:
        """Delete an API key."""
        resp = self._request("DELETE", f"/admin/api-keys/{key_id}")
        return resp.json()


# ==================== Convenience Functions ====================


def create_session(agent_id: str, **kwargs) -> Dict[str, Any]:
    """Create a new session using default client."""
    return AgentMemory().create_session(agent_id, **kwargs)


def add_message(session_id: str, role: str, content: str) -> Dict[str, str]:
    """Add a message using default client."""
    return AgentMemory().add_message(session_id, role, content)


def search(query: str, **kwargs) -> List[Dict[str, Any]]:
    """Semantic search using default client."""
    return AgentMemory().search(query, **kwargs)


def create_memory(content: str, **kwargs) -> Dict[str, Any]:
    """Create a memory using default client."""
    return AgentMemory().create_memory(content, **kwargs)


# Export public API
__all__ = [
    "AgentMemory",
    "AgentMemoryError",
    "AuthenticationError",
    "NotFoundError",
    "ValidationError",
    "RateLimitError",
    "MemoryType",
    "FeedbackType",
    "create_session",
    "add_message",
    "search",
    "create_memory",
]
