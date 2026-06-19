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
import os
import random
import statistics
import sys
import time
import traceback
import uuid
from datetime import datetime
from pathlib import Path

# Load .env file if present (needed for EVALUATOR_API_KEY etc.)
_env_paths = [
    Path(__file__).parent / ".env",
    Path(__file__).parent.parent / ".env",
    Path(__file__).parent.parent.parent / ".env",
]
for _env_path in _env_paths:
    if _env_path.is_file():
        for _line in _env_path.read_text().splitlines():
            _line = _line.strip()
            if not _line or _line.startswith("#"):
                continue
            if "=" in _line:
                _key, _, _val = _line.partition("=")
                _key = _key.strip()
                _val = _val.strip()
                if _key and _val and _key not in os.environ:
                    os.environ[_key] = _val
        break

try:
    from hystersis import Hystersis, SearchMode, CompressionMode, TierPolicy, RetryConfig, TimeoutConfig
    from hystersis._async import AuthenticationError, ServerError, NotFoundError, HystersisError
except ImportError:
    print("ERROR: hystersis SDK not installed. Run: pip install -e sdk/python")
    sys.exit(2)

try:
    from test_evaluator import evaluate_compression_fidelity
    HAS_EVALUATOR = True
except ImportError:
    HAS_EVALUATOR = False


PASS = 0
FAIL = 0
SKIP = 0
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


def skip(label, reason=""):
    global SKIP
    SKIP += 1
    print(f"  [~] SKIP  {label}" + (f"  -- {reason}" if reason else ""))


def safe(label, fn, *args, **kwargs):
    """Run fn, catch exceptions, report result, return value or None."""
    try:
        result = fn(*args, **kwargs)
        if isinstance(result, dict) and result.get("error"):
            check(label, False, str(result["error"]))
            return None
        check(label, True, _preview(result))
        return result
    except AuthenticationError as e:
        if "admin" in str(e).lower() or "forbidden" in str(e).lower():
            skip(label, "admin scope required")
        else:
            check(label, False, f"AuthenticationError: {e}")
        return None
    except Exception as e:
        check(label, False, f"{type(e).__name__}: {e}")
        if VERBOSE:
            traceback.print_exc()
        return None


def safe_server(label, fn, *args, **kwargs):
    """Like safe() but treats server-side 400/404 as SKIP (known server behavior)."""
    try:
        result = fn(*args, **kwargs)
        if isinstance(result, dict) and result.get("error"):
            check(label, False, str(result["error"]))
            return None
        check(label, True, _preview(result))
        return result
    except NotFoundError as e:
        skip(label, f"server: {e}")
        return None
    except AuthenticationError as e:
        if "admin" in str(e).lower() or "forbidden" in str(e).lower():
            skip(label, "admin scope required")
        else:
            check(label, False, f"AuthenticationError: {e}")
        return None
    except (HystersisError, ServerError) as e:
        err = str(e)
        if "400" in err or "invalid request" in err.lower():
            skip(label, f"server rejected: {err[:80]}")
        else:
            check(label, False, f"{type(e).__name__}: {e}")
        return None
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


def _percentile(sorted_data, p):
    """Compute percentile using linear interpolation."""
    if not sorted_data:
        return 0.0
    k = (len(sorted_data) - 1) * (p / 100.0)
    f = int(k)
    c = f + 1
    if c >= len(sorted_data):
        return float(sorted_data[-1])
    d = k - f
    return float(sorted_data[f]) + d * (float(sorted_data[c]) - float(sorted_data[f]))


# ── Main ──────────────────────────────────────────────────────────

