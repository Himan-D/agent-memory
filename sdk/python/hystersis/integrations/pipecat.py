"""
Pipecat Integration for Hystersis

This module provides a memory processor for Pipecat voice AI pipelines
that stores transcription frames and augments LLM prompts with relevant context.

Usage:
    from hystersis.integrations.pipecat import HystersisMemoryProcessor

    processor = HystersisMemoryProcessor(
        api_key="your-api-key",
        user_id="user-123",
        base_url="https://api.hystersis.ai"
    )

    # Process a transcription frame
    await processor.process_frame({"text": "Hello, I need help", "role": "user"})

    # Augment LLM messages with memory context
    messages = await processor.augment_prompt([
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "What did we discuss last time?"}
    ])
"""

from typing import Any, Dict, List, Optional

import requests


class HystersisMemoryProcessor:
    """
    Memory processor for Pipecat voice AI pipelines.

    Processes transcription frames, stores them in Hystersis memory,
    and augments LLM prompts with relevant context from past conversations.

    Attributes:
        user_id: Unique identifier for the user
        base_url: Base URL of the Hystersis API
        api_key: API key for authentication
        context_limit: Max number of memories to inject into prompts
    """

    def __init__(
        self,
        api_key: str,
        user_id: str,
        base_url: str = "https://api.hystersis.ai",
        session_id: Optional[str] = None,
        context_limit: int = 5,
    ):
        self.user_id = user_id
        self.session_id = session_id
        self.context_limit = context_limit
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

    async def process_frame(self, frame: Dict[str, Any]) -> None:
        """
        Process a Pipecat transcription frame and store in memory.

        Args:
            frame: Transcription frame with 'text' and optional 'role' keys
        """
        text = frame.get("text", "")
        if not text or not text.strip():
            return

        role = frame.get("role", "user")
        metadata: Dict[str, Any] = {
            "source": "pipecat",
            "role": role,
        }
        if self.session_id:
            metadata["session_id"] = self.session_id
        if "language" in frame:
            metadata["language"] = frame["language"]
        if "confidence" in frame:
            metadata["confidence"] = frame["confidence"]

        try:
            self._request(
                "POST",
                "/memories",
                json={
                    "content": text,
                    "user_id": self.user_id,
                    "type": "conversation",
                    "metadata": metadata,
                },
            )
        except Exception as e:
            print(f"Warning: Failed to store frame: {e}")

    async def augment_prompt(
        self, messages: List[Dict[str, str]]
    ) -> List[Dict[str, str]]:
        """
        Augment LLM prompt messages with relevant memory context.

        Searches for relevant memories based on the last user message and
        injects them as a system message before the conversation.

        Args:
            messages: List of message dicts with 'role' and 'content' keys

        Returns:
            Augmented message list with memory context injected
        """
        # Extract the last user message as the search query
        query = ""
        for msg in reversed(messages):
            if msg.get("role") == "user":
                query = msg.get("content", "")
                break

        if not query:
            return messages

        # Search for relevant memories
        try:
            resp = self._request(
                "GET",
                "/search",
                params={
                    "q": query,
                    "user_id": self.user_id,
                    "limit": self.context_limit,
                },
            )
            results = resp.json()
            items = results if isinstance(results, list) else results.get("results", [])
        except Exception:
            return messages

        if not items:
            return messages

        # Build context string
        memory_texts = []
        for r in items:
            text = r.get("text", "") or r.get("content", "")
            if text:
                memory_texts.append(f"- {text}")

        if not memory_texts:
            return messages

        context_msg = {
            "role": "system",
            "content": (
                "Relevant context from previous conversations:\n"
                + "\n".join(memory_texts)
            ),
        }

        # Inject after the first system message, or at the beginning
        augmented = list(messages)
        insert_idx = 0
        for i, msg in enumerate(augmented):
            if msg.get("role") == "system":
                insert_idx = i + 1
                break

        augmented.insert(insert_idx, context_msg)
        return augmented

    def __repr__(self) -> str:
        return f"HystersisMemoryProcessor(user_id='{self.user_id}')"


__all__ = ["HystersisMemoryProcessor"]
