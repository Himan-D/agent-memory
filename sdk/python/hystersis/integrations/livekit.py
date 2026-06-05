"""
LiveKit Voice Agent Memory Integration for Hystersis

This module provides a memory plugin for LiveKit voice agents that stores
conversation transcripts and retrieves relevant context during live calls.

Usage:
    from hystersis.integrations.livekit import HystersisLiveKitPlugin

    plugin = HystersisLiveKitPlugin(
        api_key="your-api-key",
        user_id="caller-123",
        base_url="https://api.hystersis.ai"
    )

    # Store transcribed speech
    await plugin.on_transcript("I need help with my order", role="user")

    # Retrieve relevant context
    context = await plugin.get_context("order help")
"""

from typing import Any, Dict, List, Optional

import requests


class HystersisLiveKitPlugin:
    """
    Memory plugin for LiveKit voice agents.

    Stores conversation transcripts and retrieves relevant context during
    live calls. Supports both real-time transcript storage and contextual
    memory retrieval for voice-based AI agents.

    Attributes:
        user_id: Unique identifier for the caller/user
        session_id: Optional session identifier for grouping transcripts
        base_url: Base URL of the Hystersis API
        api_key: API key for authentication
    """

    def __init__(
        self,
        api_key: str,
        user_id: str,
        base_url: str = "https://api.hystersis.ai",
        session_id: Optional[str] = None,
        agent_id: Optional[str] = None,
    ):
        self.user_id = user_id
        self.session_id = session_id
        self.agent_id = agent_id
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

    async def on_transcript(self, transcript: str, role: str = "user") -> None:
        """
        Store transcribed speech as memory.

        Called when LiveKit delivers a transcript event. Stores the
        transcript content along with metadata about the speaker role.

        Args:
            transcript: The transcribed speech text
            role: Speaker role - "user" for caller, "assistant" for agent
        """
        metadata: Dict[str, Any] = {
            "source": "livekit",
            "role": role,
        }
        if self.session_id:
            metadata["session_id"] = self.session_id
        if self.agent_id:
            metadata["agent_id"] = self.agent_id

        try:
            self._request(
                "POST",
                "/memories",
                json={
                    "content": transcript,
                    "user_id": self.user_id,
                    "type": "conversation",
                    "metadata": metadata,
                },
            )
        except Exception as e:
            print(f"Warning: Failed to store transcript: {e}")

    async def get_context(self, query: str, limit: int = 5) -> List[str]:
        """
        Retrieve relevant memories for current conversation context.

        Args:
            query: Search query based on current conversation topic
            limit: Maximum number of memories to return

        Returns:
            List of memory content strings relevant to the query
        """
        try:
            resp = self._request(
                "GET",
                "/search",
                params={
                    "q": query,
                    "user_id": self.user_id,
                    "limit": limit,
                },
            )
            results = resp.json()
            memories = []
            for r in results if isinstance(results, list) else results.get("results", []):
                text = r.get("text", "") or r.get("content", "")
                if text:
                    memories.append(text)
            return memories
        except Exception:
            return []

    async def store_conversation_turn(
        self, user_message: str, assistant_response: str
    ) -> None:
        """
        Store a complete conversation turn (user input + agent response).

        Args:
            user_message: The user's transcribed speech
            assistant_response: The agent's response text
        """
        await self.on_transcript(user_message, role="user")
        await self.on_transcript(assistant_response, role="assistant")

    def get_tool_definitions(self) -> List[Dict[str, Any]]:
        """
        Return MCP-compatible tool definitions for LiveKit agents.

        These tools can be registered with LiveKit's function calling
        interface to give the voice agent memory capabilities.

        Returns:
            List of tool definition dictionaries
        """
        return [
            {
                "name": "remember",
                "description": "Store important information from the conversation",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "content": {
                            "type": "string",
                            "description": "The information to remember",
                        }
                    },
                    "required": ["content"],
                },
            },
            {
                "name": "recall",
                "description": "Recall relevant information from past conversations",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "What to search for in memory",
                        }
                    },
                    "required": ["query"],
                },
            },
        ]

    async def handle_tool_call(
        self, tool_name: str, arguments: Dict[str, Any]
    ) -> str:
        """
        Handle a tool call from the LiveKit agent.

        Args:
            tool_name: Name of the tool being called
            arguments: Tool arguments

        Returns:
            Tool result as a string
        """
        if tool_name == "remember":
            content = arguments.get("content", "")
            await self.on_transcript(content, role="system")
            return "Remembered."
        elif tool_name == "recall":
            query = arguments.get("query", "")
            memories = await self.get_context(query)
            if memories:
                return "\n".join(f"- {m}" for m in memories)
            return "No relevant memories found."
        else:
            return f"Unknown tool: {tool_name}"

    def __repr__(self) -> str:
        return f"HystersisLiveKitPlugin(user_id='{self.user_id}')"


__all__ = ["HystersisLiveKitPlugin"]
