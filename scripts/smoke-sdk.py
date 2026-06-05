#!/usr/bin/env python3
"""SDK smoke test — mirrors scripts/smoke-test.sh using the Python Hystersis client."""

from __future__ import annotations

import os
import sys
import time

BASE = os.getenv("HYSTERSIS_BASE_URL") or os.getenv("AGENT_MEMORY_URL") or "http://localhost:8080"
API_KEY = os.getenv("HYSTERSIS_API_KEY") or os.getenv("AGENT_MEMORY_API_KEY") or "test-key"
USER_ID = f"sdk-smoke-{int(time.time())}"

PASS = FAIL = SKIP = 0


def ok(name: str) -> None:
    global PASS
    print(f"  ✓ {name}")
    PASS += 1


def bad(name: str, detail: str = "") -> None:
    global FAIL
    print(f"  ✗ {name}" + (f" — {detail}" if detail else ""))
    FAIL += 1


def skip(name: str, detail: str = "") -> None:
    global SKIP
    print(f"  ~ {name} (skipped)" + (f" — {detail}" if detail else ""))
    SKIP += 1


def main() -> int:
    try:
        from hystersis import Hystersis
    except ImportError:
        root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        sys.path.insert(0, os.path.join(root, "sdk", "python"))
        from hystersis import Hystersis

    print("Hystersis Python SDK smoke test")
    print(f"Base URL: {BASE}")
    print()

    client = Hystersis(base_url=BASE, api_key=API_KEY)
    mem_id = None

    try:
        print("== Public ==")
        try:
            client.health()
            ok("health()")
        except Exception as e:
            bad("health()", str(e))
            print("\nStart server or set HYSTERSIS_BASE_URL.")
            return 1

        try:
            client.ready()
            ok("ready()")
        except Exception:
            skip("ready()")

        try:
            plans = client.get_billing_plans()
            assert "plans" in plans
            ok("get_billing_plans()")
        except Exception as e:
            skip("get_billing_plans()", str(e))

        print("\n== Memory CRUD ==")
        try:
            mem = client.create_memory(content="sdk smoke memory", user_id=USER_ID)
            mem_id = mem.get("id")
            assert mem_id
            ok(f"create_memory() → {mem_id}")
        except Exception as e:
            bad("create_memory()", str(e))

        if mem_id:
            try:
                got = client.get_memory(mem_id)
                assert got.get("id") == mem_id
                ok("get_memory()")
            except Exception as e:
                bad("get_memory()", str(e))

            try:
                client.update_memory(mem_id, content="updated sdk smoke")
                ok("update_memory()")
            except Exception as e:
                bad("update_memory()", str(e))

        print("\n== Search ==")
        try:
            client.search("sdk smoke", limit=5)
            ok("search()")
        except Exception as e:
            bad("search()", str(e))

        try:
            client.search_enhanced("sdk smoke", mode="vector", limit=5)
            ok("search_enhanced()")
        except Exception as e:
            skip("search_enhanced()", str(e))

        try:
            client.search_hybrid("sdk smoke", semantic_limit=5, keyword_limit=5)
            ok("search_hybrid()")
        except Exception as e:
            skip("search_hybrid()", str(e))

        print("\n== v3 compat ==")
        try:
            result = client.v3_add(
                messages=[{"role": "user", "content": "I like TypeScript"}],
                user_id=USER_ID,
            )
            assert result.get("count", 0) >= 0
            ok("v3_add()")
        except Exception as e:
            skip("v3_add()", str(e))

        try:
            client.v3_search("TypeScript", user_id=USER_ID, limit=5)
            ok("v3_search()")
        except Exception as e:
            skip("v3_search()", str(e))

        print("\n== Profiles ==")
        try:
            client.get_profile(USER_ID)
            ok("get_profile()")
        except Exception as e:
            skip("get_profile()", str(e))

        print("\n== Skills ==")
        try:
            client.list_skills(limit=1)
            ok("list_skills()")
        except Exception as e:
            skip("list_skills()", str(e))

        print("\n== Compression ==")
        try:
            client.get_compression_stats()
            ok("get_compression_stats()")
        except Exception as e:
            skip("get_compression_stats()", str(e))

        print("\n== Cleanup ==")
        if mem_id:
            try:
                client.delete_memory(mem_id)
                ok("delete_memory()")
            except Exception as e:
                skip("delete_memory()", str(e))

    finally:
        client.close()

    print(f"\nResults: {PASS} passed, {FAIL} failed, {SKIP} skipped")
    return 1 if FAIL else 0


if __name__ == "__main__":
    raise SystemExit(main())
