"""Hystersis integration for Pydantic AI — type-safe memory dependency injection."""
from typing import Any, Dict, List
from dataclasses import dataclass


@dataclass
class HystersisMemoryDeps:
    """Dependency class for Pydantic AI agents. Inject into agent's deps_type."""
    base_url: str
    api_key: str
    user_id: str = "default"
    agent_id: str = "default"

    def __post_init__(self):
        from hystersis import Hystersis
        self._client = Hystersis(base_url=self.base_url, api_key=self.api_key)

    def store(self, content: str, category: str = "", importance: str = "medium") -> Dict[str, Any]:
        """Store a memory."""
        return self._client.add(
            content,
            user_id=self.user_id,
            agent_id=self.agent_id,
            metadata={"category": category, "importance": importance},
        )

    def recall(self, query: str, limit: int = 5) -> List[Dict[str, Any]]:
        """Search memories."""
        return self._client.search(query, user_id=self.user_id, agent_id=self.agent_id, limit=limit)

    def feedback(self, memory_id: str, feedback_type: str) -> None:
        """Add feedback to a memory."""
        self._client.add_feedback(memory_id, feedback_type)

    def get_context_string(self, query: str, limit: int = 5) -> str:
        """Get memories formatted as a context string for system prompts."""
        results = self.recall(query, limit)
        if not results:
            return ""
        lines = [f"- {r.get('memory', '')}" for r in results]
        return "Relevant memories:\n" + "\n".join(lines)