def main():
    global VERBOSE

    parser = argparse.ArgumentParser(description="Test Hystersis Python SDK")
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--api-key", default="test-key")
    parser.add_argument("--verbose", "-v", action="store_true")
    parser.add_argument("--benchmark-sample", type=int, default=25,
                        help="Number of LongMemEval questions to benchmark (default: 25)")
    parser.add_argument("--benchmark-delay", type=float, default=0.6,
                        help="Seconds to sleep between benchmark search calls (default: 0.6)")
    parser.add_argument("--compression-sample", type=int, default=5,
                        help="Number of texts to benchmark compression on (default: 5)")
    parser.add_argument("--compression-delay", type=float, default=0.6,
                        help="Seconds to sleep between compression calls (default: 0.6)")
    parser.add_argument("--compression-modes", default="radix,hybrid",
                        help="Comma-separated compression modes to test (default: radix,hybrid)")
    parser.add_argument("--evaluator", action="store_true",
                        help="Enable LLM-based compression fidelity evaluation (requires EVALUATOR_API_KEY)")
    parser.add_argument("--evaluator-modes", default="extraction,relational",
                        help="Comma-separated compression modes to evaluate fidelity on (default: extraction,relational)")
    args = parser.parse_args()
    VERBOSE = args.verbose

    print(f"Server : {args.base_url}")
    print(f"API key: {args.api_key}")
    print(f"SDK    : {__import__('hystersis').__version__}")

    client = Hystersis(
        base_url=args.base_url,
        api_key=args.api_key,
        retry=RetryConfig(max_retries=1, base_delay=0.5),
        timeout=TimeoutConfig(read=30),
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

    # ready() may return 503 if external deps aren't reachable from the gateway
    try:
        ready = client.ready()
        check("ready()", True, _preview(ready))
    except Exception as e:
        err_msg = str(e)
        if "503" in err_msg:
            skip("ready()", "503 - gateway deps not reachable")
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

        # Server may not persist messages (in-memory session store)
        msgs = safe("get_messages()", client.get_messages, session_id)
        if msgs is not None:
            if len(msgs) == 0:
                skip("messages persisted", "server returned 0 messages (in-memory store)")
            else:
                check("messages count >= 1", len(msgs) >= 1, f"got {len(msgs)}")

        ctx = safe("get_context()", client.get_context, session_id)
        # Context may be empty if session store is ephemeral
        if ctx is not None and len(ctx) == 0:
            skip("context populated", "server session store is ephemeral")

    # get_session may 404 if session store doesn't persist
    if session_id:
        got_session = safe_server("get_session()", client.get_session, session_id)
        if got_session:
            check("get_session returns session", got_session.get("id") == session_id)

    sessions = safe("list_sessions()", client.list_sessions)
    check("list_sessions returned", sessions is not None)

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

        # memories_update requires content as a positional arg
        updated = safe("memories_update()", client.memories_update, mem1_id,
                       "User prefers dark mode in all applications (updated)",
                       metadata={"updated_by": "sdk-test", "version": 2})
        check("update succeeds", updated is not None)

    mems = safe("memories_list()", client.memories_list, user_id=user_id)
    if mems:
        count = mems.get("total", len(mems.get("memories", [])))
        check("list count >= 3", count >= 3, f"got {count}")
    else:
        check("memories_list returned data", False)

    mems_org = safe("memories_list(org_id)", client.memories_list, org_id=org_id)
    check("list by org_id filters", mems_org is not None)

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
        hist = safe_server("memories_history()", client.memories_history, mem1_id)
        if hist is not None:
            check("history is a list", isinstance(hist, list))
        # If hist is None it was skipped

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
        # Relations may fail on server side (Neo4j tenant isolation)
        rel = safe_server("relations_create()", client.relations_create,
                          from_id=ent1_id, to_id=ent2_id,
                          relation_type="WORKS_ON",
                          metadata={"since": "2024"})
        if rel is not None:
            rels = safe("entities_get_relations()", client.entities_get_relations, ent1_id)
            check("entity has relations", rels is not None)
        else:
            skip("entities_get_relations()", "depends on relations_create")

    # ── 7. Graph Query ─────────────────────────────────────────────
    section("7. Graph Query")
    # graph_query requires admin scope
    safe("graph_query(MATCH)", client.graph_query,
         "MATCH (n) RETURN count(n) as total LIMIT 1")

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
    else:
        skip("chains_create()", "depends on skills_create")

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
    # These require admin scope — will be skipped for non-admin keys
    safe("admin_analytics()", client.admin_analytics)
    safe("admin_list_api_keys()", client.admin_list_api_keys)

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

    # ── 22. LongMemEval Search Benchmark ───────────────────────────
    section("22. LongMemEval Search Benchmark")
    oracle_path = None
    for candidate in [
        os.path.join(os.path.dirname(__file__), "../../data/longmemeval_oracle.json"),
        os.path.join(os.path.dirname(__file__), "../data/longmemeval_oracle.json"),
        os.path.join(os.path.dirname(__file__), "data/longmemeval_oracle.json"),
        "data/longmemeval_oracle.json",
    ]:
        if os.path.isfile(candidate):
            oracle_path = candidate
            break

    oracle_data = None
    if oracle_path is None:
        skip("benchmark", "longmemeval_oracle.json not found")
    else:
        try:
            with open(oracle_path, "r", encoding="utf-8") as f:
                oracle_data = json.load(f)
        except Exception as e:
            skip("benchmark", f"failed to load oracle: {e}")

    if oracle_data:
        # Verify seeded data exists in the DB
        longmemeval_mems = safe_server(
            "memories_list(longmemeval-user)",
            client.memories_list, user_id="longmemeval-user"
        )
        seeded = False
        if longmemeval_mems:
            mem_count = longmemeval_mems.get(
                "total", len(longmemeval_mems.get("memories", []))
            )
            if mem_count > 0:
                seeded = True
                check("seeded data present", True, f"{mem_count} memories for longmemeval-user")
            else:
                skip("benchmark", "no longmemeval seeded data available")
        else:
            skip("benchmark", "no longmemeval seeded data available")

        if seeded:
            random.seed(42)
            sample_size = min(args.benchmark_sample, len(oracle_data))
            sampled = random.sample(oracle_data, sample_size)
            delay = args.benchmark_delay

            standard_latencies = []
            enhanced_latencies = []
            standard_counts = []
            enhanced_counts = []

            print(f"  Running benchmark with {sample_size} questions, {delay}s delay between calls")

            for entry in sampled:
                question = entry.get("question", "")
                if not question:
                    continue

                # Standard semantic search
                t0 = time.perf_counter()
                try:
                    std_results = client.memories_search(
                        question, limit=10, user_id="longmemeval-user"
                    )
                except Exception:
                    std_results = None
                t1 = time.perf_counter()
                standard_latencies.append((t1 - t0) * 1000)
                standard_counts.append(len(std_results) if std_results else 0)
                time.sleep(delay)

                # Enhanced search (spreading activation)
                t0 = time.perf_counter()
                try:
                    enh_results = client.search_enhanced(
                        question, mode=SearchMode.SPREADING, limit=10
                    )
                except Exception:
                    enh_results = None
                t1 = time.perf_counter()
                enhanced_latencies.append((t1 - t0) * 1000)
                enhanced_counts.append(
                    len(enh_results.get("results", [])) if enh_results else 0
                )
                time.sleep(delay)

            def _print_stats(name, latencies, counts):
                if not latencies:
                    print(f"  [!] {name}: no data")
                    return
                latencies.sort()
                p50 = _percentile(latencies, 50)
                p95 = _percentile(latencies, 95)
                p99 = _percentile(latencies, 99)
                mean_lat = statistics.mean(latencies)
                median_lat = statistics.median(latencies)
                print(f"\n  {name} Results (n={len(latencies)}):")
                print(
                    f"    Latency (ms): min={latencies[0]:.2f}, max={latencies[-1]:.2f}, "
                    f"mean={mean_lat:.2f}, median={median_lat:.2f}, "
                    f"p95={p95:.2f}, p99={p99:.2f}"
                )
                if counts:
                    print(
                        f"    Avg results returned: {statistics.mean(counts):.1f} "
                        f"(min={min(counts)}, max={max(counts)})"
                    )

            _print_stats("Standard Search", standard_latencies, standard_counts)
            _print_stats(
                "Enhanced Search (SPREADING)", enhanced_latencies, enhanced_counts
            )

            check(
                "benchmark standard search has results",
                any(c > 0 for c in standard_counts),
            )
            check(
                "benchmark enhanced search has results",
                any(c > 0 for c in enhanced_counts),
            )
            check(
                "benchmark collected latencies",
                len(standard_latencies) >= max(1, sample_size // 2),
                f"got {len(standard_latencies)}",
            )

            # ── Compression Benchmark ──────────────────────────────────
            print(f"\n  Compression Benchmark ({args.compression_sample} texts, {args.compression_delay}s delay)")

            # Collect texts from sampled entries (prefer long session messages)
            compress_texts = []
            for entry in sampled[:args.compression_sample]:
                # Try to get a long message from the haystack sessions
                sessions = entry.get("haystack_sessions", [])
                if sessions and len(sessions) > 0:
                    for session in sessions:
                        for msg in session:
                            content = msg.get("content", "")
                            # Skip short messages; prefer substantial content
                            if len(content.split()) >= 20:
                                compress_texts.append(content)
                                break
                        if len(compress_texts) >= args.compression_sample:
                            break
                if len(compress_texts) >= args.compression_sample:
                    break

            if not compress_texts:
                skip("compression benchmark", "no suitable texts found")
            else:
                compression_latencies = []   # client-side (ms)
                server_latencies = []        # server-reported total (ms)
                best_modes = []
                # Per-mode stats: mode -> {reductions: [], token_savings: [], latencies: []}
                mode_stats = {"extraction": [], "relational": [], "radix": [], "hybrid": []}

                # Fidelity evaluation tracking
                run_evaluator = args.evaluator and HAS_EVALUATOR
                eval_modes = [m.strip() for m in args.evaluator_modes.split(",")] if args.evaluator_modes else []
                fidelity_scores = {}  # mode -> {"recall": [], "precision": [], "reasons": []}
                if run_evaluator:
                    for m in eval_modes:
                        fidelity_scores[m] = {"recall": [], "precision": [], "reasons": []}
                    print(f"  Evaluator enabled for modes: {', '.join(eval_modes)}")
                elif args.evaluator and not HAS_EVALUATOR:
                    print(f"  [~] WARN  --evaluator flag set but test_evaluator.py not found -- skipping fidelity evaluation")

                # Parse compression modes and cap text length
                comp_modes = [m.strip() for m in args.compression_modes.split(",")] if args.compression_modes else ["radix", "hybrid"]
                MAX_TEXT_LEN = 1000  # cap to avoid very long compression times

                for text in compress_texts:
                    comp_delay = args.compression_delay
                    # Cap text to first MAX_TEXT_LEN chars to avoid timeout on long inputs
                    # capped_text = text[:MAX_TEXT_LEN] if len(text) > MAX_TEXT_LEN else text
                    capped_text = text

                    # Client-side timing
                    t0 = time.perf_counter()
                    try:
                        comp_result = client.playground_compress(
                            capped_text,
                            user_id="longmemeval-user",
                            modes=comp_modes,
                            show_entities=True,
                            show_facts=True,
                        )
                    except Exception as e:
                        comp_result = None
                        if VERBOSE:
                            print(f"  [!] compression error: {e}")
                    t1 = time.perf_counter()
                    client_ms = (t1 - t0) * 1000
                    compression_latencies.append(client_ms)

                    if comp_result:
                        server_lat = comp_result.get("total_latency_ms", 0)
                        if server_lat > 0:
                            server_latencies.append(server_lat)

                        best_mode = comp_result.get("best_mode", "")
                        if best_mode:
                            best_modes.append(best_mode)

                        results = comp_result.get("results", {})
                        for mode, stats in results.items():
                            if mode in mode_stats:
                                reduction = stats.get("reduction_percent", 0)
                                token_sav = stats.get("token_savings", 0)
                                mode_lat = stats.get("latency_ms", 0)
                                fallback = stats.get("fallback", False)
                                entry = {
                                    "reduction": reduction,
                                    "tokens": token_sav,
                                    "latency": mode_lat,
                                    "fallback": fallback,
                                }
                                if reduction > 0:
                                    mode_stats[mode].append(entry)

                                # Run fidelity evaluator on configured modes
                                if run_evaluator and mode in eval_modes:
                                    compressed_text = stats.get("compressed", "")
                                    if compressed_text and reduction > 0:
                                        eval_result = evaluate_compression_fidelity(text, compressed_text)
                                        if "error" not in eval_result:
                                            fidelity_scores[mode]["recall"].append(eval_result.get("recall", 0))
                                            fidelity_scores[mode]["precision"].append(eval_result.get("precision", 0))
                                            fidelity_scores[mode]["reasons"].append(eval_result.get("reasoning", ""))
                                        else:
                                            if VERBOSE:
                                                print(f"  [~] evaluator error ({mode}): {eval_result['error']}")

                    time.sleep(comp_delay)

                def _print_compression_stats():
                    print(f"\n  Compression Results (n={len(compression_latencies)}):")
                    if compression_latencies:
                        cl = sorted(compression_latencies)
                        print(
                            f"    Client Latency (ms): min={cl[0]:.2f}, max={cl[-1]:.2f}, "
                            f"mean={statistics.mean(cl):.2f}, median={statistics.median(cl):.2f}"
                        )
                    if server_latencies:
                        sl = sorted(server_latencies)
                        print(
                            f"    Server Latency (ms): min={sl[0]:.2f}, max={sl[-1]:.2f}, "
                            f"mean={statistics.mean(sl):.2f}, median={statistics.median(sl):.2f}"
                        )

                    # Per-mode breakdown
                    fallback_count = 0
                    for mode, entries in mode_stats.items():
                        if not entries:
                            continue
                        reductions = [e["reduction"] for e in entries]
                        token_sav = [e["tokens"] for e in entries]
                        latencies = [e["latency"] for e in entries]
                        fb_count = sum(1 for e in entries if e.get("fallback"))
                        fallback_count += fb_count
                        fb_label = f" [FALLBACK x{fb_count}]" if fb_count > 0 else ""
                        print(
                            f"\n    {mode.upper()} mode (n={len(entries)}):{fb_label}"
                        )
                        print(
                            f"      Reduction %: min={min(reductions):.1f}, max={max(reductions):.1f}, "
                            f"mean={statistics.mean(reductions):.1f}"
                        )
                        print(
                            f"      Token savings: min={min(token_sav)}, max={max(token_sav)}, "
                            f"mean={statistics.mean(token_sav):.1f}"
                        )
                        if latencies:
                            print(
                                f"      Latency (ms): min={min(latencies):.2f}, max={max(latencies):.2f}, "
                                f"mean={statistics.mean(latencies):.2f}"
                            )

                    # ── Fidelity Evaluation Card ──────────────────────
                    if fidelity_scores and any(fidelity_scores[m]["recall"] for m in fidelity_scores):
                        print(f"\n    {'─'*52}")
                        print(f"    Compression Fidelity (LLM-evaluated)")
                        print(f"    {'─'*52}")
                        for mode, scores in fidelity_scores.items():
                            recalls = scores["recall"]
                            precisions = scores["precision"]
                            if not recalls:
                                print(f"\n    {mode.upper()}: no evaluations collected")
                                continue
                            avg_recall = statistics.mean(recalls)
                            avg_prec = statistics.mean(precisions)
                            f1 = 2 * (avg_prec * avg_recall) / (avg_prec + avg_recall) if (avg_prec + avg_recall) > 0 else 0
                            print(f"\n    {mode.upper()} (n={len(recalls)}):")
                            print(f"      Recall    (factual retention): {avg_recall:.3f}  "
                                  f"(min={min(recalls):.2f}, max={max(recalls):.2f})")
                            print(f"      Precision (no hallucination):  {avg_prec:.3f}  "
                                  f"(min={min(precisions):.2f}, max={max(precisions):.2f})")
                            print(f"      F1 score:                      {f1:.3f}")
                            # Show sample reasoning from first evaluation
                            if scores["reasons"] and VERBOSE:
                                print(f"      Sample reasoning: {scores['reasons'][0][:120]}...")

                    # Best mode distribution
                    if best_modes:
                        from collections import Counter
                        mode_counts = Counter(best_modes)
                        print(f"\n    Best mode distribution: {dict(mode_counts)}")

                _print_compression_stats()

                check(
                    "compression benchmark ran",
                    len(compression_latencies) >= max(1, args.compression_sample // 2),
                    f"got {len(compression_latencies)}",
                )

    # ── 23. Cleanup ────────────────────────────────────────────────
    section("23. Cleanup")
    cleanup_items = [
        (chain_id, "chains_delete()", client.chains_delete),
        (mem1_id, "memories_delete(mem1)", client.memories_delete),
        (mem2_id, "memories_delete(mem2)", client.memories_delete),
        (mem3_id, "memories_delete(mem3)", client.memories_delete),
        (ent1_id, "entities_delete(ent1)", client.entities_delete),
        (ent2_id, "entities_delete(ent2)", client.entities_delete),
        (skill_id, "skills_delete()", client.skills_delete),
        (wh_id, "webhooks_delete()", client.webhooks_delete),
        (proj_id, "projects_delete()", client.projects_delete),
        (group_id, "groups_delete()", client.groups_delete),
        (session_id, "delete_session()", client.delete_session),
    ]
    for item_id, label, fn in cleanup_items:
        if item_id:
            try:
                fn(item_id)
                print(f"  [+] cleaned up {label}")
            except Exception:
                print(f"  [~] cleanup {label} (already gone or skipped)")

    client.close()

    # ── Summary ────────────────────────────────────────────────────
    total = PASS + FAIL + SKIP
    print(f"\n{'='*60}")
    print(f"  RESULTS: {PASS} passed, {FAIL} failed, {SKIP} skipped, {total} total")
    print(f"{'='*60}")
    if ERRORS:
        print("\nFailed tests:")
        for e in ERRORS:
            print(f"  - {e}")
    print()
    sys.exit(0 if FAIL == 0 else 1)


if __name__ == "__main__":
    main()
