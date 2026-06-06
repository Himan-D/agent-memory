"""Tests for Hystersis Python SDK."""

import pytest
from unittest.mock import AsyncMock, patch, MagicMock
from hystersis import Hystersis, AsyncHystersis


@pytest.fixture
def client():
    """Create a sync test client."""
    return Hystersis(api_key="test-key", base_url="http://localhost:8080")


@pytest.fixture
def async_client():
    """Create an async test client."""
    return AsyncHystersis(api_key="test-key", base_url="http://localhost:8080")


class TestHealth:
    def test_health(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"status": "ok"}
            result = client.health()
            assert result == {"status": "ok"}

    @pytest.mark.asyncio
    async def test_health_async(self, async_client):
        with patch.object(
            async_client, "_send_request", new_callable=AsyncMock
        ) as mock_send:
            mock_response = MagicMock()
            mock_response.status_code = 200
            mock_response.json.return_value = {"status": "ok"}
            mock_send.return_value = mock_response
            result = await async_client.health()
            assert result == {"status": "ok"}


class TestSessions:
    def test_create_session(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"id": "sess-123", "agent_id": "test-agent"}
            result = client.create_session("test-agent")
            assert result["id"] == "sess-123"
            assert result["agent_id"] == "test-agent"

    def test_get_messages(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = [
                {"id": "msg-1", "role": "user", "content": "Hello"}
            ]
            result = client.get_messages("sess-123")
            assert len(result) == 1
            assert result[0]["role"] == "user"

    def test_add_message(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"status": "ok"}
            result = client.add_message("sess-123", "user", "Hello")
            assert result["status"] == "ok"


class TestEntities:
    def test_create_entity(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"id": "ent-123", "name": "ML", "type": "Concept"}
            result = client.entities_create("ML", "Concept")
            assert result["id"] == "ent-123"

    def test_get_entity(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"id": "ent-123", "name": "ML", "type": "Concept"}
            result = client.entities_get("ent-123")
            assert result["name"] == "ML"


class TestSearch:
    def test_search(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = [{"score": 0.95, "text": "relevant"}]
            result = client.memories_search("test query")
            assert len(result) == 1
            assert result[0]["score"] == 0.95


class TestAPIKeys:
    def test_list_api_keys(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = [{"id": "key-1", "label": "prod"}]
            result = client.admin_list_api_keys()
            assert len(result) == 1

    def test_create_api_key(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"id": "key-1", "key": "am_xxx", "label": "prod"}
            result = client.admin_create_api_key("prod")
            assert "key" in result


class TestCompression:
    def test_get_stats(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {
                "accuracy_retention": 0.973,
                "token_reduction": 0.84,
            }
            result = client.compression_get_stats()
            assert result["accuracy_retention"] == 0.973

    def test_set_mode(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"success": True}
            result = client.compression_set_mode("extract")
            assert result["success"] is True


class TestTier:
    def test_get_policy(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"policy": "balanced"}
            result = client.tier_get_policy()
            assert result == "balanced"

    def test_set_policy(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"success": True}
            result = client.tier_set_policy("conservative")
            assert result["success"] is True


class TestEnhancedSearch:
    def test_search_enhanced(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {
                "results": [{"score": 0.9}],
                "mode": "spreading",
            }
            result = client.search_enhanced("complex query", mode="spreading")
            assert result["mode"] == "spreading"


class TestHybridSearch:
    def test_search_hybrid(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"results": [{"score": 0.8}], "count": 1}
            result = client.search_hybrid("test query", semantic_limit=5)
            assert result["count"] == 1
            mock_req.assert_called_once()
            call_args = mock_req.call_args
            assert call_args[0][0] == "POST"
            assert call_args[0][1] == "/search/hybrid"


class TestV3Compat:
    def test_v3_add(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"results": [{"id": "m1"}], "count": 1}
            result = client.v3_add(
                messages=[{"role": "user", "content": "hello"}],
                user_id="u1",
            )
            assert result["count"] == 1

    def test_v3_search(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"results": [], "count": 0}
            client.v3_search("hello", user_id="u1")
            mock_req.assert_called_with(
                "POST", "/v3/search", json={"query": "hello", "limit": 10, "user_id": "u1"}
            )


class TestBilling:
    def test_get_billing_plans(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"plans": [], "stripe_enabled": False}
            result = client.get_billing_plans()
            assert "plans" in result


class TestProfiles:
    def test_get_profile(self, client):
        with patch.object(
            client._async_client, "request", new_callable=AsyncMock
        ) as mock_req:
            mock_req.return_value = {"id": "user-1"}
            result = client.get_profile("user-1")
            assert result["id"] == "user-1"


class TestBackwardCompat:
    def test_old_names_work(self, client):
        assert callable(client.create_session)
        assert callable(client.create_memory)
        assert callable(client.search)
        assert callable(client.create_entity)
        assert callable(client.create_relation)
        assert callable(client.health)
        assert callable(client.search_hybrid)
        assert callable(client.v3_add)
        assert callable(client.get_billing_plans)

    def test_invalid_attribute_raises(self, client):
        with pytest.raises(AttributeError):
            client.nonexistent_method_xyz
