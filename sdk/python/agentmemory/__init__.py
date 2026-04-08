"""
Agent Memory Python SDK

A Python client for the Agent Memory System API.
"""

import os
from typing import Optional, List, Dict, Any
from datetime import datetime
import requests


class AgentMemory:
    """Python SDK for Agent Memory System."""

    def __init__(
        self,
        api_key: Optional[str] = None,
        base_url: str = "http://localhost:8080",
        timeout: int = 30,
    ):
        """
        Initialize the Agent Memory client.

        Args:
            api_key: API key for authentication
            base_url: Base URL of the Agent Memory API
            timeout: Request timeout in seconds
        """
        self.api_key = api_key or os.environ.get("AGENT_MEMORY_API_KEY", "")
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self._session = requests.Session()

        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make an HTTP request."""
        url = f"{self.base_url}{endpoint}"
        return self._session.request(method, url, timeout=self.timeout, **kwargs)

    # ==================== Health ====================

    def health(self) -> Dict[str, str]:
        """Check API health."""
        resp = self._request("GET", "/health")
        resp.raise_for_status()
        return resp.json()

    def ready(self) -> Dict[str, str]:
        """Check API readiness."""
        resp = self._request("GET", "/ready")
        resp.raise_for_status()
        return resp.json()

    # ==================== Sessions ====================

    def create_session(
        self,
        agent_id: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Create a new session.

        Args:
            agent_id: Identifier for the agent
            metadata: Optional metadata for the session

        Returns:
            Session object with id, agent_id, metadata, timestamps
        """
        payload = {"agent_id": agent_id}
        if metadata:
            payload["metadata"] = metadata

        resp = self._request("POST", "/sessions", json=payload)
        resp.raise_for_status()
        return resp.json()

    def get_messages(
        self,
        session_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """
        Get messages for a session.

        Args:
            session_id: Session ID
            limit: Maximum number of messages

        Returns:
            List of messages
        """
        resp = self._request(
            "GET",
            f"/sessions/{session_id}/messages",
            params={"limit": limit},
        )
        resp.raise_for_status()
        return resp.json()

    def add_message(
        self,
        session_id: str,
        role: str,
        content: str,
    ) -> Dict[str, str]:
        """
        Add a message to a session.

        Args:
            session_id: Session ID
            role: Message role (user/assistant)
            content: Message content

        Returns:
            Status response
        """
        payload = {"role": role, "content": content}
        resp = self._request(
            "POST",
            f"/sessions/{session_id}/messages",
            json=payload,
        )
        resp.raise_for_status()
        return resp.json()

    def get_context(
        self,
        session_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """
        Get conversation context for a session.
        Alias for get_messages.
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
        Create a new entity in the knowledge graph.

        Args:
            name: Entity name
            entity_type: Entity type
            properties: Additional properties

        Returns:
            Response with entity ID
        """
        payload = {"name": name}
        if entity_type:
            payload["type"] = entity_type
        if properties:
            payload["properties"] = properties

        resp = self._request("POST", "/entities", json=payload)
        resp.raise_for_status()
        return resp.json()

    def get_entity(self, entity_id: str) -> Dict[str, Any]:
        """
        Get an entity by ID.

        Args:
            entity_id: Entity ID

        Returns:
            Entity object
        """
        resp = self._request("GET", f"/entities/{entity_id}")
        resp.raise_for_status()
        return resp.json()

    def get_relations(
        self,
        entity_id: str,
        rel_type: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """
        Get relations for an entity.

        Args:
            entity_id: Entity ID
            rel_type: Optional relation type filter

        Returns:
            List of relations
        """
        params = {}
        if rel_type:
            params["type"] = rel_type

        resp = self._request(
            "GET",
            f"/entities/{entity_id}/relations",
            params=params,
        )
        resp.raise_for_status()
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
        Create a relation between two entities.

        Args:
            from_id: Source entity ID
            to_id: Target entity ID
            rel_type: Relation type
            metadata: Optional metadata

        Returns:
            Status response
        """
        payload = {
            "from_id": from_id,
            "to_id": to_id,
            "type": rel_type,
        }
        if metadata:
            payload["metadata"] = metadata

        resp = self._request("POST", "/relations", json=payload)
        resp.raise_for_status()
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

        Args:
            query: Search query
            limit: Maximum results
            threshold: Minimum similarity score

        Returns:
            List of search results with scores
        """
        params = {
            "q": query,
            "limit": limit,
            "threshold": threshold,
        }
        resp = self._request("GET", "/search", params=params)
        resp.raise_for_status()
        return resp.json()

    # ==================== Graph ====================

    def graph_query(
        self,
        cypher: str,
        params: Optional[Dict[str, Any]] = None,
    ) -> List[Dict[str, Any]]:
        """
        Execute a Cypher query on the knowledge graph.

        Args:
            cypher: Cypher query
            params: Query parameters

        Returns:
            Query results
        """
        payload = {"cypher": cypher}
        if params:
            payload["params"] = params

        resp = self._request("POST", "/graph/query", json=payload)
        resp.raise_for_status()
        return resp.json()

    # ==================== API Keys (Admin) ====================

    def list_api_keys(self) -> List[Dict[str, Any]]:
        """List all API keys (admin only)."""
        resp = self._request("GET", "/admin/api-keys")
        resp.raise_for_status()
        return resp.json()

    def create_api_key(
        self,
        label: str,
        expires_in_hours: int = 0,
    ) -> Dict[str, str]:
        """
        Create a new API key.

        Args:
            label: Label for the key
            expires_in_hours: Expiration time in hours (0 = never)

        Returns:
            Response with key ID and the actual key
        """
        payload = {
            "label": label,
            "expires_in_hours": expires_in_hours,
        }
        resp = self._request("POST", "/admin/api-keys", json=payload)
        resp.raise_for_status()
        return resp.json()

    def delete_api_key(self, key_id: str) -> Dict[str, str]:
        """
        Delete an API key.

        Args:
            key_id: Key ID to delete

        Returns:
            Status response
        """
        resp = self._request("DELETE", f"/admin/api-keys/{key_id}")
        resp.raise_for_status()
        return resp.json()


# Convenience functions


def create_session(agent_id: str, **kwargs) -> Dict[str, Any]:
    """Create a new session."""
    return AgentMemory().create_session(agent_id, **kwargs)


def add_message(session_id: str, role: str, content: str) -> Dict[str, str]:
    """Add a message to a session."""
    return AgentMemory().add_message(session_id, role, content)


def search(query: str, **kwargs) -> List[Dict[str, Any]]:
    """Semantic search."""
    return AgentMemory().search(query, **kwargs)
