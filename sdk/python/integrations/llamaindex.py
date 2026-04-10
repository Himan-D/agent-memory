"""
LlamaIndex Integration for Agent Memory

This module provides LlamaIndex components that use Agent Memory
as a backend for memory storage and retrieval.

Usage:
    from agentmemory.integrations.llamaindex import AgentMemoryIndex

    # Create from existing memories
    index = AgentMemoryIndex.from_memories(
        memories=client.search("relevant topic"),
        user_id="user-123"
    )

    # Query the index
    retriever = index.as_retriever()
    nodes = retriever.retrieve("What did I learn about AI?")
"""

from typing import Any, Dict, List, Optional, Callable
import requests


class AgentMemoryReader:
    """
    LlamaIndex reader for loading memories from Agent Memory.

    Example:
        >>> reader = AgentMemoryReader(
        ...     base_url="http://localhost:8080",
        ...     user_id="user-123"
        ... )
        >>> documents = reader.load_data(query="AI projects")
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.user_id = user_id
        self.org_id = org_id
        self.agent_id = agent_id

        self._session = requests.Session()
        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    def load_data(
        self,
        query: Optional[str] = None,
        limit: int = 10,
        memory_type: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """
        Load memories as documents.

        Args:
            query: Optional search query to filter memories
            limit: Maximum number of memories to load
            memory_type: Type of memory to filter by

        Returns:
            List of document dictionaries
        """
        if query:
            params = {"q": query, "limit": limit}
            if self.user_id:
                params["user_id"] = self.user_id
            if self.org_id:
                params["org_id"] = self.org_id
            if self.agent_id:
                params["agent_id"] = self.agent_id
            if memory_type:
                params["memory_type"] = memory_type

            resp = self._request("GET", "/search", params=params)
            results = resp.json()
        else:
            if self.user_id:
                resp = self._request(
                    "GET", "/memories", params={"user_id": self.user_id}
                )
            elif self.org_id:
                resp = self._request("GET", "/memories", params={"org_id": self.org_id})
            else:
                resp = self._request("GET", "/memories")
            results = resp.json().get("memories", [])

        documents = []
        for r in results:
            doc = {
                "id": r.get("id", ""),
                "content": r.get("content", ""),
                "metadata": {
                    "memory_type": r.get("type", ""),
                    "category": r.get("category", ""),
                    "user_id": r.get("user_id", ""),
                    "org_id": r.get("org_id", ""),
                    "agent_id": r.get("agent_id", ""),
                    "created_at": r.get("created_at", ""),
                    "score": r.get("score", 1.0),
                },
            }
            documents.append(doc)

        return documents

    def load_memories_by_feedback(
        self,
        feedback_type: str = "positive",
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """
        Load memories by feedback score.

        Args:
            feedback_type: Type of feedback (positive, negative, very_negative)
            limit: Maximum number of memories

        Returns:
            List of high-quality memory documents
        """
        resp = self._request(
            "GET", "/feedback/memories", params={"type": feedback_type, "limit": limit}
        )
        return resp.json()


class AgentMemoryIndex:
    """
    LlamaIndex-compatible index using Agent Memory as backend.

    Example:
        >>> index = AgentMemoryIndex(
        ...     base_url="http://localhost:8080",
        ...     user_id="user-123"
        ... )
        >>> # Query
        >>> retriever = index.as_retriever()
        >>> nodes = retriever.retrieve("What projects did I work on?")
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        memory_type: Optional[str] = None,
        **kwargs,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.user_id = user_id
        self.org_id = org_id
        self.agent_id = agent_id
        self.memory_type = memory_type
        self.kwargs = kwargs

        self._session = requests.Session()
        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    @classmethod
    def from_memories(
        cls,
        memories: List[Dict[str, Any]],
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        **kwargs,
    ) -> "AgentMemoryIndex":
        """
        Create index from existing memory results.

        Args:
            memories: List of memory dictionaries
            user_id: User identifier
            org_id: Organization identifier
            **kwargs: Additional arguments

        Returns:
            AgentMemoryIndex instance
        """
        index = cls(user_id=user_id, org_id=org_id, **kwargs)
        index._memories = memories
        return index

    def query(
        self,
        query: str,
        limit: int = 5,
        threshold: float = 0.5,
        mode: str = "semantic",
    ) -> List[Dict[str, Any]]:
        """
        Query the index.

        Args:
            query: Search query
            limit: Maximum results
            threshold: Minimum score threshold
            mode: Search mode (semantic, hybrid, etc.)

        Returns:
            List of result dictionaries
        """
        params = {
            "q": query,
            "limit": limit,
            "threshold": threshold,
        }
        if self.user_id:
            params["user_id"] = self.user_id
        if self.org_id:
            params["org_id"] = self.org_id
        if self.agent_id:
            params["agent_id"] = self.agent_id
        if self.memory_type:
            params["memory_type"] = self.memory_type

        resp = self._request("GET", "/search", params=params)
        return resp.json()

    def retrieve(self, query: str, **kwargs) -> List[Dict[str, Any]]:
        """Alias for query for LlamaIndex compatibility."""
        return self.query(query, **kwargs)

    def as_retriever(self, **kwargs) -> "AgentMemoryRetriever":
        """Get a retriever for this index."""
        return AgentMemoryRetriever(
            base_url=self.base_url,
            api_key=self.api_key,
            user_id=self.user_id,
            org_id=self.org_id,
            agent_id=self.agent_id,
            **kwargs,
        )

    def as_query_engine(self, **kwargs) -> "AgentMemoryQueryEngine":
        """Get a query engine for this index."""
        return AgentMemoryQueryEngine(index=self, **kwargs)

    def insert_memory(self, content: str, **kwargs) -> Dict[str, Any]:
        """
        Insert a new memory.

        Args:
            content: Memory content
            **kwargs: Additional memory fields (category, metadata, etc.)

        Returns:
            Created memory dictionary
        """
        payload = {
            "content": content,
            "type": self.memory_type or "user",
        }
        if self.user_id:
            payload["user_id"] = self.user_id
        if self.org_id:
            payload["org_id"] = self.org_id
        if self.agent_id:
            payload["agent_id"] = self.agent_id

        for key in ["category", "metadata", "immutable", "expiration_date"]:
            if key in kwargs:
                payload[key] = kwargs[key]

        resp = self._request("POST", "/memories", json=payload)
        return resp.json()

    def delete_memory(self, memory_id: str) -> bool:
        """Delete a memory by ID."""
        try:
            self._request("DELETE", f"/memories/{memory_id}")
            return True
        except Exception:
            return False


