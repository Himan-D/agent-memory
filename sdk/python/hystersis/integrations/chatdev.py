"""
ChatDev Integration for Hystersis

This module provides a shared memory store for ChatDev agent teams,
enabling agents to persist and retrieve knowledge across development phases.

Usage:
    from hystersis.integrations.chatdev import HystsersisChatDevMemory

    memory = HystsersisChatDevMemory(
        api_key="your-api-key",
        project_id="my-project",
        base_url="https://api.hystersis.ai"
    )

    # Save phase output
    memory.save("design", "The architecture uses a microservices pattern")

    # Retrieve context for a phase
    results = memory.retrieve("coding", "microservices implementation")
"""

from typing import Any, Dict, List, Optional

import requests


class HystsersisChatDevMemory:
    """
    Shared memory store for ChatDev agent teams.

    Enables agents across ChatDev phases (design, coding, testing, etc.)
    to share knowledge and build on each other's outputs.

    Attributes:
        project_id: Unique identifier for the ChatDev project
        base_url: Base URL of the Hystersis API
        api_key: API key for authentication
    """

    def __init__(
        self,
        api_key: str,
        project_id: str,
        base_url: str = "https://api.hystersis.ai",
    ):
        self.project_id = project_id
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key

        self._session = requests.Session()
        self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make an HTTP request to the Hystersis API."""
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    def save(self, phase: str, content: str) -> Optional[str]:
        """
        Save content from a ChatDev phase to memory.

        Args:
            phase: The development phase (e.g., "design", "coding", "testing")
            content: The content to store

        Returns:
            Memory ID if successful, None otherwise
        """
        try:
            resp = self._request(
                "POST",
                "/memories",
                json={
                    "content": content,
                    "user_id": self.project_id,
                    "type": "user",
                    "metadata": {
                        "source": "chatdev",
                        "project_id": self.project_id,
                        "phase": phase,
                    },
                },
            )
            return resp.json().get("id")
        except Exception as e:
            print(f"Warning: Failed to save phase content: {e}")
            return None

    def retrieve(self, phase: str, query: str, limit: int = 5) -> List[str]:
        """
        Retrieve relevant context for a ChatDev phase.

        Searches memory for content relevant to the given query,
        useful for providing context to agents in subsequent phases.

        Args:
            phase: Current phase requesting context
            query: Search query describing what context is needed
            limit: Maximum number of results

        Returns:
            List of relevant memory content strings
        """
        try:
            resp = self._request(
                "GET",
                "/search",
                params={
                    "q": query,
                    "user_id": self.project_id,
                    "limit": limit,
                },
            )
            results = resp.json()
            items = results if isinstance(results, list) else results.get("results", [])
            return [
                r.get("text", "") or r.get("content", "")
                for r in items
                if r.get("text") or r.get("content")
            ]
        except Exception:
            return []

    def get_phase_history(self, phase: str) -> List[Dict[str, Any]]:
        """
        Get all memories stored during a specific phase.

        Args:
            phase: The phase to retrieve history for

        Returns:
            List of memory dictionaries
        """
        try:
            resp = self._request(
                "GET",
                "/memories",
                params={
                    "user_id": self.project_id,
                    "limit": 50,
                },
            )
            results = resp.json()
            items = results if isinstance(results, list) else results.get("memories", [])
            return [
                item for item in items
                if item.get("metadata", {}).get("phase") == phase
            ]
        except Exception:
            return []

    def __repr__(self) -> str:
        return f"HystsersisChatDevMemory(project_id='{self.project_id}')"


__all__ = ["HystsersisChatDevMemory"]
