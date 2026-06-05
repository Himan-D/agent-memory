"""
Dify Integration for Hystersis

This module provides an external tool provider for Dify workflows,
exposing Hystersis memory operations as Dify-compatible tools.

Usage:
    from hystersis.integrations.dify import HystsersisDifyTool

    tool = HystsersisDifyTool(
        api_key="your-api-key",
        base_url="https://api.hystersis.ai"
    )

    # Get tool schema for Dify registration
    schema = tool.get_tool_schema()

    # Execute a tool call from Dify
    result = tool.execute("add_memory", {
        "content": "User prefers dark mode",
        "user_id": "user-123"
    })
"""

from typing import Any, Dict, List, Optional

import requests


class HystsersisDifyTool:
    """
    External tool provider for Dify workflows.

    Exposes Hystersis memory operations (add, search, get) as Dify-compatible
    tools that can be used in Dify workflow nodes.

    Attributes:
        base_url: Base URL of the Hystersis API
        api_key: API key for authentication
    """

    def __init__(
        self,
        api_key: str,
        base_url: str = "https://api.hystersis.ai",
    ):
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

    def get_tool_schema(self) -> List[Dict[str, Any]]:
        """
        Get Dify-compatible tool schema definitions.

        Returns a list of tool schemas that can be registered with Dify
        as external tool providers.

        Returns:
            List of tool schema dictionaries
        """
        return [
            {
                "name": "add_memory",
                "description": "Store information in long-term memory for future retrieval",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "content": {
                            "type": "string",
                            "description": "The content to store in memory",
                        },
                        "user_id": {
                            "type": "string",
                            "description": "User identifier",
                        },
                    },
                    "required": ["content"],
                },
            },
            {
                "name": "search_memory",
                "description": "Search for relevant memories using semantic search",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "Search query",
                        },
                        "user_id": {
                            "type": "string",
                            "description": "Filter by user",
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
                "name": "get_memories",
                "description": "Get all stored memories for a user",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "user_id": {
                            "type": "string",
                            "description": "User identifier",
                        },
                        "limit": {
                            "type": "integer",
                            "description": "Max results to return",
                            "default": 10,
                        },
                    },
                },
            },
        ]

    def execute(self, tool_name: str, params: Dict[str, Any]) -> Dict[str, Any]:
        """
        Execute a Dify tool call.

        Args:
            tool_name: Name of the tool to execute
            params: Tool parameters from Dify

        Returns:
            Tool result dictionary
        """
        try:
            if tool_name == "add_memory":
                return self._add_memory(params)
            elif tool_name == "search_memory":
                return self._search_memory(params)
            elif tool_name == "get_memories":
                return self._get_memories(params)
            else:
                return {"error": f"Unknown tool: {tool_name}"}
        except Exception as e:
            return {"error": str(e)}

    def _add_memory(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Add a memory via the API."""
        content = params.get("content", "")
        if not content:
            return {"error": "content is required"}

        payload: Dict[str, Any] = {
            "content": content,
            "type": "user",
            "metadata": {"source": "dify"},
        }
        if user_id := params.get("user_id"):
            payload["user_id"] = user_id

        resp = self._request("POST", "/memories", json=payload)
        return resp.json()

    def _search_memory(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Search memories via the API."""
        query = params.get("query", "")
        if not query:
            return {"error": "query is required"}

        search_params: Dict[str, Any] = {
            "q": query,
            "limit": params.get("limit", 5),
        }
        if user_id := params.get("user_id"):
            search_params["user_id"] = user_id

        resp = self._request("GET", "/search", params=search_params)
        return resp.json()

    def _get_memories(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Get all memories via the API."""
        query_params: Dict[str, Any] = {
            "limit": params.get("limit", 10),
        }
        if user_id := params.get("user_id"):
            query_params["user_id"] = user_id

        resp = self._request("GET", "/memories", params=query_params)
        return resp.json()

    def __repr__(self) -> str:
        return "HystsersisDifyTool()"


__all__ = ["HystsersisDifyTool"]
