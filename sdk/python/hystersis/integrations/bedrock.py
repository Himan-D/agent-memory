"""
AWS Bedrock Integration for Hystersis

This module provides memory capabilities for AWS Bedrock agents,
enabling persistent context across agent sessions.

Usage:
    from hystersis.integrations.bedrock import HystersisBedrockMemory

    memory = HystersisBedrockMemory(
        api_key="your-api-key",
        base_url="https://api.hystersis.ai"
    )

    # Get session context for a Bedrock agent
    context = memory.get_session_context(session_id="session-123")

    # Save a conversation turn
    memory.save_turn(
        session_id="session-123",
        input_text="What is quantum computing?",
        output_text="Quantum computing uses quantum mechanics..."
    )
"""

from typing import Any, Dict, List, Optional

import requests


class HystersisBedrockMemory:
    """
    Memory provider for AWS Bedrock agents.

    Stores and retrieves conversation context for Bedrock agent sessions,
    enabling persistent memory across invocations.

    Attributes:
        base_url: Base URL of the Hystersis API
        api_key: API key for authentication
        agent_id: Optional Bedrock agent identifier
    """

    def __init__(
        self,
        api_key: str,
        base_url: str = "https://api.hystersis.ai",
        agent_id: Optional[str] = None,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.agent_id = agent_id

        self._session = requests.Session()
        self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make an HTTP request to the Hystersis API."""
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    def get_session_context(
        self, session_id: str, query: Optional[str] = None, limit: int = 10
    ) -> List[Dict[str, Any]]:
        """
        Get conversation context for a Bedrock session.

        Retrieves recent memories associated with the session, optionally
        filtered by a search query for relevance.

        Args:
            session_id: Bedrock session identifier
            query: Optional search query to filter by relevance
            limit: Maximum number of context items

        Returns:
            List of memory dictionaries with content and metadata
        """
        try:
            if query:
                resp = self._request(
                    "GET",
                    "/search",
                    params={
                        "q": query,
                        "user_id": session_id,
                        "limit": limit,
                    },
                )
                results = resp.json()
                items = results if isinstance(results, list) else results.get("results", [])
            else:
                resp = self._request(
                    "GET",
                    "/memories",
                    params={
                        "user_id": session_id,
                        "limit": limit,
                    },
                )
                results = resp.json()
                items = results if isinstance(results, list) else results.get("memories", [])

            return items
        except Exception:
            return []

    def save_turn(
        self,
        session_id: str,
        input_text: str,
        output_text: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> None:
        """
        Save a conversation turn from a Bedrock agent session.

        Stores both the user input and agent output as separate memories
        linked to the session.

        Args:
            session_id: Bedrock session identifier
            input_text: User input text
            output_text: Agent output text
            metadata: Optional additional metadata
        """
        base_metadata: Dict[str, Any] = {
            "source": "bedrock",
            "session_id": session_id,
        }
        if self.agent_id:
            base_metadata["agent_id"] = self.agent_id
        if metadata:
            base_metadata.update(metadata)

        # Store user input
        if input_text:
            try:
                input_meta = {**base_metadata, "role": "user"}
                self._request(
                    "POST",
                    "/memories",
                    json={
                        "content": input_text,
                        "user_id": session_id,
                        "type": "conversation",
                        "metadata": input_meta,
                    },
                )
            except Exception as e:
                print(f"Warning: Failed to store input: {e}")

        # Store agent output
        if output_text:
            try:
                output_meta = {**base_metadata, "role": "assistant"}
                self._request(
                    "POST",
                    "/memories",
                    json={
                        "content": output_text,
                        "user_id": session_id,
                        "type": "conversation",
                        "metadata": output_meta,
                    },
                )
            except Exception as e:
                print(f"Warning: Failed to store output: {e}")

    def get_session_summary(self, session_id: str) -> str:
        """
        Get a formatted summary of the session context.

        Args:
            session_id: Bedrock session identifier

        Returns:
            Formatted string summary of session context
        """
        items = self.get_session_context(session_id)
        if not items:
            return "No previous context available."

        lines = []
        for item in items:
            role = item.get("metadata", {}).get("role", "unknown")
            content = item.get("text", "") or item.get("content", "")
            if content:
                lines.append(f"[{role}] {content}")

        return "\n".join(lines) if lines else "No previous context available."

    def __repr__(self) -> str:
        return f"HystersisBedrockMemory(agent_id='{self.agent_id}')"


__all__ = ["HystersisBedrockMemory"]
