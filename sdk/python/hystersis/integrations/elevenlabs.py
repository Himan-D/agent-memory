"""
ElevenLabs Conversational AI Integration for Hystersis

This module provides memory context for ElevenLabs conversational AI agents,
enabling persistent memory across voice conversations.

Usage:
    from hystersis.integrations.elevenlabs import HystersisElevenLabsMemory

    memory = HystersisElevenLabsMemory(
        api_key="your-api-key",
        base_url="https://api.hystersis.ai"
    )

    # Get system prompt context for a user
    context = memory.get_system_prompt_context(user_id="user-123")

    # Store a conversation
    memory.store_conversation(
        messages=[
            {"role": "user", "content": "Tell me about my last order"},
            {"role": "assistant", "content": "Your last order was..."}
        ],
        user_id="user-123"
    )
"""

from typing import Any, Dict, List, Optional

import requests


class HystersisElevenLabsMemory:
    """
    Memory context provider for ElevenLabs conversational AI.

    Retrieves relevant memories to include in system prompts and stores
    conversation history for future context retrieval.

    Attributes:
        base_url: Base URL of the Hystersis API
        api_key: API key for authentication
        context_limit: Max number of memories to include in system prompt
    """

    def __init__(
        self,
        api_key: str,
        base_url: str = "https://api.hystersis.ai",
        context_limit: int = 5,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.context_limit = context_limit

        self._session = requests.Session()
        self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make an HTTP request to the Hystersis API."""
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    def get_system_prompt_context(
        self, user_id: str, query: Optional[str] = None
    ) -> str:
        """
        Get memory context to prepend to the ElevenLabs system prompt.

        Retrieves the most relevant memories for the user and formats them
        as a context block suitable for inclusion in a system prompt.

        Args:
            user_id: User identifier to retrieve memories for
            query: Optional search query; if None, retrieves recent memories

        Returns:
            Formatted context string for the system prompt
        """
        try:
            if query:
                resp = self._request(
                    "GET",
                    "/search",
                    params={
                        "q": query,
                        "user_id": user_id,
                        "limit": self.context_limit,
                    },
                )
            else:
                resp = self._request(
                    "GET",
                    "/memories",
                    params={
                        "user_id": user_id,
                        "limit": self.context_limit,
                    },
                )

            results = resp.json()
            items = results if isinstance(results, list) else results.get("results", results.get("memories", []))
        except Exception:
            return ""

        if not items:
            return ""

        memory_texts = []
        for item in items:
            text = item.get("text", "") or item.get("content", "")
            if text:
                memory_texts.append(f"- {text}")

        if not memory_texts:
            return ""

        return (
            "You have the following context about this user from previous conversations:\n"
            + "\n".join(memory_texts)
        )

    def store_conversation(
        self, messages: List[Dict[str, str]], user_id: str
    ) -> None:
        """
        Store conversation messages in Hystersis memory.

        Args:
            messages: List of message dicts with 'role' and 'content'
            user_id: User identifier
        """
        for msg in messages:
            content = msg.get("content", "")
            role = msg.get("role", "user")
            if not content:
                continue

            try:
                self._request(
                    "POST",
                    "/memories",
                    json={
                        "content": content,
                        "user_id": user_id,
                        "type": "conversation",
                        "metadata": {
                            "source": "elevenlabs",
                            "role": role,
                        },
                    },
                )
            except Exception as e:
                print(f"Warning: Failed to store message: {e}")

    def __repr__(self) -> str:
        return "HystersisElevenLabsMemory()"


__all__ = ["HystersisElevenLabsMemory"]