class AgentMemoryRetriever:
    """
    LlamaIndex retriever for Agent Memory.

    Example:
        >>> retriever = AgentMemoryRetriever(
        ...     base_url="http://localhost:8080",
        ...     user_id="user-123",
        ...     similarity_top_k=5
        ... )
        >>> nodes = retriever.retrieve("What did I learn?")
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        memory_type: Optional[str] = None,
        similarity_top_k: int = 5,
        score_threshold: float = 0.5,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.user_id = user_id
        self.org_id = org_id
        self.agent_id = agent_id
        self.memory_type = memory_type
        self.similarity_top_k = similarity_top_k
        self.score_threshold = score_threshold

        self._session = requests.Session()
        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    def retrieve(self, query: str) -> List[Dict[str, Any]]:
        """Retrieve relevant memories."""
        params = {
            "q": query,
            "limit": self.similarity_top_k,
            "threshold": self.score_threshold,
        }
        if self.user_id:
            params["user_id"] = self.user_id
        if self.org_id:
            params["org_id"] = self.org_id
        if self.agent_id:
            params["agent_id"] = self.agent_id
        if self.memory_type:
            params["memory_type"] = self.memory_type

        resp = self._request("GET", "/search", params=params)
        return resp.json()


class AgentMemoryQueryEngine:
    """
    LlamaIndex query engine for Agent Memory.

    Example:
        >>> query_engine = AgentMemoryQueryEngine(
        ...     index=index,
        ...     similarity_top_k=5
        ... )
        >>> response = query_engine.query("What projects did I complete?")
    """

    def __init__(
        self,
        index: AgentMemoryIndex,
        similarity_top_k: int = 5,
        score_threshold: float = 0.5,
        response_mode: str = "compact",
    ):
        self.index = index
        self.similarity_top_k = similarity_top_k
        self.score_threshold = score_threshold
        self.response_mode = response_mode

    def query(self, query: str) -> Dict[str, Any]:
        """
        Query the memory index.

        Args:
            query: Search query

        Returns:
            Query response dictionary
        """
        results = self.index.query(
            query,
            limit=self.similarity_top_k,
            threshold=self.score_threshold,
        )

        return {
            "query": query,
            "results": results,
            "response": self._format_response(results),
            "source_nodes": results,
        }

    def _format_response(self, results: List[Dict[str, Any]]) -> str:
        """Format results into a response string."""
        if not results:
            return "No relevant memories found."

        lines = ["Here are relevant memories:\n"]
        for i, r in enumerate(results, 1):
            text = r.get("text", "")
            if not text and r.get("entity"):
                text = r["entity"].get("name", "")
            score = r.get("score", 0)
            lines.append(f"{i}. {text} (relevance: {score:.2f})")

        return "\n".join(lines)


class AgentMemoryMemoryStore:
    """
    LlamaIndex Node storage using Agent Memory.

    Example:
        >>> store = AgentMemoryMemoryStore(
        ...     base_url="http://localhost:8080",
        ...     user_id="user-123"
        ... )
        >>> store.put(node_id, node)
        >>> node = store.get(node_id)
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.user_id = user_id
        self.org_id = org_id

        self._session = requests.Session()
        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    def put(self, key: str, node: Dict[str, Any]) -> None:
        """Store a node."""
        payload = {
            "content": node.get("content", ""),
            "type": "user",
            "metadata": {"node_id": key, **node.get("metadata", {})},
        }
        if self.user_id:
            payload["user_id"] = self.user_id
        if self.org_id:
            payload["org_id"] = self.org_id

        self._request("POST", "/memories", json=payload)

    def get(self, key: str) -> Optional[Dict[str, Any]]:
        """Retrieve a node."""
        results = self._request(
            "GET", "/search", params={"q": f"node_id:{key}", "limit": 1}
        ).json()

        for r in results:
            if r.get("entity", {}).get("properties", {}).get("node_id") == key:
                return r
        return None

    def delete(self, key: str) -> bool:
        """Delete a node."""
        node = self.get(key)
        if node and node.get("id"):
            try:
                self._request("DELETE", f"/memories/{node['id']}")
                return True
            except Exception:
                pass
        return False

    def __contains__(self, key: str) -> bool:
        return self.get(key) is not None


__all__ = [
    "AgentMemoryReader",
    "AgentMemoryIndex",
    "AgentMemoryRetriever",
    "AgentMemoryQueryEngine",
    "AgentMemoryMemoryStore",
]
