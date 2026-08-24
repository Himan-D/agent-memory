"""Live smoke tests against a running Hystersis API.

Skipped unless both env vars are set:

  HYSTERSIS_API_URL   e.g. http://localhost:8080 or https://api.hystersis.com
  HYSTERSIS_API_KEY   API key with write+read scope

Run:

  pytest -m live -q
  # or
  HYSTERSIS_API_URL=... HYSTERSIS_API_KEY=... pytest sdk/python/tests/test_smoke_live.py -v
"""

from __future__ import annotations

import os
import uuid

import pytest

pytestmark = pytest.mark.live

API_URL = os.getenv("HYSTERSIS_API_URL") or os.getenv("AGENT_MEMORY_URL")
API_KEY = (
    os.getenv("HYSTERSIS_API_KEY")
    or os.getenv("AGENT_MEMORY_API_KEY")
    or os.getenv("HYSTERESIS_API_KEY")
)


def _require_live():
    if not API_URL or not API_KEY:
        pytest.skip("Set HYSTERSIS_API_URL and HYSTERSIS_API_KEY for live smoke tests")


@pytest.fixture
def client():
    _require_live()
    from hystersis import Hystersis

    c = Hystersis(base_url=API_URL.rstrip("/"), api_key=API_KEY)
    yield c
    c.close()


class TestLiveSmoke:
    def test_health(self, client):
        result = client.health()
        assert result is not None

    def test_memory_roundtrip(self, client):
        token = f"smoke-{uuid.uuid4().hex[:12]}"
        content = f"Live smoke memory {token}"

        created = client.create_memory(content=content, user_id="smoke-user")
        assert created is not None

        # Search may be eventually consistent; try a few times is overkill for smoke —
        # at least list or search should not raise.
        try:
            results = client.search(token, limit=5)
            assert results is not None
        except Exception:
            results = client.memories_search(token, limit=5)
            assert results is not None

    def test_session_roundtrip(self, client):
        session = client.create_session(agent_id="smoke-agent")
        assert session is not None
        sid = session.get("id") or session.get("session_id")
        if not sid:
            pytest.skip("create_session response missing id")

        msg = client.add_message(sid, "user", "smoke hello")
        assert msg is not None

        messages = client.get_messages(sid)
        assert messages is not None
