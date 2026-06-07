"""
Hystersis Async Python SDK

Persistent memory infrastructure for AI agents.
Give your agents memory that adapts and compounds over time.

Example:
    import asyncio
    from hystersis import AsyncHystersis

    async def main():
        client = AsyncHystersis(
            base_url="https://api.hystersis.ai",
            api_key="your-key"
        )

        # Create a session for your agent
        session = await client.sessions.create(agent_id="assistant-bot")

        # Store a semantic memory
        memory = await client.memories.create(
            content="User is interested in machine learning and AI",
            user_id="user-123"
        )

        # Later, search semantically
        results = await client.memories.search("deep learning")

        await client.close()

    asyncio.run(main())
"""

import asyncio
import logging
import os
from contextlib import asynccontextmanager
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from typing import Any, Awaitable, Callable, Dict, List, Optional, TypeVar

import httpx

logger = logging.getLogger(__name__)

T = TypeVar("T")


class MemoryType(str, Enum):
    CONVERSATION = "conversation"
    SESSION = "session"
    USER = "user"
    ORG = "org"


class FeedbackType(str, Enum):
    POSITIVE = "positive"
    NEGATIVE = "negative"
    VERY_NEGATIVE = "very_negative"


class ImportanceLevel(str, Enum):
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"


class MemoryLinkType(str, Enum):
    PARENT = "parent"
    RELATED = "related"
    REPLY = "reply"
    CITE = "cite"


class MemberRole(str, Enum):
    ADMIN = "admin"
    CONTRIBUTOR = "contributor"
    READER = "reader"


class ReviewStatus(str, Enum):
    PENDING = "pending"
    APPROVED = "approved"
    REJECTED = "rejected"


class CompressionMode(str, Enum):
    EXTRACT = "extract"
    BALANCED = "balanced"
    AGGRESSIVE = "aggressive"


class TierPolicy(str, Enum):
    AGGRESSIVE = "aggressive"
    BALANCED = "balanced"
    CONSERVATIVE = "conservative"


class SearchMode(str, Enum):
    VECTOR = "vector"
    SPREADING = "spreading"
    HYBRID = "hybrid"


class ChainStatus(str, Enum):
    ACTIVE = "active"
    PAUSED = "paused"
    COMPLETED = "completed"
    FAILED = "failed"


# ==================== Exceptions ====================


class HystersisError(Exception):
    """Base exception for Hystersis errors."""

    def __init__(
        self,
        message: str,
        status_code: Optional[int] = None,
        response: Optional[httpx.Response] = None,
    ):
        super().__init__(message)
        self.message = message
        self.status_code = status_code
        self.response = response


class AuthenticationError(HystersisError):
    """Raised when authentication fails."""

    pass


class NotFoundError(HystersisError):
    """Raised when a resource is not found."""

    pass


class ValidationError(HystersisError):
    """Raised when input validation fails."""

    pass


class RateLimitError(HystersisError):
    """Raised when rate limit is exceeded."""

    pass


class ServerError(HystersisError):
    """Raised when server returns 5xx error."""

    pass


# ==================== Configuration ====================


@dataclass
class RetryConfig:
    """Configuration for automatic retry with exponential backoff."""

    max_retries: int = 3
    base_delay: float = 1.0
    max_delay: float = 60.0
    exponential_base: float = 2.0
    retry_on_status_codes: tuple = (429, 500, 502, 503, 504)


@dataclass
class RateLimitConfig:
    """Configuration for rate limiting."""

    requests_per_second: float = 10.0
    burst_size: int = 20


@dataclass
class TimeoutConfig:
    """Configuration for request timeouts."""

    connect: float = 10.0
    read: float = 30.0
    write: float = 30.0
    pool: float = 5.0


@dataclass
class HystersisConfig:
    """Configuration for the Hystersis client."""

    base_url: str = "https://api.hystersis.ai"
    api_key: Optional[str] = None
    timeout: TimeoutConfig = field(default_factory=TimeoutConfig)
    retry: RetryConfig = field(default_factory=RetryConfig)
    rate_limit: RateLimitConfig = field(default_factory=RateLimitConfig)
    max_connections: int = 100
    max_keepalive_connections: int = 20
    follow_redirects: bool = True


# ==================== Interceptors ====================

RequestInterceptor = Callable[[httpx.Request], Awaitable[httpx.Request]]
ResponseInterceptor = Callable[[httpx.Response], Awaitable[httpx.Response]]


# ==================== Rate Limiter ====================


class TokenBucketRateLimiter:
    """Token bucket rate limiter for API requests."""

    def __init__(self, requests_per_second: float, burst_size: int):
        self.rate = requests_per_second
        self.burst = burst_size
        self.tokens = float(burst_size)
        self.last_update = datetime.now()
        self._lock = asyncio.Lock()

    async def acquire(self):
        async with self._lock:
            now = datetime.now()
            elapsed = (now - self.last_update).total_seconds()
            self.tokens = min(self.burst, self.tokens + elapsed * self.rate)
            self.last_update = now

            if self.tokens < 1:
                wait_time = (1 - self.tokens) / self.rate
                await asyncio.sleep(wait_time)
                self.tokens = 0
            else:
                self.tokens -= 1


# ==================== Base Client ====================


