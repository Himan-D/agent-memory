"""
LangGraph Integration for Agent Memory - Python SDK

Provides memory integration for LangGraph workflows and agents.

Example:
    >>> from agentmemory.integrations.langgraph import HystersisChecker, HystersisUpdater
    >>>
    >>> checker = HystersisChecker(
    ...     user_id='user-123',
    ...     base_url='http://localhost:8080'
    ... )
    >>>
    >>> updater = HystersisUpdater(
    ...     user_id='user-123',
    ...     base_url='http://localhost:8080'
    ... )
"""

from typing import Any, Dict, List, Optional, TypedDict
from hystersis import Hystersis


class CheckMemoryInput(TypedDict, total=False):
    query: str
    limit: Optional[int]
    threshold: Optional[float]
    memory_type: Optional[str]


class CheckMemoryOutput(TypedDict, total=False):
    memories: List[Dict[str, Any]]
    relevant_memories: List[Dict[str, Any]]
    has_relevant_info: bool


class UpdateMemoryInput(TypedDict, total=False):
    content: str
    category: Optional[str]
    metadata: Optional[Dict[str, Any]]
    immutable: Optional[bool]
    expiration_date: Optional[str]


class UpdateMemoryOutput(TypedDict, total=False):
    success: bool
    memory_id: Optional[str]
    error: Optional[str]


class HystersisChecker:
    """Memory checker for LangGraph - retrieves relevant memories."""

    def __init__(
        self,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
    ):
        """
        Initialize the memory checker.

        Args:
            user_id: Optional user ID
            org_id: Optional organization ID
            agent_id: Optional agent ID
            base_url: Base URL of the Agent Memory API
            api_key: Optional API key
        """
        self.client = Hystersis(base_url=base_url, api_key=api_key)
        self.user_id = user_id
        self.org_id = org_id
        self.agent_id = agent_id

    def check(self, input_data: CheckMemoryInput) -> CheckMemoryOutput:
        """
        Check for relevant memories in the store.

        Args:
            input_data: CheckMemoryInput with query and optional filters

        Returns:
            CheckMemoryOutput with memories and relevance info
        """
        results = self.client.search(
            input_data.get("query", ""),
            limit=input_data.get("limit", 10),
            threshold=input_data.get("threshold", 0.5),
            user_id=self.user_id,
            org_id=self.org_id,
            agent_id=self.agent_id,
        )

        memories = [r.get("metadata", {}) for r in results if r.get("metadata")]

        return {
            "memories": memories,
            "relevant_memories": results,
            "has_relevant_info": len(results) > 0,
        }

    def check_with_feedback(self, input_data: CheckMemoryInput) -> CheckMemoryOutput:
        """
        Check with feedback consideration - boosts positive memories.

        Args:
            input_data: CheckMemoryInput with query and optional filters

        Returns:
            CheckMemoryOutput with memories and relevance info
        """
        results = self.client.search(
            input_data.get("query", ""),
            limit=input_data.get("limit", 10),
            threshold=input_data.get("threshold", 0.3),
            user_id=self.user_id,
            org_id=self.org_id,
            agent_id=self.agent_id,
            rerank=True,
        )

        memories = [r.get("metadata", {}) for r in results if r.get("metadata")]

        return {
            "memories": memories,
            "relevant_memories": results,
            "has_relevant_info": len(results) > 0,
        }


