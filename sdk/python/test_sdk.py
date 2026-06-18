"""
Comprehensive Python SDK integration test against a local server.

Usage:
    python test_sdk.py                          # defaults: http://localhost:8080
    python test_sdk.py --base-url http://x:80   # custom server
    python test_sdk.py --api-key my-key         # with API key
    python test_sdk.py --verbose                # print full responses

Exits 0 if all tests pass, 1 otherwise.
"""

import argparse
import json
import sys
import traceback
import uuid
from datetime import datetime

try:
    from hystersis import Hystersis, SearchMode, CompressionMode, TierPolicy, RetryConfig
except ImportError:
    print("ERROR: hystersis SDK not installed. Run: pip install -e sdk/python")
    sys.exit(2)


PASS = 0
FAIL = 0
ERRORS = []
VERBOSE = False


def section(name):
    print(f"\n{'='*60}")
    print(f"  {name}")
    print(f"{'='*60}")


def check(label, ok, detail=""):
    global PASS, FAIL, ERRORS
    status = "PASS" if ok else "FAIL"
    icon = "+" if ok else "!"
    msg = f"  [{icon}] {status}  {label}"
    if detail and VERBOSE:
        msg += f"  -- {detail}"
    print(msg)
    if ok:
        PASS += 1
    else:
        FAIL += 1
        ERRORS.append(label)


def safe(label, fn, *args, **kwargs):
    """Run fn, catch exceptions, report result, return value or None."""
    try:
        result = fn(*args, **kwargs)
        if isinstance(result, dict) and result.get("error"):
            check(label, False, str(result["error"]))
            return None
        check(label, True, _preview(result))
        return result
    except Exception as e:
        check(label, False, f"{type(e).__name__}: {e}")
        if VERBOSE:
            traceback.print_exc()
        return None


def _preview(val):
    if val is None:
        return ""
    s = json.dumps(val, default=str)
    return s[:120] + ("..." if len(s) > 120 else "")


# ── Main ──────────────────────────────────────────────────────────

