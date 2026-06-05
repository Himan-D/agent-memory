"""
Agno (Phidata) Integration for Hystersis

Provides memory storage for Agno AI agents with create, read, update,
delete, and search operations backed by the Hystersis API.

Example:
    from hystersis.integrations.agno import HystersisAgnoStorage

    storage = HystersisAgnoStorage(
        base_url="http://localhost:8080",
        api_key="your-key",
        user_id="user-123",
    )

    # Use with Agno agent
    from agno.agent import Agent

    agent = Agent(
        name="Assistant",
        storage=storage,
    )
"""

from typing import Optional

from hystersis import Hystersis


class AgnoMemoryEntry:
    def __init__(
        self,
        content: str,
        id: Optional[str] = None,
        metadata: Optional[dict] = None,
        created_at: Optional[str] = None,
        updated_at: Optional[str] = None,
    ):
        self.id = id
        self.content = content
        self.metadata = metadata or {}
        self.created_at = created_at
        self.updated_at = updated_at


class AgnoSearchResult:
    def __init__(self, id: str, content: str, score: float, metadata: Optional[dict] = None):
        self.id = id
        self.content = content
        self.score = score
        self.metadata = metadata or {}


class HystersisAgnoStorage:
    """
    Agno-compatible memory storage using Hystersis as the backend.
    Implements the Agno Storage protocol for agent memory persistence.
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        user_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        session_id: Optional[str] = None,
    ):
        self._client = Hystersis(base_url=base_url, api_key=api_key)
        self._user_id = user_id
        self._agent_id = agent_id
        self._session_id = session_id

    def create(self, entry: AgnoMemoryEntry) -> str:
        memory = self._client.create_memory(
            content=entry.content,
            metadata=entry.metadata,
            agent_id=self._agent_id,
            session_id=self._session_id,
        )
        return memory["id"]

    def get(self, id: str) -> Optional[AgnoMemoryEntry]:
        try:
            memory = self._client.get_memory(id)
            return AgnoMemoryEntry(
                id=memory["id"],
                content=memory["content"],
                metadata=memory.get("metadata"),
                created_at=memory.get("created_at"),
                updated_at=memory.get("updated_at"),
            )
        except Exception:
            return None

    def update(self, id: str, entry: AgnoMemoryEntry) -> None:
        self._client.update_memory(id, entry.content, entry.metadata)

    def delete(self, id: str) -> None:
        self._client.delete_memory(id)

    def search(self, query: str, limit: int = 10, threshold: float = 0.5) -> list[AgnoSearchResult]:
        results = self._client.search(query, limit=limit, threshold=threshold)
        return [
            AgnoSearchResult(
                id=r.get("memory_id") or r.get("entity", {}).get("id", ""),
                content=r.get("text", ""),
                score=r.get("score", 0.0),
                metadata=r.get("entity", {}).get("properties"),
            )
            for r in results
        ]

    def list(self, limit: int = 100) -> list[AgnoMemoryEntry]:
        result = self._client.list_memories(user_id=self._user_id, agent_id=self._agent_id)
        memories = result.get("memories", [])[:limit]
        return [
            AgnoMemoryEntry(
                id=m["id"],
                content=m["content"],
                metadata=m.get("metadata"),
                created_at=m.get("created_at"),
                updated_at=m.get("updated_at"),
            )
            for m in memories
        ]

    def count(self) -> int:
        result = self._client.list_memories(user_id=self._user_id, agent_id=self._agent_id)
        return result.get("count", 0)

    def drop(self) -> None:
        result = self._client.list_memories(user_id=self._user_id, agent_id=self._agent_id)
        for m in result.get("memories", []):
            try:
                self._client.delete_memory(m["id"])
            except Exception:
                pass

    def clear(self) -> None:
        self.drop()
