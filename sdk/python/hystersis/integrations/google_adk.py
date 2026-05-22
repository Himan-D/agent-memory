"""Hystersis integration for Google Agent Development Kit (ADK)."""
from typing import Any, Dict, List, Optional


class HystersisGoogleADKTool:
    """Memory tool for Google ADK agents."""

    def __init__(self, base_url: str, api_key: str, user_id: str = "default"):
        from hystersis import Hystersis
        self.client = Hystersis(base_url=base_url, api_key=api_key)
        self.user_id = user_id

    def get_tool_declarations(self) -> List[Dict[str, Any]]:
        """Returns Google ADK-compatible tool declarations."""
        return [
            {
                "name": "store_memory",
                "description": "Store information in the agent's long-term memory",
                "parameters": {
                    "type": "OBJECT",
                    "properties": {
                        "content": {"type": "STRING", "description": "The information to store"},
                        "category": {"type": "STRING", "description": "Memory category"},
                    },
                    "required": ["content"],
                },
            },
            {
                "name": "recall_memories",
                "description": "Search the agent's long-term memory",
                "parameters": {
                    "type": "OBJECT",
                    "properties": {
                        "query": {"type": "STRING", "description": "Search query"},
                        "limit": {"type": "INTEGER", "description": "Max results"},
                    },
                    "required": ["query"],
                },
            },
        ]

    def execute(self, tool_name: str, args: Dict[str, Any]) -> str:
        """Execute a tool call from Google ADK."""
        import json

        if tool_name == "store_memory":
            mem = self.client.add(args["content"], user_id=self.user_id, metadata={"category": args.get("category", "")})
            return json.dumps({"stored": True, "id": getattr(mem, "id", "")})

        elif tool_name == "recall_memories":
            results = self.client.search(args["query"], user_id=self.user_id, limit=args.get("limit", 5))
            return json.dumps([{"content": r.get("memory", ""), "score": r.get("score", 0)} for r in results])

        return json.dumps({"error": f"Unknown tool: {tool_name}"})
