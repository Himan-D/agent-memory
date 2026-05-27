"""
Hermes Integration for Hystersis

This module provides a memory provider for Hermes function-calling agents,
exposing Hystersis memory operations as callable functions.

Usage:
    from hystersis.integrations.hermes import HystersisHermesMemory

    memory = HystersisHermesMemory(
        api_key="your-api-key",
        user_id="user-123",
        base_url="https://api.hystersis.ai"
    )

    # Get function definitions for Hermes
    functions = memory.get_functions()

    # Call a function
    result = memory.call_function("remember", {"content": "User prefers dark mode"})
"""

from typing import Any, Dict, List, Optional

import requests


class HystersisHermesMemory:
    """
    Memory provider for Hermes function-calling agents.

    Provides function definitions and execution for Hermes agents
    to store and retrieve persistent memories.

    Attributes:
        user_id: User identifier for memory operations
        base_url: Base URL of the Hystersis API
        api_key: API key for authentication
    """

    def __init__(
        self,
        api_key: str,
        user_id: str,
        base_url: str = "https://api.hystersis.ai",
    ):
        self.user_id = user_id
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

    def get_functions(self) -> List[Dict[str, Any]]:
        """
        Get function definitions for Hermes agent registration.

        Returns function schemas compatible with the Hermes
        function-calling format.

        Returns:
            List of function definition dictionaries
        """
        return [
            {
                "name": "remember",
                "description": "Store important information in long-term memory",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "content": {
                            "type": "string",
                            "description": "The information to remember",
                        },
                    },
                    "required": ["content"],
                },
            },
            {
                "name": "recall",
                "description": "Search and retrieve relevant memories",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "What to search for in memory",
                        },
                        "limit": {
                            "type": "integer",
                            "description": "Max results to return",
                            "default": 5,
                        },
                    },
                    "required": ["query"],
                },
            },
            {
                "name": "forget",
                "description": "Delete a specific memory by ID",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "memory_id": {
                            "type": "string",
                            "description": "ID of the memory to delete",
                        },
                    },
                    "required": ["memory_id"],
                },
            },
        ]

    def call_function(self, name: str, args: Dict[str, Any]) -> Dict[str, Any]:
        """
        Execute a function call from the Hermes agent.

        Args:
            name: Function name to call
            args: Function arguments

        Returns:
            Function result dictionary
        """
        try:
            if name == "remember":
                return self._remember(args)
            elif name == "recall":
                return self._recall(args)
            elif name == "forget":
                return self._forget(args)
            else:
                return {"error": f"Unknown function: {name}"}
        except Exception as e:
            return {"error": str(e)}

    def _remember(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Store content in memory."""
        content = args.get("content", "")
        if not content:
            return {"error": "content is required"}

        resp = self._request(
            "POST",
            "/memories",
            json={
                "content": content,
                "user_id": self.user_id,
                "type": "user",
                "metadata": {"source": "hermes"},
            },
        )
        return resp.json()

    def _recall(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Search memories."""
        query = args.get("query", "")
        if not query:
            return {"error": "query is required"}

        resp = self._request(
            "GET",
            "/search",
            params={
                "q": query,
                "user_id": self.user_id,
                "limit": args.get("limit", 5),
            },
        )
        return resp.json()

    def _forget(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Delete a memory."""
        memory_id = args.get("memory_id", "")
        if not memory_id:
            return {"error": "memory_id is required"}

        self._request("DELETE", f"/memories/{memory_id}")
        return {"status": "deleted", "memory_id": memory_id}

    def __repr__(self) -> str:
        return f"HystersisHermesMemory(user_id='{self.user_id}')"


__all__ = ["HystersisHermesMemory"]
