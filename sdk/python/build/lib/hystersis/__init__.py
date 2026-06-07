"""
Hystersis Python SDK

Persistent memory infrastructure for AI agents.
Give your agents memory that adapts and compounds over time.

Example (Sync):
    from hystersis import Hystersis

    client = Hystersis("https://api.hystersis.com", api_key="your-key")
    session = client.create_session(agent_id="assistant-bot")
    client.add_message(session["id"], "user", "Hello!")
    memory = client.create_memory(content="User likes ML", user_id="user-123")
    results = client.search("deep learning")
    client.add_feedback(memory["id"], "positive")
    client.close()

Example (Async):
    import asyncio
    from hystersis import AsyncHystersis

    async def main():
        async with AsyncHystersis("https://api.hystersis.com", api_key="your-key") as client:
            session = await client.create_session(agent_id="assistant-bot")
            memory = await client.create_memory(content="User likes ML", user_id="user-123")
            results = await client.search("deep learning")
"""

from hystersis._async import (
    # Main client classes
    AsyncHystersis,
    Hystersis,
    
    # Exceptions
    HystersisError,
    AuthenticationError,
    NotFoundError,
    ValidationError,
    RateLimitError,
    ServerError,
    
    # Alias for backwards compatibility
    AgentMemoryError,
    
    # Enums
    MemoryType,
    FeedbackType,
    ImportanceLevel,
    MemoryLinkType,
    MemberRole,
    ReviewStatus,
    CompressionMode,
    TierPolicy,
    SearchMode,
    ChainStatus,
    
    # Config classes
    RetryConfig,
    RateLimitConfig,
    TimeoutConfig,
    HystersisConfig,
)

# For backwards compatibility, allow importing from top level
__all__ = [
    "AsyncHystersis",
    "Hystersis",
    "HystersisError",
    "AuthenticationError",
    "NotFoundError",
    "ValidationError",
    "RateLimitError",
    "ServerError",
    "AgentMemoryError",
    "MemoryType",
    "FeedbackType",
    "ImportanceLevel",
    "MemoryLinkType",
    "MemberRole",
    "ReviewStatus",
    "CompressionMode",
    "TierPolicy",
    "SearchMode",
    "ChainStatus",
    "RetryConfig",
    "RateLimitConfig",
    "TimeoutConfig",
    "HystersisConfig",
]

# Default export for backwards compatibility
__version__ = "0.1.0"
default_client = Hystersis