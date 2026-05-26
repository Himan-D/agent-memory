"""Hystersis integration for OpenAI Agents SDK."""
from typing import Any, Dict, List


class HystersisOpenAITools:
    """Creates OpenAI-compatible tool definitions for Hystersis memory operations."""

    def __init__(self, base_url: str, api_key: str, user_id: str = "default", agent_id: str = "default"):
        from hystersis import Hystersis
        self.client = Hystersis(base_url=base_url, api_key=api_key)
        self.user_id = user_id
        self.agent_id = agent_id

    def get_tool_definitions(self) -> List[Dict[str, Any]]:
        """Returns tool definitions compatible with OpenAI's function calling schema."""
        return [
            {
                "type": "function",
                "function": {
                    "name": "store_memory",
                    "description": "Store information in long-term memory",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "content": {"type": "string", "description": "The information to remember"},
                            "category": {"type": "string", "description": "Category: preference, fact, decision, skill, goal"},
                            "importance": {"type": "string", "enum": ["critical", "high", "medium", "low"]},
                        },
                        "required": ["content"],
                    },
                },
            },
            {
                "type": "function",
                "function": {
                    "name": "recall_memories",
                    "description": "Search long-term memory for relevant information",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "query": {"type": "string", "description": "What to search for"},
                            "limit": {"type": "number", "description": "Max results (default 5)"},
                        },
                        "required": ["query"],
                    },
                },
            },
            {
                "type": "function",
                "function": {
                    "name": "memory_feedback",
                    "description": "Rate whether a retrieved memory was helpful",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "memory_id": {"type": "string", "description": "ID of the memory"},
                            "feedback": {"type": "string", "enum": ["positive", "negative"]},
                        },
                        "required": ["memory_id", "feedback"],
                    },
                },
            },
        ]

    def execute_tool(self, name: str, args: Dict[str, Any]) -> str:
        """Execute a tool call from an OpenAI agent response."""
        import json

        if name == "store_memory":
            mem = self.client.add(
                args["content"],
                user_id=self.user_id,
                agent_id=self.agent_id,
                metadata={"category": args.get("category", ""), "importance": args.get("importance", "medium")},
            )
            return json.dumps({"stored": True, "id": getattr(mem, "id", "")})

        elif name == "recall_memories":
            results = self.client.search(
                args["query"],
                user_id=self.user_id,
                agent_id=self.agent_id,
                limit=args.get("limit", 5),
            )
            return json.dumps([{"id": r.get("id", ""), "content": r.get("memory", ""), "score": r.get("score", 0)} for r in results])

        elif name == "memory_feedback":
            self.client.add_feedback(args["memory_id"], args["feedback"])
            return json.dumps({"recorded": True})

        return json.dumps({"error": f"Unknown tool: {name}"})