class AsyncHystersis:
    """
    Async Python SDK for Hystersis - Persistent Memory Infrastructure.

    Provides async interface to store and retrieve agent memories,
    including conversation history, knowledge graph entities, and
    semantic search capabilities.
    """

    def __init__(
        self,
        base_url: str = "https://api.hystersis.ai",
        api_key: Optional[str] = None,
        timeout: Optional[TimeoutConfig] = None,
        retry: Optional[RetryConfig] = None,
        rate_limit: Optional[RateLimitConfig] = None,
        max_connections: int = 100,
        max_keepalive_connections: int = 20,
        request_interceptors: Optional[List[RequestInterceptor]] = None,
        response_interceptors: Optional[List[ResponseInterceptor]] = None,
    ):
        self.config = HystersisConfig(
            base_url=base_url,
            api_key=api_key
            or os.environ.get("HYSTERSIS_API_KEY")
            or os.environ.get("AGENT_MEMORY_API_KEY"),
            timeout=timeout or TimeoutConfig(),
            retry=retry or RetryConfig(),
            rate_limit=rate_limit or RateLimitConfig(),
            max_connections=max_connections,
            max_keepalive_connections=max_keepalive_connections,
        )

        self._request_interceptors = request_interceptors or []
        self._response_interceptors = response_interceptors or []
        self._rate_limiter = TokenBucketRateLimiter(
            self.config.rate_limit.requests_per_second,
            self.config.rate_limit.burst_size,
        )
        self._client: Optional[httpx.AsyncClient] = None
        self._closed = False

    async def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(
                timeout=httpx.Timeout(
                    connect=self.config.timeout.connect,
                    read=self.config.timeout.read,
                    write=self.config.timeout.write,
                    pool=self.config.timeout.pool,
                ),
                limits=httpx.Limits(
                    max_connections=self.config.max_connections,
                    max_keepalive_connections=self.config.max_keepalive_connections,
                ),
                follow_redirects=self.config.follow_redirects,
            )
        return self._client

    async def _build_request(
        self, method: str, endpoint: str, **kwargs
    ) -> httpx.Request:
        """Build request with interceptors."""
        url = f"{self.config.base_url.rstrip('/')}{endpoint}"

        headers = kwargs.pop("headers", httpx.Headers())
        headers.setdefault("Content-Type", "application/json")

        if self.config.api_key:
            headers.setdefault("X-API-Key", self.config.api_key)

        request = httpx.Request(method, url, headers=headers, **kwargs)

        for interceptor in self._request_interceptors:
            request = await interceptor(request)

        return request

    async def _send_request(self, request: httpx.Request) -> httpx.Response:
        """Send request with rate limiting and retries."""
        await self._rate_limiter.acquire()

        client = await self._get_client()

        retry_config = self.config.retry
        last_exception = None

        for attempt in range(retry_config.max_retries + 1):
            try:
                response = await client.send(request)

                for interceptor in self._response_interceptors:
                    response = await interceptor(response)

                if response.status_code == 401:
                    raise AuthenticationError(
                        "Invalid or missing API key", status_code=401, response=response
                    )
                elif response.status_code == 403:
                    raise AuthenticationError(
                        "Forbidden: " + response.text,
                        status_code=403,
                        response=response,
                    )
                elif response.status_code == 404:
                    raise NotFoundError(
                        f"Resource not found: {request.url}",
                        status_code=404,
                        response=response,
                    )
                elif response.status_code == 429:
                    if attempt < retry_config.max_retries:
                        delay = min(
                            retry_config.base_delay
                            * (retry_config.exponential_base**attempt),
                            retry_config.max_delay,
                        )
                        logger.warning(
                            f"Rate limited, retrying in {delay:.2f}s (attempt {attempt + 1})"
                        )
                        await asyncio.sleep(delay)
                        continue
                    raise RateLimitError(
                        "Rate limit exceeded", status_code=429, response=response
                    )
                elif response.status_code >= 500 and attempt < retry_config.max_retries:
                    if response.status_code in retry_config.retry_on_status_codes:
                        delay = min(
                            retry_config.base_delay
                            * (retry_config.exponential_base**attempt),
                            retry_config.max_delay,
                        )
                        logger.warning(
                            f"Server error {response.status_code}, retrying in {delay:.2f}s (attempt {attempt + 1})"
                        )
                        await asyncio.sleep(delay)
                        continue
                    raise ServerError(
                        f"Server error: {response.status_code}",
                        status_code=response.status_code,
                        response=response,
                    )
                elif response.status_code == 400:
                    raise ValidationError(
                        response.text, status_code=400, response=response
                    )

                response.raise_for_status()
                return response

            except (httpx.TimeoutException, httpx.NetworkError) as e:
                last_exception = e
                if attempt < retry_config.max_retries:
                    delay = min(
                        retry_config.base_delay
                        * (retry_config.exponential_base**attempt),
                        retry_config.max_delay,
                    )
                    logger.warning(
                        f"Network error, retrying in {delay:.2f}s (attempt {attempt + 1})"
                    )
                    await asyncio.sleep(delay)
                    continue
                raise HystersisError(f"Network error: {str(e)}")

        raise HystersisError(
            f"Request failed after {retry_config.max_retries + 1} attempts: {last_exception}"
        )

    async def request(
        self,
        method: str,
        endpoint: str,
        params: Optional[Dict[str, Any]] = None,
        json: Optional[Dict[str, Any]] = None,
        data: Optional[Any] = None,
    ) -> Dict[str, Any]:
        """Make an HTTP request."""
        request = await self._build_request(
            method,
            endpoint,
            params=params,
            json=json,
            data=data,
        )
        response = await self._send_request(request)
        return response.json()

    async def request_raw(
        self,
        method: str,
        endpoint: str,
        params: Optional[Dict[str, Any]] = None,
        json: Optional[Dict[str, Any]] = None,
    ) -> str:
        """Make an HTTP request and return raw text."""
        request = await self._build_request(method, endpoint, params=params, json=json)
        response = await self._send_request(request)
        return response.text

    def add_request_interceptor(self, interceptor: RequestInterceptor):
        """Add a request interceptor."""
        self._request_interceptors.append(interceptor)

    def add_response_interceptor(self, interceptor: ResponseInterceptor):
        """Add a response interceptor."""
        self._response_interceptors.append(interceptor)

    async def close(self):
        """Close the HTTP client."""
        if self._client:
            await self._client.aclose()
            self._client = None
            self._closed = True

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.close()

    # ==================== Health ====================

    async def health(self) -> Dict[str, str]:
        """Check API health status."""
        return await self.request("GET", "/health")

    async def ready(self) -> Dict[str, Any]:
        """Check API readiness including dependency health."""
        return await self.request("GET", "/ready")

    # ==================== Sessions ====================

    async def create_session(
        self,
        agent_id: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Create a new conversation session for an agent."""
        payload = {"agent_id": agent_id}
        if metadata:
            payload["metadata"] = metadata
        return await self.request("POST", "/sessions", json=payload)

    async def get_session(self, session_id: str) -> Dict[str, Any]:
        """Get session details and messages."""
        return await self.request("GET", f"/sessions/{session_id}")

    async def delete_session(self, session_id: str) -> Dict[str, str]:
        """Delete a session and all its messages."""
        return await self.request("DELETE", f"/sessions/{session_id}")

    async def list_sessions(
        self,
        agent_id: Optional[str] = None,
        limit: int = 50,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List sessions."""
        params = {"limit": limit, "offset": offset}
        if agent_id:
            params["agent_id"] = agent_id
        return await self.request("GET", "/sessions", params=params)

    async def add_message(
        self,
        session_id: str,
        role: str,
        content: str,
    ) -> Dict[str, str]:
        """Add a message to a session conversation."""
        if role not in ("user", "assistant", "system", "tool"):
            raise ValidationError(f"Invalid role: {role}")
        return await self.request(
            "POST",
            f"/sessions/{session_id}/messages",
            json={"role": role, "content": content},
        )

    async def get_messages(
        self,
        session_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """Get conversation messages for a session."""
        return await self.request(
            "GET", f"/sessions/{session_id}/messages", params={"limit": limit}
        )

    # Aliases for backward compatibility
    sessions_create = create_session
    sessions_get = get_session
    sessions_delete = delete_session
    sessions_list = list_sessions
    sessions_messages_add = add_message
    sessions_messages_list = get_messages

    async def get_context(
        self, session_id: str, limit: int = 50
    ) -> List[Dict[str, Any]]:
        """Get conversation context for a session."""
        return await self.get_messages(session_id, limit)

    # ==================== Memories ====================

    async def create_memory(
        self,
        content: str,
        memory_type: str = MemoryType.USER,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        session_id: Optional[str] = None,
        category: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        immutable: bool = False,
        expiration_date: Optional[datetime] = None,
        tags: Optional[List[str]] = None,
        importance: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Create a new memory."""
        if not content:
            raise ValidationError("content is required")

        payload = {"content": content, "type": memory_type}

        if user_id:
            payload["user_id"] = user_id
        if org_id:
            payload["org_id"] = org_id
        if agent_id:
            payload["agent_id"] = agent_id
        if session_id:
            payload["session_id"] = session_id
        if category:
            payload["category"] = category
        if metadata:
            payload["metadata"] = metadata
        if immutable:
            payload["immutable"] = True
        if expiration_date:
            payload["expiration_date"] = expiration_date.isoformat()
        if tags:
            payload["tags"] = tags
        if importance:
            payload["importance"] = importance

        return await self.request("POST", "/memories", json=payload)

    async def memories_get(self, memory_id: str) -> Dict[str, Any]:
        """Get a specific memory by ID."""
        return await self.request("GET", f"/memories/{memory_id}")

    async def memories_update(
        self,
        memory_id: str,
        content: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Update a memory's content and/or metadata."""
        if not content:
            raise ValidationError("content is required")
        payload = {"content": content}
        if metadata:
            payload["metadata"] = metadata
        return await self.request("PUT", f"/memories/{memory_id}", json=payload)

    async def memories_delete(self, memory_id: str) -> Dict[str, str]:
        """Delete a specific memory."""
        return await self.request("DELETE", f"/memories/{memory_id}")

    async def memories_list(
        self,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        category: Optional[str] = None,
        limit: int = 100,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List memories with optional filters."""
        params = {"limit": limit, "offset": offset}
        if user_id:
            params["user_id"] = user_id
        if org_id:
            params["org_id"] = org_id
        if agent_id:
            params["agent_id"] = agent_id
        if category:
            params["category"] = category
        return await self.request("GET", "/memories", params=params)

    async def memories_search(
        self,
        query: str,
        limit: int = 10,
        threshold: float = 0.5,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        agent_id: Optional[str] = None,
        category: Optional[str] = None,
        memory_type: Optional[str] = None,
        rerank: bool = False,
        rerank_top_k: int = 20,
        mode: str = "vector",
    ) -> List[Dict[str, Any]]:
        """Semantic search over stored memories."""
        params = {
            "q": query,
            "limit": min(max(limit, 1), 100),
            "threshold": min(max(threshold, 0.0), 1.0),
            "rerank": rerank,
            "rerank_top_k": rerank_top_k,
            "mode": mode,
        }
        if user_id:
            params["user_id"] = user_id
        if org_id:
            params["org_id"] = org_id
        if agent_id:
            params["agent_id"] = agent_id
        if category:
            params["category"] = category
        if memory_type:
            params["memory_type"] = memory_type

        return await self.request("GET", "/search", params=params)

    async def memories_history(self, memory_id: str) -> List[Dict[str, Any]]:
        """Get modification history for a memory."""
        return await self.request("GET", f"/memories/{memory_id}/history")

    async def memories_set_expiration(
        self,
        memory_id: str,
        expiration_date: datetime,
    ) -> Dict[str, str]:
        """Set an expiration date for a memory."""
        return await self.request(
            "POST",
            f"/memories/{memory_id}/expire",
            json={"expiration_date": expiration_date.isoformat()},
        )

    async def memories_link_to_entity(
        self,
        memory_id: str,
        entity_id: str,
    ) -> Dict[str, str]:
        """Link a memory to an entity."""
        return await self.request("POST", f"/memories/{memory_id}/link/{entity_id}")

    async def memories_batch_create(
        self,
        memories: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """Create multiple memories in one request."""
        if len(memories) > 1000:
            raise ValidationError("Maximum 1000 memories per batch")
        return await self.request(
            "POST", "/memories/batch", json={"memories": memories}
        )

    async def memories_batch_update(
        self,
        memory_ids: List[str],
        action: str = "update",
        content: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Batch update, archive, or delete memories."""
        if len(memory_ids) > 1000:
            raise ValidationError("Maximum 1000 IDs per batch")
        if action not in ("update", "archive", "delete"):
            raise ValidationError("action must be update, archive, or delete")
        payload = {"ids": memory_ids, "action": action}
        if content:
            payload["content"] = content
        if metadata:
            payload["metadata"] = metadata
        return await self.request("PUT", "/memories/batch-update", json=payload)

    async def memories_bulk_delete(
        self,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        category: Optional[str] = None,
    ) -> Dict[str, int]:
        """Delete all memories matching a filter."""
        if not user_id and not org_id and not category:
            raise ValidationError("At least one filter is required")
        payload = {}
        if user_id:
            payload["user_id"] = user_id
        if org_id:
            payload["org_id"] = org_id
        if category:
            payload["category"] = category
        return await self.request("DELETE", "/memories/bulk-delete", json=payload)

    # ==================== Feedback ====================

    async def feedback_add(
        self,
        memory_id: str,
        feedback_type: str,
        comment: Optional[str] = None,
        user_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Provide feedback on a memory."""
        if feedback_type not in (
            FeedbackType.POSITIVE,
            FeedbackType.NEGATIVE,
            FeedbackType.VERY_NEGATIVE,
        ):
            raise ValidationError(f"Invalid feedback_type: {feedback_type}")
        payload = {"memory_id": memory_id, "type": feedback_type}
        if comment:
            payload["comment"] = comment
        if user_id:
            payload["user_id"] = user_id
        return await self.request("POST", "/feedback", json=payload)

    async def feedback_get_memories(
        self,
        feedback_type: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """Get memories filtered by feedback type."""
        return await self.request(
            "GET", "/feedback/memories", params={"type": feedback_type, "limit": limit}
        )

    # ==================== Entities ====================

    async def entities_create(
        self,
        name: str,
        entity_type: str,
        properties: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Create a knowledge graph entity."""
        payload = {"name": name, "type": entity_type}
        if properties:
            payload["properties"] = properties
        return await self.request("POST", "/entities", json=payload)

    async def entities_get(self, entity_id: str) -> Dict[str, Any]:
        """Get an entity by ID."""
        return await self.request("GET", f"/entities/{entity_id}")

    async def entities_list(
        self,
        entity_type: Optional[str] = None,
        limit: int = 100,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List entities with optional type filter."""
        params = {"limit": limit, "offset": offset}
        if entity_type:
            params["entity_type"] = entity_type
        return await self.request("GET", "/entities", params=params)

    async def entities_update(
        self,
        entity_id: str,
        name: Optional[str] = None,
        entity_type: Optional[str] = None,
        properties: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Update an entity."""
        payload = {}
        if name:
            payload["name"] = name
        if entity_type:
            payload["type"] = entity_type
        if properties:
            payload["properties"] = properties
        return await self.request("PUT", f"/entities/{entity_id}", json=payload)

    async def entities_delete(self, entity_id: str) -> Dict[str, str]:
        """Delete an entity."""
        return await self.request("DELETE", f"/entities/{entity_id}")

    async def entities_get_memories(
        self,
        entity_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """Get all memories linked to an entity."""
        return await self.request(
            "GET", f"/entities/{entity_id}/memories", params={"limit": limit}
        )

    async def entities_get_relations(
        self,
        entity_id: str,
        relation_type: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """Get all relations for an entity."""
        params = {}
        if relation_type:
            params["type"] = relation_type
        return await self.request(
            "GET", f"/entities/{entity_id}/relations", params=params
        )

    # ==================== Relations ====================

    async def relations_create(
        self,
        from_id: str,
        to_id: str,
        relation_type: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, str]:
        """Create a typed relationship between two entities."""
        payload = {"from_id": from_id, "to_id": to_id, "type": relation_type}
        if metadata:
            payload["metadata"] = metadata
        return await self.request("POST", "/relations", json=payload)

    async def relations_delete(self, from_id: str, to_id: str) -> Dict[str, str]:
        """Delete a relation between two entities."""
        return await self.request("DELETE", f"/relations/{from_id}/{to_id}")

    # ==================== Graph ====================

    async def graph_query(
        self,
        cypher: str,
        params: Optional[Dict[str, Any]] = None,
    ) -> List[Dict[str, Any]]:
        """Execute a raw Cypher query on the knowledge graph."""
        payload = {"cypher": cypher}
        if params:
            payload["params"] = params
        return await self.request("POST", "/graph/query", json=payload)

    async def graph_traverse(
        self,
        entity_id: str,
        depth: int = 3,
    ) -> Dict[str, Any]:
        """Traverse graph from an entity."""
        return await self.request(
            "GET", f"/graph/traverse/{entity_id}", params={"depth": depth}
        )

    # ==================== Projects ====================

    async def projects_create(
        self,
        name: str,
        description: Optional[str] = None,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        settings: Optional[Dict[str, Any]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Create a new project."""
        payload = {"name": name}
        if description:
            payload["description"] = description
        if user_id:
            payload["user_id"] = user_id
        if org_id:
            payload["org_id"] = org_id
        if settings:
            payload["settings"] = settings
        if metadata:
            payload["metadata"] = metadata
        return await self.request("POST", "/projects", json=payload)

    async def projects_get(self, project_id: str) -> Dict[str, Any]:
        """Get a project by ID."""
        return await self.request("GET", f"/projects/{project_id}")

    async def projects_list(
        self,
        user_id: Optional[str] = None,
        org_id: Optional[str] = None,
        limit: int = 50,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List projects."""
        params = {"limit": limit, "offset": offset}
        if user_id:
            params["user_id"] = user_id
        if org_id:
            params["org_id"] = org_id
        return await self.request("GET", "/projects", params=params)

    async def projects_update(
        self,
        project_id: str,
        name: Optional[str] = None,
        description: Optional[str] = None,
        settings: Optional[Dict[str, Any]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Update a project."""
        payload = {}
        if name:
            payload["name"] = name
        if description:
            payload["description"] = description
        if settings:
            payload["settings"] = settings
        if metadata:
            payload["metadata"] = metadata
        return await self.request("PUT", f"/projects/{project_id}", json=payload)

    async def projects_delete(self, project_id: str) -> Dict[str, str]:
        """Delete a project."""
        return await self.request("DELETE", f"/projects/{project_id}")

    # ==================== Webhooks ====================

    async def webhooks_create(
        self,
        url: str,
        events: List[str],
        project_id: Optional[str] = None,
        secret: Optional[str] = None,
        active: bool = True,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Create a new webhook."""
        payload = {"url": url, "events": events, "active": active}
        if project_id:
            payload["project_id"] = project_id
        if secret:
            payload["secret"] = secret
        if metadata:
            payload["metadata"] = metadata
        return await self.request("POST", "/webhooks", json=payload)

    async def webhooks_get(self, webhook_id: str) -> Dict[str, Any]:
        """Get a webhook by ID."""
        return await self.request("GET", f"/webhooks/{webhook_id}")

    async def webhooks_list(
        self,
        project_id: Optional[str] = None,
        limit: int = 50,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List webhooks."""
        params = {"limit": limit, "offset": offset}
        if project_id:
            params["project_id"] = project_id
        return await self.request("GET", "/webhooks", params=params)

    async def webhooks_update(
        self,
        webhook_id: str,
        url: Optional[str] = None,
        events: Optional[List[str]] = None,
        active: Optional[bool] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Update a webhook."""
        payload = {}
        if url:
            payload["url"] = url
        if events:
            payload["events"] = events
        if active is not None:
            payload["active"] = active
        if metadata:
            payload["metadata"] = metadata
        return await self.request("PUT", f"/webhooks/{webhook_id}", json=payload)

    async def webhooks_delete(self, webhook_id: str) -> Dict[str, str]:
        """Delete a webhook."""
        return await self.request("DELETE", f"/webhooks/{webhook_id}")

    async def webhooks_test(self, webhook_id: str) -> Dict[str, Any]:
        """Test a webhook."""
        return await self.request("POST", f"/webhooks/{webhook_id}/test")

    # ==================== Skills ====================

    async def skills_create(
        self,
        name: str,
        trigger: str,
        action: str,
        domain: str = "general",
        confidence: float = 0.8,
        tags: Optional[List[str]] = None,
        examples: Optional[List[str]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Create a new skill."""
        payload = {
            "name": name,
            "trigger": trigger,
            "action": action,
            "domain": domain,
            "confidence": confidence,
        }
        if tags:
            payload["tags"] = tags
        if examples:
            payload["examples"] = examples
        if metadata:
            payload["metadata"] = metadata
        return await self.request("POST", "/skills", json=payload)

    async def skills_get(self, skill_id: str) -> Dict[str, Any]:
        """Get a skill by ID."""
        return await self.request("GET", f"/skills/{skill_id}")

    async def skills_list(
        self,
        domain: Optional[str] = None,
        limit: int = 50,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List skills."""
        params = {"limit": limit, "offset": offset}
        if domain:
            params["domain"] = domain
        return await self.request("GET", "/skills", params=params)

    async def skills_search(
        self,
        trigger: Optional[str] = None,
        domain: Optional[str] = None,
        limit: int = 20,
    ) -> Dict[str, Any]:
        """Search skills by trigger or domain."""
        params = {"limit": limit}
        if trigger:
            params["trigger"] = trigger
        if domain:
            params["domain"] = domain
        return await self.request("GET", "/skills/search", params=params)

    async def skills_update(
        self,
        skill_id: str,
        name: Optional[str] = None,
        trigger: Optional[str] = None,
        action: Optional[str] = None,
        domain: Optional[str] = None,
        confidence: Optional[float] = None,
        tags: Optional[List[str]] = None,
    ) -> Dict[str, Any]:
        """Update a skill."""
        payload = {}
        if name:
            payload["name"] = name
        if trigger:
            payload["trigger"] = trigger
        if action:
            payload["action"] = action
        if domain:
            payload["domain"] = domain
        if confidence is not None:
            payload["confidence"] = confidence
        if tags:
            payload["tags"] = tags
        return await self.request("PUT", f"/skills/{skill_id}", json=payload)

    async def skills_delete(self, skill_id: str) -> Dict[str, str]:
        """Delete a skill."""
        return await self.request("DELETE", f"/skills/{skill_id}")

    async def skills_use(self, skill_id: str) -> Dict[str, bool]:
        """Increment skill usage count."""
        return await self.request("POST", f"/skills/{skill_id}/use")

    async def skills_suggest(
        self,
        trigger: str,
        context: Optional[str] = None,
        limit: int = 5,
    ) -> Dict[str, Any]:
        """Get skill suggestions for a trigger."""
        payload = {"trigger": trigger, "limit": limit}
        if context:
            payload["context"] = context
        return await self.request("POST", "/skills/suggest", json=payload)

    async def skills_synthesize(
        self,
        skill_ids: List[str],
    ) -> Dict[str, Any]:
        """Synthesize multiple skills into a generalized skill."""
        if len(skill_ids) < 2:
            raise ValidationError("Need at least 2 skills to synthesize")
        return await self.request(
            "POST", "/skills/synthesize", json={"skill_ids": skill_ids}
        )

    async def skills_extract(
        self,
        content: str,
        user_id: Optional[str] = None,
        agent_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Extract skills from content using LLM."""
        payload = {"content": content}
        if user_id:
            payload["user_id"] = user_id
        if agent_id:
            payload["agent_id"] = agent_id
        return await self.request("POST", "/skills/extract", json=payload)

    # ==================== Skill Chains ====================

    async def chains_create(
        self,
        name: str,
        trigger: str,
        steps: List[Dict[str, Any]],
        conditions: Optional[List[Dict[str, Any]]] = None,
    ) -> Dict[str, Any]:
        """Create a new skill chain."""
        payload = {"name": name, "trigger": trigger, "steps": steps}
        if conditions:
            payload["conditions"] = conditions
        return await self.request("POST", "/chains", json=payload)

    async def chains_get(self, chain_id: str) -> Dict[str, Any]:
        """Get a chain by ID."""
        return await self.request("GET", f"/chains/{chain_id}")

    async def chains_list(
        self,
        status: Optional[str] = None,
        limit: int = 50,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List skill chains."""
        params = {"limit": limit, "offset": offset}
        if status:
            params["status"] = status
        return await self.request("GET", "/chains", params=params)

    async def chains_update(
        self,
        chain_id: str,
        name: Optional[str] = None,
        status: Optional[str] = None,
        steps: Optional[List[Dict[str, Any]]] = None,
    ) -> Dict[str, Any]:
        """Update a skill chain."""
        payload = {}
        if name:
            payload["name"] = name
        if status:
            payload["status"] = status
        if steps:
            payload["steps"] = steps
        return await self.request("PUT", f"/chains/{chain_id}", json=payload)

    async def chains_delete(self, chain_id: str) -> Dict[str, str]:
        """Delete a skill chain."""
        return await self.request("DELETE", f"/chains/{chain_id}")

    async def chains_execute(
        self,
        chain_id: str,
        context: Dict[str, Any],
        timeout_ms: Optional[int] = None,
    ) -> Dict[str, Any]:
        """Execute a skill chain."""
        payload = {"context": context}
        if timeout_ms:
            payload["timeout_ms"] = timeout_ms
        return await self.request("POST", f"/chains/{chain_id}/execute", json=payload)

    # ==================== Reviews ====================

    async def reviews_list(
        self,
        status: Optional[str] = None,
        limit: int = 50,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List skill reviews."""
        params = {"limit": limit, "offset": offset}
        if status:
            params["status"] = status
        return await self.request("GET", "/reviews", params=params)

    async def reviews_get(self, review_id: str) -> Dict[str, Any]:
        """Get a review by ID."""
        return await self.request("GET", f"/reviews/{review_id}")

    async def reviews_process(
        self,
        review_id: str,
        approved: bool,
        notes: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Process a skill review."""
        payload = {"approved": approved}
        if notes:
            payload["notes"] = notes
        return await self.request("POST", f"/reviews/{review_id}", json=payload)

    # ==================== Agents ====================

    async def agents_create(
        self,
        name: str,
        description: Optional[str] = None,
        config: Optional[Dict[str, Any]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Create a new agent."""
        payload = {"name": name}
        if description:
            payload["description"] = description
        if config:
            payload["config"] = config
        if metadata:
            payload["metadata"] = metadata
        return await self.request("POST", "/agents", json=payload)

    async def agents_get(self, agent_id: str) -> Dict[str, Any]:
        """Get an agent by ID."""
        return await self.request("GET", f"/agents/{agent_id}")

    async def agents_list(
        self,
        status: Optional[str] = None,
        limit: int = 50,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List agents."""
        params = {"limit": limit, "offset": offset}
        if status:
            params["status"] = status
        return await self.request("GET", "/agents", params=params)

    async def agents_update(
        self,
        agent_id: str,
        name: Optional[str] = None,
        description: Optional[str] = None,
        config: Optional[Dict[str, Any]] = None,
        status: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Update an agent."""
        payload = {}
        if name:
            payload["name"] = name
        if description:
            payload["description"] = description
        if config:
            payload["config"] = config
        if status:
            payload["status"] = status
        return await self.request("PUT", f"/agents/{agent_id}", json=payload)

    async def agents_delete(self, agent_id: str) -> Dict[str, str]:
        """Delete an agent."""
        return await self.request("DELETE", f"/agents/{agent_id}")

    # ==================== Groups ====================

    async def groups_create(
        self,
        name: str,
        description: Optional[str] = None,
        domain: Optional[str] = None,
        policy: Optional[Dict[str, Any]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Create a new agent group."""
        payload = {"name": name}
        if description:
            payload["description"] = description
        if domain:
            payload["domain"] = domain
        if policy:
            payload["policy"] = policy
        if metadata:
            payload["metadata"] = metadata
        return await self.request("POST", "/groups", json=payload)

    async def groups_get(self, group_id: str) -> Dict[str, Any]:
        """Get a group by ID."""
        return await self.request("GET", f"/groups/{group_id}")

    async def groups_list(
        self,
        domain: Optional[str] = None,
        limit: int = 50,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List agent groups."""
        params = {"limit": limit, "offset": offset}
        if domain:
            params["domain"] = domain
        return await self.request("GET", "/groups", params=params)

    async def groups_update(
        self,
        group_id: str,
        name: Optional[str] = None,
        description: Optional[str] = None,
        policy: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Update a group."""
        payload = {}
        if name:
            payload["name"] = name
        if description:
            payload["description"] = description
        if policy:
            payload["policy"] = policy
        return await self.request("PUT", f"/groups/{group_id}", json=payload)

    async def groups_delete(self, group_id: str) -> Dict[str, str]:
        """Delete a group."""
        return await self.request("DELETE", f"/groups/{group_id}")

    async def groups_add_member(
        self,
        group_id: str,
        agent_id: str,
        role: str = "contributor",
    ) -> Dict[str, bool]:
        """Add an agent to a group."""
        return await self.request(
            "POST",
            f"/groups/{group_id}/members",
            json={"agent_id": agent_id, "role": role},
        )

    async def groups_remove_member(
        self,
        group_id: str,
        agent_id: str,
    ) -> Dict[str, bool]:
        """Remove an agent from a group."""
        return await self.request("DELETE", f"/groups/{group_id}/members/{agent_id}")

    async def groups_get_skills(
        self,
        group_id: str,
        limit: int = 50,
    ) -> Dict[str, Any]:
        """Get skills in a group."""
        return await self.request(
            "GET", f"/groups/{group_id}/skills", params={"limit": limit}
        )

    async def groups_get_memories(
        self,
        group_id: str,
    ) -> Dict[str, Any]:
        """Get shared memories in a group."""
        return await self.request("GET", f"/groups/{group_id}/memories")

    async def groups_share_memory(
        self,
        group_id: str,
        memory_id: str,
    ) -> Dict[str, bool]:
        """Share a memory to a group."""
        return await self.request(
            "POST", f"/groups/{group_id}/memories", json={"memory_id": memory_id}
        )

    # ==================== Notifications ====================

    async def notifications_list(
        self,
        read: Optional[bool] = None,
        limit: int = 50,
        offset: int = 0,
    ) -> Dict[str, Any]:
        """List notifications."""
        params = {"limit": limit, "offset": offset}
        if read is not None:
            params["read"] = read
        return await self.request("GET", "/notifications", params=params)

    async def notifications_mark_read(
        self,
        notification_id: str,
    ) -> Dict[str, str]:
        """Mark a notification as read."""
        return await self.request("POST", f"/notifications/{notification_id}/read")

    async def notifications_mark_all_read(self) -> Dict[str, str]:
        """Mark all notifications as read."""
        return await self.request("POST", "/notifications/read-all")

    # ==================== Admin ====================

    async def admin_sync(
        self,
        entity_ids: Optional[List[str]] = None,
    ) -> Dict[str, str]:
        """Sync entities to vector store."""
        payload = {}
        if entity_ids:
            payload["entity_ids"] = entity_ids
        return await self.request("POST", "/admin/sync", json=payload)

    async def admin_analytics(self) -> Dict[str, Any]:
        """Get admin analytics."""
        return await self.request("GET", "/analytics/dashboard")

    async def admin_list_api_keys(self) -> List[Dict[str, Any]]:
        """List all API keys."""
        return await self.request("GET", "/admin/api-keys")

    async def admin_create_api_key(
        self,
        label: str,
        expires_in_hours: int = 0,
        tenant_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Create a new API key."""
        payload = {"label": label, "expires_in_hours": expires_in_hours}
        if tenant_id:
            payload["tenant_id"] = tenant_id
        return await self.request("POST", "/admin/api-keys", json=payload)

    async def admin_delete_api_key(self, key_id: str) -> Dict[str, str]:
        """Delete an API key."""
        return await self.request("DELETE", f"/admin/api-keys/{key_id}")

    # ==================== Users ====================

    async def users_invite(
        self,
        email: str,
        role: str = "member",
    ) -> Dict[str, Any]:
        """Invite a new user."""
        return await self.request(
            "POST", "/admin/invites", json={"email": email, "role": role}
        )

    async def users_list_invites(
        self,
        status: Optional[str] = None,
    ) -> Dict[str, Any]:
        """List pending invitations."""
        params = {}
        if status:
            params["status"] = status
        return await self.request("GET", "/admin/invites", params=params)

    async def users_cancel_invite(self, invite_id: str) -> Dict[str, str]:
        """Cancel an invitation."""
        return await self.request("DELETE", f"/admin/invites/{invite_id}")

    async def users_accept_invite(
        self,
        invite_id: str,
        name: str,
        password: str,
    ) -> Dict[str, Any]:
        """Accept an invitation."""
        return await self.request(
            "POST",
            f"/admin/invites/{invite_id}/accept",
            json={"name": name, "password": password},
        )

    # ==================== Compression Engine (PROPRIETARY) ====================

    async def compression_get_stats(self) -> Dict[str, Any]:
        """Get compression statistics."""
        return await self.request("GET", "/compression/stats")

    async def compression_get_mode(self) -> str:
        """Get current compression mode."""
        return (await self.request("GET", "/compression/mode")).get(
            "mode", CompressionMode.EXTRACT
        )

    async def compression_set_mode(self, mode: str) -> Dict[str, bool]:
        """Set the compression mode."""
        if mode not in [
            CompressionMode.EXTRACT,
            CompressionMode.BALANCED,
            CompressionMode.AGGRESSIVE,
        ]:
            raise ValidationError(f"Invalid compression mode: {mode}")
        return await self.request("PUT", "/compression/mode", json={"mode": mode})

    async def tier_get_policy(self) -> str:
        """Get current tier policy."""
        return (await self.request("GET", "/tier/policy")).get(
            "policy", TierPolicy.BALANCED
        )

    async def tier_set_policy(self, policy: str) -> Dict[str, bool]:
        """Set the memory tier policy."""
        if policy not in [
            TierPolicy.AGGRESSIVE,
            TierPolicy.BALANCED,
            TierPolicy.CONSERVATIVE,
        ]:
            raise ValidationError(f"Invalid tier policy: {policy}")
        return await self.request("PUT", "/tier/policy", json={"policy": policy})

    async def search_enhanced(
        self,
        query: str,
        mode: str = SearchMode.SPREADING,
        limit: int = 10,
    ) -> Dict[str, Any]:
        """Enhanced search with proprietary spreading activation."""
        if mode not in [SearchMode.VECTOR, SearchMode.SPREADING, SearchMode.HYBRID]:
            mode = SearchMode.SPREADING
        return await self.request(
            "GET",
            "/search/enhanced",
            params={"query": query, "mode": mode, "limit": limit},
        )

    async def temporal_search(
        self,
        query: str,
        time_start: Optional[str] = None,
        time_end: Optional[str] = None,
        limit: int = 10,
    ) -> List[Dict[str, Any]]:
        """Search memories within a time range."""
        params: Dict[str, Any] = {"q": query, "limit": limit}
        if time_start:
            params["time_start"] = time_start
        if time_end:
            params["time_end"] = time_end
        return await self.request("GET", "/search", params=params)

    async def get_provenance_chain(self, memory_id: str) -> List[Dict[str, Any]]:
        """Get the provenance/version chain for a memory."""
        return await self.request("GET", f"/memories/{memory_id}/versions")

    # Convenience aliases matching the task spec
    async def get_compression_stats(self) -> Dict[str, Any]:
        """Get compression statistics (alias for compression_get_stats)."""
        return await self.compression_get_stats()

    async def set_compression_mode(self, mode: str) -> Dict[str, bool]:
        """Set compression mode (alias for compression_set_mode)."""
        return await self.compression_set_mode(mode)

    async def get_tier_policy(self) -> str:
        """Get tier policy (alias for tier_get_policy)."""
        return await self.tier_get_policy()

    async def set_tier_policy(self, policy: str) -> Dict[str, bool]:
        """Set tier policy (alias for tier_set_policy)."""
        return await self.tier_set_policy(policy)


# ==================== Sync Wrapper ====================


class Hystersis:
    """
    Synchronous wrapper for AsyncHystersis.

    Provides the same API but with blocking calls.
    Maintains backward compatibility with the original SDK.
    """

    def __init__(
        self,
        base_url: str = "https://api.hystersis.ai",
        api_key: Optional[str] = None,
        timeout: Optional[TimeoutConfig] = None,
        retry: Optional[RetryConfig] = None,
        rate_limit: Optional[RateLimitConfig] = None,
        max_connections: int = 100,
        max_keepalive_connections: int = 20,
    ):
        self._async_client = AsyncHystersis(
            base_url=base_url,
            api_key=api_key,
            timeout=timeout,
            retry=retry,
            rate_limit=rate_limit,
            max_connections=max_connections,
            max_keepalive_connections=max_keepalive_connections,
        )
        self._loop: Optional[asyncio.AbstractEventLoop] = None

    def _ensure_loop(self):
        if self._loop is None:
            self._loop = asyncio.new_event_loop()
        if self._loop.is_closed():
            self._loop = asyncio.new_event_loop()

    def _run_async(self, coro):
        self._ensure_loop()
        return self._loop.run_until_complete(coro)

    def close(self):
        if self._loop and not self._loop.is_closed():
            try:
                self._loop.run_until_complete(self._async_client.close())
            finally:
                self._loop.close()
                self._loop = None

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

    # Map old method names to new async methods for backward compatibility
    def __getattr__(self, name: str):
        # Backward compatibility mappings
        compat_map = {
            # Sessions (old -> new)
            "create_session": "create_session",
            "get_session": "get_session",
            "delete_session": "delete_session",
            "list_sessions": "list_sessions",
            "add_message": "add_message",
            "get_messages": "get_messages",
            "get_context": "get_context",
            # Memories (old -> new)
            "create_memory": "create_memory",
            "get_memory": "memories_get",
            "update_memory": "memories_update",
            "delete_memory": "memories_delete",
            "list_memories": "memories_list",
            "search": "memories_search",
            "semantic_search": "memories_search",
            "add_feedback": "feedback_add",
            "get_memory_history": "memories_history",
            "set_memory_expiration": "memories_set_expiration",
            "link_memory_to_entity": "memories_link_to_entity",
            "batch_create_memories": "memories_batch_create",
            "batch_update_memories": "memories_batch_update",
            "bulk_delete": "memories_bulk_delete",
            # Entities (old -> new)
            "create_entity": "entities_create",
            "get_entity": "entities_get",
            "list_entities": "entities_list",
            "update_entity": "entities_update",
            "delete_entity": "entities_delete",
            "get_entity_memories": "entities_get_memories",
            "get_entity_relations": "entities_get_relations",
            # Relations (old -> new)
            "create_relation": "relations_create",
            "delete_relation": "relations_delete",
            # Graph (old -> new)
            "graph_query": "graph_query",
            "traverse": "graph_traverse",
            # Projects (old -> new)
            "create_project": "projects_create",
            "get_project": "projects_get",
            "list_projects": "projects_list",
            "update_project": "projects_update",
            "delete_project": "projects_delete",
            # Webhooks (old -> new)
            "create_webhook": "webhooks_create",
            "get_webhook": "webhooks_get",
            "list_webhooks": "webhooks_list",
            "update_webhook": "webhooks_update",
            "delete_webhook": "webhooks_delete",
            "test_webhook": "webhooks_test",
            # Skills (old -> new)
            "create_skill": "skills_create",
            "get_skill": "skills_get",
            "list_skills": "skills_list",
            "search_skills": "skills_search",
            "update_skill": "skills_update",
            "delete_skill": "skills_delete",
            "use_skill": "skills_use",
            "suggest_skills": "skills_suggest",
            "synthesize_skills": "skills_synthesize",
            "extract_skills": "skills_extract",
            # Skill Chains (old -> new)
            "create_chain": "chains_create",
            "get_chain": "chains_get",
            "list_chains": "chains_list",
            "update_chain": "chains_update",
            "delete_chain": "chains_delete",
            "execute_chain": "chains_execute",
            # Agents (old -> new)
            "create_agent": "agents_create",
            "get_agent": "agents_get",
            "list_agents": "agents_list",
            "update_agent": "agents_update",
            "delete_agent": "agents_delete",
            # Groups (old -> new)
            "create_group": "groups_create",
            "get_group": "groups_get",
            "list_groups": "groups_list",
            "update_group": "groups_update",
            "delete_group": "groups_delete",
            "add_member": "groups_add_member",
            "remove_member": "groups_remove_member",
            "get_group_skills": "groups_get_skills",
            "get_group_memories": "groups_get_memories",
            "share_memory_to_group": "groups_share_memory",
            # Reviews (old -> new)
            "list_reviews": "reviews_list",
            "get_review": "reviews_get",
            "process_review": "reviews_process",
            # Notifications (old -> new)
            "list_notifications": "notifications_list",
            "mark_notification_read": "notifications_mark_read",
            "mark_all_notifications_read": "notifications_mark_all_read",
            # Admin (old -> new)
            "sync_entities": "admin_sync",
            "admin_analytics": "admin_analytics",
            "list_api_keys": "admin_list_api_keys",
            "create_api_key": "admin_create_api_key",
            "delete_api_key": "admin_delete_api_key",
            # Users (old -> new)
            "invite_user": "users_invite",
            "list_invitations": "users_list_invites",
            "cancel_invitation": "users_cancel_invite",
            "accept_invitation": "users_accept_invite",
            # Compression Engine (old -> new)
            "set_compression_mode": "set_compression_mode",
            "get_compression_stats": "get_compression_stats",
            "get_compression_mode": "compression_get_mode",
            "set_tier_policy": "set_tier_policy",
            "get_tier_policy": "get_tier_policy",
            "search_enhanced": "search_enhanced",
            # New feature methods
            "temporal_search": "temporal_search",
            "get_provenance_chain": "get_provenance_chain",
            # Misc (old -> new)
            "infer_memory": "create_memory",
            "process_memory": "create_memory",
            "health": "health",
            "ready": "ready",
        }

        async_name = compat_map.get(name, name)

        if not hasattr(self._async_client, async_name):
            raise AttributeError(
                f"'{type(self).__name__}' object has no attribute '{name}'"
            )

        async_method = getattr(self._async_client, async_name)

        def wrapper(*args, **kwargs):
            return self._run_async(async_method(*args, **kwargs))

        return wrapper


# Alias for backwards compatibility
AgentMemoryError = HystersisError
