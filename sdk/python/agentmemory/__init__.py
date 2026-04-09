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

    # Later, search semantically
    results = client.search("deep learning")
"""

import os
from typing import Optional, List, Dict, Any, Iterator
from datetime import datetime
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

        Example:
            >>> client = AgentMemory(
            ...     base_url="https://api.agentmemory.io",
            ...     api_key="am_xxxxx",
            ...     timeout=60
            ... )
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
        """
        Check API health status.

        Returns:
            Dict with "status" key

        Example:
            >>> client.health()
            {'status': 'ok'}
        """
        resp = self._request("GET", "/health")
        return resp.json()

    def ready(self) -> Dict[str, Any]:
        """
        Check API readiness including dependency health.

        Returns:
            Dict with neo4j and qdrant status

        Example:
            >>> client.ready()
            {'neo4j': 'healthy', 'qdrant': 'healthy'}
        """
        resp = self._request("GET", "/ready")
        return resp.json()

    # ==================== Sessions ====================

    def create_session(
        self,
        agent_id: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Create a new conversation session for an agent.

        Args:
            agent_id: Unique identifier for the agent (1-64 chars, alphanumeric, -_).
                      Examples: "support-bot", "code-assistant-1", "researcher"
            metadata: Optional metadata dict (e.g., {"customer_id": "123"})

        Returns:
            Session dict with keys: id, agent_id, created_at, updated_at

        Example:
            >>> session = client.create_session(
            ...     agent_id="support-bot",
            ...     metadata={"customer_id": "CUST-001"}
            ... )
            >>> session['id']
            '550e8400-e29b-41d4-a716-446655440000'
        """
        payload = {"agent_id": agent_id}
        if metadata:
            payload["metadata"] = metadata

        resp = self._request("POST", "/sessions", json=payload)
        return resp.json()

    def get_messages(
        self,
        session_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """
        Get conversation messages for a session.

        Args:
            session_id: Session ID (UUID)
            limit: Maximum number of messages to return (default 50)

        Returns:
            List of message dicts with: id, session_id, role, content, timestamp

        Example:
            >>> messages = client.get_messages("session-id", limit=10)
            >>> for msg in messages:
            ...     print(f"{msg['role']}: {msg['content']}")
        """
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
        """
        Add a message to a session conversation.

        Args:
            session_id: Session ID (UUID)
            role: Message role. Must be one of: "user", "assistant", "system", "tool"
            content: Message content (max 100KB)

        Returns:
            Dict with "status": "ok"

        Example:
            >>> client.add_message(
            ...     session_id="550e8400-e29b-41d4-a716-446655440000",
            ...     role="user",
            ...     content="I need help with my account"
            ... )
            {'status': 'ok'}
        """
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
        """
        Get conversation context for a session.

        This is an alias for get_messages() that returns the full context
        needed for an LLM to continue the conversation.

        Args:
            session_id: Session ID
            limit: Maximum messages

        Returns:
            List of messages in chronological order
        """
        return self.get_messages(session_id, limit)

    # ==================== Entities ====================

    def create_entity(
        self,
        name: str,
        entity_type: str = "",
        properties: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, str]:
        """
        Create a knowledge graph entity.

        Entities represent things in your domain - people, concepts,
        documents, services, etc.

        Args:
            name: Entity name (1-64 chars, alphanumeric, -_)
            entity_type: Entity type (e.g., "Person", "Document", "Service")
            properties: Optional dict of custom properties

        Returns:
            Dict with "status": "ok" and "id" (entity UUID)

        Example:
            >>> entity = client.create_entity(
            ...     name="auth-service",
            ...     entity_type="Service",
            ...     properties={"port": 8080, "language": "python"}
            ... )
            >>> entity['id']
            '550e8400-e29b-41d4-a716-446655440000'
        """
        if not entity_type:
            raise ValidationError("entity_type is required")

        payload = {"name": name, "type": entity_type}
        if properties:
            payload["properties"] = properties

        resp = self._request("POST", "/entities", json=payload)
        return resp.json()

    def get_entity(self, entity_id: str) -> Dict[str, Any]:
        """
        Get an entity by ID.

        Args:
            entity_id: Entity UUID

        Returns:
            Entity dict with id, type, name, properties, timestamps

        Example:
            >>> entity = client.get_entity("entity-id")
            >>> print(entity['name'], entity['type'])
        """
        resp = self._request("GET", f"/entities/{entity_id}")
        return resp.json()

    def get_entity_relations(
        self,
        entity_id: str,
    ) -> List[Dict[str, Any]]:
        """
        Get all relations for an entity.

        Args:
            entity_id: Entity UUID

        Returns:
            List of relation dicts

        Example:
            >>> relations = client.get_entity_relations("entity-id")
            >>> for rel in relations:
            ...     print(f"{rel['type']} -> {rel['to_id']}")
        """
        resp = self._request("GET", f"/entities/{entity_id}/relations")
        return resp.json()

    # ==================== Relations ====================

    def create_relation(
        self,
        from_id: str,
        to_id: str,
        rel_type: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, str]:
        """
        Create a typed relationship between two entities.

        Relation types are limited to prevent injection. Allowed types:
        KNOWS, HAS, RELATED_TO, DEPENDS_ON, USES, CREATED_BY, PART_OF,
        IMPROVES, CONFLICTS, FOLLOWS, LIKES, DISLIKES, SUBSCRIBED

        Args:
            from_id: Source entity UUID
            to_id: Target entity UUID
            rel_type: One of the allowed relation types (uppercase)
            metadata: Optional relation properties

        Returns:
            Dict with "status": "ok"

        Example:
            >>> client.create_relation(
            ...     from_id="entity-a",
            ...     to_id="entity-b",
            ...     rel_type="KNOWS"
            ... )
        """
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
        if rel_type not in allowed_types:
            raise ValidationError(
                f"Invalid relation type. Must be one of: {', '.join(allowed_types)}"
            )

        payload = {
            "from_id": from_id,
            "to_id": to_id,
            "type": rel_type,
        }
        if metadata:
            payload["metadata"] = metadata

        resp = self._request("POST", "/relations", json=payload)
        return resp.json()

    # ==================== Search ====================

    def search(
        self,
        query: str,
        limit: int = 10,
        threshold: float = 0.5,
    ) -> List[Dict[str, Any]]:
        """
        Semantic search over stored memories.

        Uses vector embeddings to find semantically similar content.
        Requires OpenAI API key configured on the server.

        Args:
            query: Natural language search query (max 1000 chars)
            limit: Maximum results to return (1-100, default 10)
            threshold: Minimum similarity score (0.0-1.0, default 0.5)

        Returns:
            List of search results, each containing:
            - entity: The matching entity
            - score: Similarity score (0-1)
            - source: Content source

        Example:
            >>> results = client.search("machine learning transformers", limit=5)
            >>> for r in results:
            ...     print(f"Score: {r['score']:.2f} - {r['entity']['name']}")
            Score: 0.92 - Attention Mechanism
            Score: 0.87 - Transformer Architecture
        """
        params = {
            "q": query,
            "limit": min(max(limit, 1), 100),
            "threshold": min(max(threshold, 0.0), 1.0),
        }
        resp = self._request("GET", "/search", params=params)
        return resp.json()

    # Alias for semantic_search
    semantic_search = search

    # ==================== Graph ====================

    def graph_query(
        self,
        cypher: str,
        params: Optional[Dict[str, Any]] = None,
    ) -> List[Dict[str, Any]]:
        """
        Execute a raw Cypher query on the knowledge graph.

        WARNING: Requires admin API key. Use with caution.

        Args:
            cypher: Neo4j Cypher query
            params: Optional query parameters

        Returns:
            List of result records

        Example:
            >>> results = client.graph_query(
            ...     "MATCH (e:Entity) RETURN e LIMIT 5"
            ... )
        """
        payload = {"cypher": cypher}
        if params:
            payload["params"] = params

        resp = self._request("POST", "/graph/query", json=payload)
        return resp.json()

    # ==================== Admin ====================

    def list_api_keys(self) -> List[Dict[str, Any]]:
        """
        List all API keys (admin only).

        Returns:
            List of API key dicts (key value hidden)
        """
        resp = self._request("GET", "/admin/api-keys")
        return resp.json()

    def create_api_key(
        self,
        label: str,
        expires_in_hours: int = 0,
    ) -> Dict[str, str]:
        """
        Create a new API key.

        Args:
            label: Human-readable label for the key
            expires_in_hours: Hours until expiration (0 = never)

        Returns:
            Dict with key, id, label. SAVE THE KEY - it won't be shown again!

        Example:
            >>> new_key = client.create_api_key("production-bot")
            >>> print(new_key['key'])
            am_xxxxxxxxxxxxx  # Save this!
        """
        payload = {
            "label": label,
            "expires_in_hours": expires_in_hours,
        }
        resp = self._request("POST", "/admin/api-keys", json=payload)
        return resp.json()

    def delete_api_key(self, key_id: str) -> Dict[str, str]:
        """
        Delete an API key.

        Args:
            key_id: Key ID to delete

        Returns:
            Dict with "status": "deleted"
        """
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


# Export public API
__all__ = [
    "AgentMemory",
    "AgentMemoryError",
    "AuthenticationError",
    "NotFoundError",
    "ValidationError",
    "RateLimitError",
    "create_session",
    "add_message",
    "search",
]
