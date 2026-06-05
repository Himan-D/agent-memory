"""
LangChain Integration for Agent Memory

This module provides a LangChain memory component that uses Agent Memory
as a backend for storing and retrieving conversation history and memories.

Usage:
    from agentmemory.integrations.langchain import HystersisMemory

    memory = HystersisMemory(
        session_id="user-123",
        memory_type="user",
        api_key="your-api-key",
        base_url="http://localhost:8080"
    )

    # Use with LangChain conversation chain
    from langchain.chains import ConversationChain
    from langchain_openai import ChatOpenAI

    llm = ChatOpenAI(temperature=0)
    conversation = ConversationChain(
        llm=llm,
        memory=memory,
        verbose=True
    )
"""

from typing import Any, Dict, List, Optional
import requests


class HystersisMemory:
    """
    LangChain compatible memory component using Agent Memory backend.

    Attributes:
        session_id: Unique identifier for the conversation session
        memory_type: Type of memory (user, session, conversation, org)
        user_id: Optional user identifier for user-level memory
        agent_id: Optional agent identifier
        base_url: Base URL of the Agent Memory API
        api_key: API key for authentication
        return_messages: Whether to return messages or strings
        input_key: Key to extract input from conversation context
        output_key: Key to extract output from conversation context

    Example:
        >>> memory = HystersisMemory(session_id="test-session")
        >>> memory.save_context({"input": "Hello"}, {"output": "Hi there!"})
        >>> memory.load_memory_variables({})
        {'history': 'Human: Hello\\nAI: Hi there!'}
    """

    def __init__(
        self,
        session_id: str,
        memory_type: str = "session",
        user_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        return_messages: bool = False,
        input_key: str = "input",
        output_key: str = "output",
    ):
        self.session_id = session_id
        self.memory_type = memory_type
        self.user_id = user_id
        self.agent_id = agent_id
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.return_messages = return_messages
        self.input_key = input_key
        self.output_key = output_key

        self._session = requests.Session()
        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make an HTTP request."""
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    @property
    def memories(self) -> List[Dict[str, Any]]:
        """Get all messages from the session."""
        try:
            resp = self._request(
                "GET", f"/sessions/{self.session_id}/messages", params={"limit": 1000}
            )
            return resp.json()
        except Exception:
            return []

    def load_memory_variables(self, inputs: Dict[str, Any]) -> Dict[str, Any]:
        """
        Load memory variables for LangChain.

        Args:
            inputs: Dictionary containing input values

        Returns:
            Dictionary with 'history' key containing conversation history
        """
        messages = self.memories

        if self.return_messages:
            history = messages
        else:
            history = self._format_history(messages)

        return {"history": history}

    def _format_history(self, messages: List[Dict[str, Any]]) -> str:
        """Format messages into a conversation string."""
        formatted = []
        for msg in messages:
            role = msg.get("role", "unknown")
            content = msg.get("content", "")
            if role == "user":
                formatted.append(f"Human: {content}")
            elif role == "assistant":
                formatted.append(f"AI: {content}")
            else:
                formatted.append(f"{role}: {content}")
        return "\n".join(formatted)

    def save_context(self, inputs: Dict[str, Any], outputs: Dict[str, Any]) -> None:
        """
        Save context from the current conversation turn.

        Args:
            inputs: Dictionary with input key containing user message
            outputs: Dictionary with output key containing AI response
        """
        input_text = inputs.get(self.input_key, "")
        output_text = outputs.get(self.output_key, "")

        if input_text:
            self._add_message("user", input_text)
        if output_text:
            self._add_message("assistant", output_text)

    def _add_message(self, role: str, content: str) -> None:
        """Add a message to the session."""
        try:
            self._request(
                "POST",
                f"/sessions/{self.session_id}/messages",
                json={"role": role, "content": content},
            )
        except Exception as e:
            print(f"Warning: Failed to save message: {e}")

    def clear(self) -> None:
        """Clear all messages from the session."""
        try:
            self._request("DELETE", f"/sessions/{self.session_id}")
        except Exception as e:
            print(f"Warning: Failed to clear session: {e}")

    def search_memories(self, query: str, limit: int = 5) -> List[Dict[str, Any]]:
        """
        Search past memories semantically.

        Args:
            query: Search query
            limit: Maximum number of results

        Returns:
            List of memory results with scores
        """
        try:
            resp = self._request("GET", "/search", params={"q": query, "limit": limit})
            return resp.json()
        except Exception:
            return []

    def get_relevant_memories(
        self, query: str, limit: int = 5, threshold: float = 0.5
    ) -> List[str]:
        """
        Get relevant memories as formatted strings.

        Args:
            query: Search query
            limit: Maximum number of results
            threshold: Minimum relevance score

        Returns:
            List of memory content strings
        """
        results = self.search_memories(query, limit)
        memories = []
        for r in results:
            if r.get("score", 0) >= threshold:
                text = r.get("text", "")
                if not text and r.get("entity"):
                    text = r["entity"].get("name", "")
                if text:
                    memories.append(text)
        return memories

    @property
    def chat_memory(self):
        """Alias for memories for ChatVectorStoreRouter compatibility."""
        return self

    def __repr__(self) -> str:
        return f"HystersisMemory(session_id='{self.session_id}')"


class HystersisRetriever:
    """
    LangChain retriever for semantic memory search.

    Example:
        >>> retriever = HystersisRetriever(
        ...     base_url="http://localhost:8080",
        ...     user_id="user-123"
        ... )
        >>> retriever.get_relevant_documents("What did I learn about Python?")
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        memory_type: Optional[str] = None,
        top_k: int = 5,
        score_threshold: float = 0.5,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.user_id = user_id
        self.org_id = org_id
        self.agent_id = agent_id
        self.memory_type = memory_type
        self.top_k = top_k
        self.score_threshold = score_threshold

        self._session = requests.Session()
        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        url = f"{self.base_url}{endpoint}"
        resp = self._session.request(method, url, **kwargs)
        resp.raise_for_status()
        return resp

    def get_relevant_documents(self, query: str) -> List[Dict[str, Any]]:
        """
        Get relevant documents for a query.

        Args:
            query: Search query

        Returns:
            List of relevant document dictionaries
        """
        params = {
            "q": query,
            "limit": self.top_k,
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

    async def aget_relevant_documents(self, query: str) -> List[Dict[str, Any]]:
        """Async version of get_relevant_documents."""
        return self.get_relevant_documents(query)


class HystersisVectorStore:
    """
    LangChain VectorStore implementation using Agent Memory.

    Example:
        >>> from langchain_openai import OpenAIEmbeddings
        >>> embeddings = OpenAIEmbeddings()
        >>> vectorstore = HystersisVectorStore(
        ...     embedding=embeddings,
        ...     base_url="http://localhost:8080"
        ... )
    """

    def __init__(
        self,
        embedding: Any = None,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
    ):
        self.embedding = embedding
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.user_id = user_id
        self.org_id = org_id

        self._session = requests.Session()
        if self.api_key:
            self._session.headers.update({"X-API-Key": self.api_key})

    def add_texts(
        self,
        texts: List[str],
        metadatas: Optional[List[Dict[str, Any]]] = None,
        **kwargs,
    ) -> List[str]:
        """Add texts to the vector store."""
        ids = []
        for i, text in enumerate(texts):
            metadata = metadatas[i] if metadatas and i < len(metadatas) else {}
            if self.user_id:
                metadata["user_id"] = self.user_id
            if self.org_id:
                metadata["org_id"] = self.org_id

            resp = self._session.post(
                f"{self.base_url}/memories",
                json={
                    "content": text,
                    "type": "user",
                    "metadata": metadata,
                },
            )
            ids.append(resp.json().get("id", ""))
        return ids

    def similarity_search(
        self, query: str, k: int = 4, filter: Optional[Dict[str, Any]] = None, **kwargs
    ) -> List[Dict[str, Any]]:
        """Perform similarity search."""
        params = {"q": query, "limit": k}
        if filter:
            if "user_id" in filter:
                params["user_id"] = filter["user_id"]
            if "category" in filter:
                params["category"] = filter["category"]

        resp = self._session.get(f"{self.base_url}/search", params=params)
        return resp.json()

    def similarity_search_with_score(
        self, query: str, k: int = 4, **kwargs
    ) -> List[tuple]:
        """Perform similarity search with scores."""
        results = self.similarity_search(query, k)
        return [(r, r.get("score", 0.0)) for r in results]


__all__ = [
    "HystersisMemory",
    "HystersisRetriever",
    "HystersisVectorStore",
]
