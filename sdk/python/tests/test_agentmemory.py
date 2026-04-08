"""Tests for Agent Memory Python SDK."""

import pytest
from unittest.mock import Mock, patch
from agentmemory import AgentMemory


@pytest.fixture
def client():
    """Create a test client."""
    return AgentMemory(api_key="test-key", base_url="http://localhost:8080")


class TestHealth:
    def test_health_success(self, client):
        with patch.object(client, "_request") as mock_request:
            mock_response = Mock()
            mock_response.json.return_value = {"status": "ok"}
            mock_response.raise_for_status = Mock()
            mock_request.return_value = mock_response

            result = client.health()
            assert result == {"status": "ok"}
            mock_request.assert_called_once_with("GET", "/health")


class TestSessions:
    def test_create_session(self, client):
        with patch.object(client, "_request") as mock_request:
            mock_response = Mock()
            mock_response.json.return_value = {
                "id": "sess-123",
                "agent_id": "test-agent",
            }
            mock_response.raise_for_status = Mock()
            mock_request.return_value = mock_response

            result = client.create_session("test-agent")
            assert result["id"] == "sess-123"
            assert result["agent_id"] == "test-agent"

    def test_get_messages(self, client):
        with patch.object(client, "_request") as mock_request:
            mock_response = Mock()
            mock_response.json.return_value = [
                {"id": "msg-1", "role": "user", "content": "Hello"}
            ]
            mock_response.raise_for_status = Mock()
            mock_request.return_value = mock_response

            result = client.get_messages("sess-123")
            assert len(result) == 1
            assert result[0]["role"] == "user"

    def test_add_message(self, client):
        with patch.object(client, "_request") as mock_request:
            mock_response = Mock()
            mock_response.json.return_value = {"status": "ok"}
            mock_response.raise_for_status = Mock()
            mock_request.return_value = mock_response

            result = client.add_message("sess-123", "user", "Hello")
            assert result["status"] == "ok"


class TestEntities:
    def test_create_entity(self, client):
        with patch.object(client, "_request") as mock_request:
            mock_response = Mock()
            mock_response.json.return_value = {"id": "ent-123", "status": "ok"}
            mock_response.raise_for_status = Mock()
            mock_request.return_value = mock_response

            result = client.create_entity("ML", "Concept")
            assert result["id"] == "ent-123"

    def test_get_entity(self, client):
        with patch.object(client, "_request") as mock_request:
            mock_response = Mock()
            mock_response.json.return_value = {
                "id": "ent-123",
                "name": "ML",
                "type": "Concept",
            }
            mock_response.raise_for_status = Mock()
            mock_request.return_value = mock_response

            result = client.get_entity("ent-123")
            assert result["name"] == "ML"


class TestSearch:
    def test_search(self, client):
        with patch.object(client, "_request") as mock_request:
            mock_response = Mock()
            mock_response.json.return_value = [{"score": 0.95, "text": "relevant"}]
            mock_response.raise_for_status = Mock()
            mock_request.return_value = mock_response

            result = client.search("test query")
            assert len(result) == 1
            assert result[0]["score"] == 0.95


class TestAPIKeys:
    def test_list_api_keys(self, client):
        with patch.object(client, "_request") as mock_request:
            mock_response = Mock()
            mock_response.json.return_value = [{"id": "key-1", "label": "prod"}]
            mock_response.raise_for_status = Mock()
            mock_request.return_value = mock_response

            result = client.list_api_keys()
            assert len(result) == 1

    def test_create_api_key(self, client):
        with patch.object(client, "_request") as mock_request:
            mock_response = Mock()
            mock_response.json.return_value = {
                "id": "key-1",
                "key": "am_xxx",
                "label": "prod",
            }
            mock_response.raise_for_status = Mock()
            mock_request.return_value = mock_response

            result = client.create_api_key("prod")
            assert "key" in result
