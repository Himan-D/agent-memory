"""
Camel AI Integration for Hystersis

This module provides an external memory backend for the Camel AI multi-agent
framework, enabling persistent shared memory across agent teams.

Usage:
    from hystersis.integrations.camelai import HystsersisCamelMemory

    memory = HystsersisCamelMemory(
        api_key="your-api-key",
        agent_id="researcher",
        base_url="https://api.hystersis.ai"
    )

    # Write memory records from agent interactions
    memory.write_records([
        {"role": "user", "content": "Research quantum computing papers"},
        {"role": "assistant", "content": "Found 5 relevant papers on quantum error correction"}
    ])

    # Get a context creator for the agent
    context = memory.get_context_creator()
"""

from typing import Any, Callable, Dict, List, Optional

import requests


class HystsersisCamelMemory:
    """
    External memory backend for Camel AI multi-agent framework.

    Provides persistent memory storage that can be shared across agents
    in a Camel AI team, with support for writing conversation records
    and creating context-aware prompts.

    Attributes:
        agent_id: Identifier for the agent using this memory
        user_id: Optional user identifier
        base_url: Base URL of the Hystersis API
        api_key: API key for authentication
    """

    def __init__(
        self,
        api_key: str,
        agent_id: str,
        base_url: str = "https://api.hystersis.ai",
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
    ):
        self.agent_id = agent_id
        self.user_id = user_id
        self.org_id = org_id
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

    def write_records(self, records: List[Dict[str, str]]) -> List[str]:
        """
        Write conversation records to memory.

        Stores each record as a separate memory entry with metadata
        linking it to this agent.

        Args:
            records: List of message dicts with 'role' and 'content' keys

        Returns:
            List of created memory IDs
        """
        ids = []
        for record in records:
            content = record.get("content", "")
            if not content:
                continue

            metadata: Dict[str, Any] = {
                "source": "camelai",
                "agent_id": self.agent_id,
                "role": record.get("role", "user"),
            }

            payload: Dict[str, Any] = {
                "content": content,
                "type": "conversation",
                "metadata": metadata,
            }
            if self.user_id:
                payload["user_id"] = self.user_id
            if self.org_id:
                payload["org_id"] = self.org_id

            try:
                resp = self._request("POST", "/memories", json=payload)
                data = resp.json()
                ids.append(data.get("id", ""))
            except Exception as e:
                print(f"Warning: Failed to write record: {e}")

        return ids

    def get_context_creator(self) -> Callable[[str], str]:
        """
        Get a context creator function for this agent.

        Returns a callable that takes a query string and returns
        relevant context from memory, suitable for injection into
        agent prompts.

        Returns:
            A function (query: str) -> str that returns memory context
        """

        def create_context(query: str) -> str:
            memories = self.retrieve(query)
            if not memories:
                return ""
            lines = [f"- {m}" for m in memories]
            return "Relevant context from memory:\n" + "\n".join(lines)

        return create_context

    def retrieve(self, query: str, limit: int = 5) -> List[str]:
        """
        Retrieve relevant memories for a query.

        Args:
            query: Search query
            limit: Max results

        Returns:
            List of memory content strings
        """
        try:
            params: Dict[str, Any] = {
                "q": query,
                "limit": limit,
            }
            if self.user_id:
                params["user_id"] = self.user_id

            resp = self._request("GET", "/search", params=params)
            results = resp.json()
            items = results if isinstance(results, list) else results.get("results", [])
            return [
                r.get("text", "") or r.get("content", "")
                for r in items
                if r.get("text") or r.get("content")
            ]
        except Exception:
            return []

    def clear(self) -> None:
        """Clear all memories for this agent."""
        try:
            params: Dict[str, Any] = {}
            if self.user_id:
                params["user_id"] = self.user_id
            self._request("DELETE", "/memories", params=params)
        except Exception as e:
            print(f"Warning: Failed to clear memories: {e}")

    def __repr__(self) -> str:
        return f"HystsersisCamelMemory(agent_id='{self.agent_id}')"


__all__ = ["HystsersisCamelMemory"]