class HystersisUpdater:
    """Memory updater for LangGraph - stores new memories."""

    def __init__(
        self,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
    ):
        """
        Initialize the memory updater.

        Args:
            user_id: Optional user ID
            org_id: Optional organization ID
            agent_id: Optional agent ID
            base_url: Base URL of the Agent Memory API
            api_key: Optional API key
        """
        self.client = Hystersis(base_url=base_url, api_key=api_key)
        self.user_id = user_id
        self.org_id = org_id
        self.agent_id = agent_id

    def update(self, input_data: UpdateMemoryInput) -> UpdateMemoryOutput:
        """
        Store a new memory.

        Args:
            input_data: UpdateMemoryInput with content and optional metadata

        Returns:
            UpdateMemoryOutput with success status and memory ID
        """
        try:
            memory = self.client.create_memory(
                content=input_data["content"],
                user_id=self.user_id,
                org_id=self.org_id,
                agent_id=self.agent_id,
                category=input_data.get("category"),
                metadata=input_data.get("metadata"),
                immutable=input_data.get("immutable"),
            )

            return {
                "success": True,
                "memory_id": memory.get("id"),
            }
        except Exception as e:
            return {
                "success": False,
                "error": str(e),
            }

    def update_batch(self, inputs: List[UpdateMemoryInput]) -> Dict[str, Any]:
        """
        Store multiple memories.

        Args:
            inputs: List of UpdateMemoryInput dicts

        Returns:
            Dict with success status and count
        """
        try:
            memories_data = [
                {
                    "content": inp["content"],
                    "user_id": self.user_id,
                    "org_id": self.org_id,
                    "agent_id": self.agent_id,
                    "category": inp.get("category"),
                    "metadata": inp.get("metadata"),
                    "immutable": inp.get("immutable"),
                }
                for inp in inputs
            ]

            result = self.client.batch_create_memories(memories_data)

            return {
                "success": True,
                "count": result.get("count", 0),
                "memory_ids": [m.get("id") for m in result.get("created", [])],
            }
        except Exception as e:
            return {
                "success": False,
                "count": 0,
                "error": str(e),
            }


class LangGraphMemoryState(TypedDict, total=False):
    """State interface for LangGraph integration."""

    messages: List[Dict[str, str]]
    memories: Optional[List[Dict[str, Any]]]
    query: Optional[str]
    response: Optional[str]


class HystersisNode:
    """Memory node for LangGraph StateGraph."""

    def __init__(
        self,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        base_url: str = "http://localhost:8080",
        api_key: Optional[str] = None,
    ):
        """
        Initialize the memory node.

        Args:
            user_id: Optional user ID
            org_id: Optional organization ID
            agent_id: Optional agent ID
            base_url: Base URL of the Agent Memory API
            api_key: Optional API key
        """
        self.checker = HystersisChecker(
            user_id=user_id,
            org_id=org_id,
            agent_id=agent_id,
            base_url=base_url,
            api_key=api_key,
        )
        self.updater = HystersisUpdater(
            user_id=user_id,
            org_id=org_id,
            agent_id=agent_id,
            base_url=base_url,
            api_key=api_key,
        )

    def retrieve_memories(self, state: LangGraphMemoryState) -> Dict[str, Any]:
        """
        Retrieve relevant memories based on last message.

        Args:
            state: LangGraphMemoryState with messages

        Returns:
            Dict with memories
        """
        messages = state.get("messages", [])
        if not messages:
            return {}

        last_message = messages[-1]
        query = last_message.get("content", "")

        result = self.checker.check_with_feedback(
            {
                "query": query,
                "limit": 5,
                "threshold": 0.4,
            }
        )

        return {
            "memories": result.get("memories", []),
        }

    def store_memory(
        self,
        state: LangGraphMemoryState,
        options: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Store important information from the conversation.

        Args:
            state: LangGraphMemoryState with messages
            options: Optional dict with category and metadata

        Returns:
            Empty dict on success
        """
        messages = state.get("messages", [])
        if not messages:
            return {}

        last_message = messages[-1]
        if last_message.get("role") != "assistant":
            return {}

        self.updater.update(
            {
                "content": last_message.get("content", ""),
                "category": (options or {}).get("category", "conversation"),
                "metadata": (options or {}).get("metadata"),
            }
        )

        return {}

    def memory_aware_response(
        self,
        state: LangGraphMemoryState,
        options: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Full memory workflow: retrieve, respond, store.

        Args:
            state: LangGraphMemoryState with messages
            options: Optional dict with retrieve_category and store_category

        Returns:
            Dict with memories
        """
        messages = state.get("messages", [])
        if not messages:
            return {}

        last_message = messages[-1]
        query = last_message.get("content", "")

        result = self.checker.check_with_feedback(
            {
                "query": query,
                "limit": 5,
                "threshold": 0.3,
            }
        )

        memories = result.get("memories", [])
        if memories:
            self.updater.update(
                {
                    "content": query,
                    "category": (options or {}).get(
                        "retrieve_category", "conversation"
                    ),
                }
            )

        return {"memories": memories}


__all__ = [
    "HystersisChecker",
    "HystersisUpdater",
    "HystersisNode",
    "CheckMemoryInput",
    "CheckMemoryOutput",
    "UpdateMemoryInput",
    "UpdateMemoryOutput",
    "LangGraphMemoryState",
]