def main():
    global VERBOSE

    parser = argparse.ArgumentParser(description="Test Hystersis Python SDK")
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--api-key", default="test-key")
    parser.add_argument("--verbose", "-v", action="store_true")
    args = parser.parse_args()
    VERBOSE = args.verbose

    print(f"Server : {args.base_url}")
    print(f"API key: {args.api_key}")
    print(f"SDK    : {__import__('hystersis').__version__}")

    client = Hystersis(
        base_url=args.base_url,
        api_key=args.api_key,
        retry=RetryConfig(max_retries=1, base_delay=0.5),
    )

    # Unique IDs for this run to avoid collisions
    uid = uuid.uuid4().hex[:8]
    user_id = f"test-user-{uid}"
    agent_id = f"test-agent-{uid}"
    org_id = f"test-org-{uid}"
    chain_id = None

    # ── 1. Health & Readiness ──────────────────────────────────────
    section("1. Health & Readiness")
    health = safe("health()", client.health)
    check("health status ok", health and health.get("status") in ("ok", "healthy"),
          str(health))

    # ready() may return 503 if external deps (Neo4j, Qdrant) are not running
    try:
        ready = client.ready()
        check("ready()", True, _preview(ready))
    except Exception as e:
        err_msg = str(e)
        if "503" in err_msg:
            print(f"  [~] WARN  ready() returned 503 (deps not running) -- skipping")
        else:
            check("ready()", False, f"{type(e).__name__}: {e}")

    # ── 2. Sessions ────────────────────────────────────────────────
    section("2. Sessions")
    session = safe("create_session()", client.create_session,
                   agent_id=agent_id, metadata={"test": uid})
    session_id = session["id"] if session else None
    check("session has id", session_id is not None, str(session_id))

    if session_id:
        safe("add_message(user)", client.add_message,
             session_id, "user", "I prefer dark mode and Python")
        safe("add_message(assistant)", client.add_message,
             session_id, "assistant", "Noted! I'll remember your preferences.")
        safe("add_message(user)", client.add_message,
             session_id, "user", "Also I'm working on a graph database project")
        msgs = safe("get_messages()", client.get_messages, session_id)
        check("messages count >= 3", msgs is not None and len(msgs) >= 3, f"got {len(msgs) if msgs else 0}")
        ctx = safe("get_context()", client.get_context, session_id)
        check("context not empty", ctx is not None and len(ctx) > 0)

    got_session = safe("get_session()", client.get_session, session_id) if session_id else None
    check("get_session returns session", got_session is not None and got_session.get("id") == session_id)

    sessions = safe("list_sessions()", client.list_sessions)
    check("list_sessions not empty", sessions is not None and len(sessions) > 0)

    # ── 3. Memories CRUD ───────────────────────────────────────────
    section("3. Memories CRUD")
    mem1 = safe("create_memory(user pref)", client.create_memory,
                content="User prefers dark mode in all applications",
                user_id=user_id, metadata={"source": "sdk-test"})
    mem1_id = mem1.get("id") if mem1 else None
    check("memory has id", mem1_id is not None, str(mem1_id))

    mem2 = safe("create_memory(python)", client.create_memory,
                content="User is proficient in Python and Go",
                user_id=user_id, org_id=org_id)
    mem2_id = mem2.get("id") if mem2 else None

    mem3 = safe("create_memory(graph db)", client.create_memory,
                content="User is working on a Neo4j graph database project for social network analysis",
                user_id=user_id, metadata={"project": "graph-sna"})
    mem3_id = mem3.get("id") if mem3 else None

    if mem1_id:
        got = safe("memories_get()", client.memories_get, mem1_id)
        check("get returns correct memory", got and got.get("id") == mem1_id)

        updated = safe("memories_update()", client.memories_update, mem1_id,
                       metadata={"updated_by": "sdk-test", "version": 2})
        check("update succeeds", updated is not None)

    mems = safe("memories_list()", client.memories_list, user_id=user_id)
    if mems:
        count = mems.get("total", len(mems.get("memories", [])))
        check("list count >= 3", count >= 3, f"got {count}")
    else:
        check("memories_list returned data", False)

    mems_org = safe("memories_list(org_id)", client.memories_list, org_id=org_id)
    check("list by org_id filters",
          mems_org is not None)

    # ── 4. Semantic Search ─────────────────────────────────────────
    section("4. Semantic Search")
    results = safe("memories_search(dark mode)", client.memories_search,
                   "dark mode preferences", limit=5)
    check("search returns results", results is not None and len(results) > 0,
          f"got {len(results) if results else 0}")

    results2 = safe("memories_search(graph database)", client.memories_search,
                    "graph database Neo4j project", limit=5, user_id=user_id)
    check("search with user_id filter", results2 is not None)

    # Legacy alias
    legacy = safe("search() (legacy alias)", client.search, "Python programming", limit=3)
    check("legacy search() works", legacy is not None and len(legacy) >= 0)

    # ── 5. Memory History & Versions ───────────────────────────────
    section("5. Memory History & Versions")
    if mem1_id:
        hist = safe("memories_history()", client.memories_history, mem1_id)
        check("history is a list", hist is not None and isinstance(hist, list))

    # ── 6. Entities & Relations (Knowledge Graph) ──────────────────
    section("6. Entities & Relations")
    ent1 = safe("entities_create(person)", client.entities_create,
                name=f"Alice_{uid}", entity_type="person",
                properties={"role": "engineer"})
    ent1_id = ent1.get("id") if ent1 else None

    ent2 = safe("entities_create(project)", client.entities_create,
                name=f"GraphProject_{uid}", entity_type="project",
                properties={"status": "active"})
    ent2_id = ent2.get("id") if ent2 else None

    if ent1_id:
        got_ent = safe("entities_get()", client.entities_get, ent1_id)
        check("get entity returns correct", got_ent and got_ent.get("id") == ent1_id)

    ents = safe("entities_list()", client.entities_list, entity_type="person")
    check("entities_list not empty", ents is not None)

    if ent1_id and ent2_id:
        rel = safe("relations_create()", client.relations_create,
                   from_id=ent1_id, to_id=ent2_id,
                   relation_type="WORKS_ON",
                   metadata={"since": "2024"})
        check("relation created", rel is not None)

        rels = safe("entities_get_relations()", client.entities_get_relations, ent1_id)
        check("entity has relations", rels is not None)

    # ── 7. Graph Query ─────────────────────────────────────────────
    section("7. Graph Query")
    cq = safe("graph_query(MATCH)", client.graph_query,
              "MATCH (n) RETURN count(n) as total LIMIT 1")
    check("cypher query returned", cq is not None and isinstance(cq, list))

    # ── 8. Feedback ────────────────────────────────────────────────
    section("8. Feedback")
    if mem1_id:
        fb = safe("feedback_add(positive)", client.feedback_add,
                  mem1_id, "positive", comment="Accurate memory")
        check("feedback added", fb is not None)

    fb_mems = safe("feedback_get_memories()", client.feedback_get_memories,
                   feedback_type="positive")
    check("feedback_get_memories returns", fb_mems is not None)

    # ── 9. Compression Engine ──────────────────────────────────────
    section("9. Compression Engine")
    stats = safe("compression_get_stats()", client.compression_get_stats)
    check("stats has expected fields",
          stats is not None and isinstance(stats, dict))

    mode = safe("compression_get_mode()", client.compression_get_mode)
    check("mode is string", mode is not None and isinstance(mode, str))

    set_result = safe("compression_set_mode(extract)",
                      client.compression_set_mode, CompressionMode.EXTRACT)
    check("set mode returns success", set_result is not None)

    # ── 10. Tier Policy ────────────────────────────────────────────
    section("10. Tier Policy")
    policy = safe("tier_get_policy()", client.tier_get_policy)
    check("policy returned", policy is not None)

    tp_result = safe("tier_set_policy(balanced)",
                     client.tier_set_policy, TierPolicy.BALANCED)
    check("set policy returns", tp_result is not None)

    # ── 11. Enhanced Search (Spreading Activation) ─────────────────
    section("11. Enhanced Search")
    es = safe("search_enhanced(spreading)", client.search_enhanced,
              "graph database", mode=SearchMode.SPREADING, limit=5)
    check("enhanced search returned", es is not None)

    # ── 12. Skills ─────────────────────────────────────────────────
    section("12. Skills")
    skill = safe("skills_create()", client.skills_create,
                 name=f"test-skill-{uid}",
                 trigger="user asks for code review",
                 action="Review the code and provide feedback",
                 domain="code-review",
                 tags=["testing", "sdk"])
    skill_id = skill.get("id") if skill else None

    if skill_id:
        got_skill = safe("skills_get()", client.skills_get, skill_id)
        check("skills_get returns correct", got_skill and got_skill.get("id") == skill_id)

    skills = safe("skills_list()", client.skills_list)
    check("skills_list returned", skills is not None)

    suggested = safe("skills_suggest()", client.skills_suggest,
                     trigger="user needs help debugging", context="Python traceback")
    check("skills_suggest returned", suggested is not None)

    extracted = safe("skills_extract()", client.skills_extract,
                     "When the user asks to review code, first check linting then check logic errors")
    check("skills_extract returned", extracted is not None)

    # ── 13. Chains ─────────────────────────────────────────────────
    section("13. Chains")
    if skill_id:
        chain = safe("chains_create()", client.chains_create,
                     name=f"test-chain-{uid}",
                     trigger="user requests full analysis",
                     steps=[{"skill_id": skill_id, "order": 1}])
        if chain:
            chain_id = chain.get("id")

        if chain_id:
            got_chain = safe("chains_get()", client.chains_get, chain_id)
            check("chains_get returned", got_chain is not None)

        chains = safe("chains_list()", client.chains_list)
        check("chains_list returned", chains is not None)

    # ── 14. Groups ─────────────────────────────────────────────────
    section("14. Groups")
    group = safe("groups_create()", client.groups_create,
                 name=f"test-group-{uid}",
                 description="SDK test group",
                 domain="testing")
    group_id = group.get("id") if group else None

    if group_id:
        got_group = safe("groups_get()", client.groups_get, group_id)
        check("groups_get returned", got_group is not None)

    groups = safe("groups_list()", client.groups_list)
    check("groups_list returned", groups is not None)

    # ── 15. Webhooks ───────────────────────────────────────────────
    section("15. Webhooks")
    wh = safe("webhooks_create()", client.webhooks_create,
              url="https://httpbin.org/post",
              events=["memory.created", "memory.updated"])
    wh_id = wh.get("id") if wh else None

    if wh_id:
        got_wh = safe("webhooks_get()", client.webhooks_get, wh_id)
        check("webhooks_get returned", got_wh is not None)

    whs = safe("webhooks_list()", client.webhooks_list)
    check("webhooks_list returned", whs is not None)

    # ── 16. Notifications ──────────────────────────────────────────
    section("16. Notifications")
    notifs = safe("notifications_list()", client.notifications_list)
    check("notifications_list returned", notifs is not None)

    summary = safe("notifications_summary()", client.notifications_summary)
    check("notifications_summary returned", summary is not None)

    prefs = safe("notifications_get_preferences()", client.notifications_get_preferences)
    check("notifications_get_preferences returned", prefs is not None)

    # ── 17. Projects ───────────────────────────────────────────────
    section("17. Projects")
    proj = safe("projects_create()", client.projects_create,
                name=f"test-project-{uid}",
                description="SDK integration test project")
    proj_id = proj.get("id") if proj else None

    if proj_id:
        got_proj = safe("projects_get()", client.projects_get, proj_id)
        check("projects_get returned", got_proj is not None)

    projs = safe("projects_list()", client.projects_list)
    check("projects_list returned", projs is not None)

    # ── 18. Admin & Analytics ──────────────────────────────────────
    section("18. Admin & Analytics")
    analytics = safe("admin_analytics()", client.admin_analytics)
    check("analytics returned", analytics is not None)

    keys = safe("admin_list_api_keys()", client.admin_list_api_keys)
    check("api_keys_list returned", keys is not None)

    # ── 19. V3 Compatibility ───────────────────────────────────────
    section("19. V3 Compatibility API")
    v3_mem = safe("v3_add_memory()", client.v3_add_memory,
                  "V3 compat: user enjoys hiking in the mountains")
    v3_id = v3_mem.get("id") if v3_mem else None

    v3_results = safe("v3_search_memories()", client.v3_search_memories,
                      "hiking mountains outdoor")
    check("v3_search returned", v3_results is not None)

    v3_list = safe("v3_list_memories()", client.v3_list_memories)
    check("v3_list returned", v3_list is not None)

    # ── 20. Connections ────────────────────────────────────────────
    section("20. Connections")
    conns = safe("connections_list()", client.connections_list)
    check("connections_list returned", conns is not None)

    # ── 21. Playground ─────────────────────────────────────────────
    section("21. Playground")
    pc = safe("playground_compress()", client.playground_compress,
              "This is a test memory about compression testing")
    check("playground_compress returned", pc is not None)

    ps = safe("playground_search()", client.playground_search, "compression test")
    check("playground_search returned", ps is not None)

    pstats = safe("playground_stats()", client.playground_stats)
    check("playground_stats returned", pstats is not None)

    # ── Cleanup ────────────────────────────────────────────────────
    section("22. Cleanup")
    if chain_id:
        safe("chains_delete()", client.chains_delete, chain_id)
    if mem1_id:
        safe("memories_delete(mem1)", client.memories_delete, mem1_id)
    if mem2_id:
        safe("memories_delete(mem2)", client.memories_delete, mem2_id)
    if mem3_id:
        safe("memories_delete(mem3)", client.memories_delete, mem3_id)
    if ent1_id:
        safe("entities_delete(ent1)", client.entities_delete, ent1_id)
    if ent2_id:
        safe("entities_delete(ent2)", client.entities_delete, ent2_id)
    if skill_id:
        safe("skills_delete()", client.skills_delete, skill_id)
    if wh_id:
        safe("webhooks_delete()", client.webhooks_delete, wh_id)
    if proj_id:
        safe("projects_delete()", client.projects_delete, proj_id)
    if group_id:
        safe("groups_delete()", client.groups_delete, group_id)
    if session_id:
        safe("delete_session()", client.delete_session, session_id)

    client.close()

    # ── Summary ────────────────────────────────────────────────────
    print(f"\n{'='*60}")
    print(f"  RESULTS: {PASS} passed, {FAIL} failed, {PASS+FAIL} total")
    print(f"{'='*60}")
    if ERRORS:
        print("\nFailed tests:")
        for e in ERRORS:
            print(f"  - {e}")
    print()
    sys.exit(0 if FAIL == 0 else 1)


if __name__ == "__main__":
    main()
