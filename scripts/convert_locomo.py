#!/usr/bin/env python3
"""Convert official snap-research/locomo locomo10.json into Hystersis BenchmarkDataset JSON.

Usage:
  python3 scripts/convert_locomo.py [/path/to/locomo10.json] [out.json]

Default input: downloads official file if missing under data/benchmarks/raw/locomo10.json
Default output: data/benchmarks/locomo/dataset.json
"""
from __future__ import annotations

import json
import sys
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_URL = "https://raw.githubusercontent.com/snap-research/locomo/main/data/locomo10.json"
RAW = ROOT / "data" / "benchmarks" / "raw" / "locomo10.json"
OUT = ROOT / "data" / "benchmarks" / "locomo" / "dataset.json"

# LoCoMo category ints → labels matching published per-category tables
CAT_MAP = {
    1: "single_hop",
    2: "temporal",
    3: "multi_hop",
    4: "open_domain",
    5: "adversarial",
}


def ensure_raw(path: Path) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and path.stat().st_size > 1000:
        return path
    print(f"downloading {DEFAULT_URL} -> {path}", file=sys.stderr)
    urllib.request.urlretrieve(DEFAULT_URL, path)
    return path


def turn_text(turn: dict, session_date: str = "") -> str:
    speaker = turn.get("speaker") or turn.get("role") or "Speaker"
    text = turn.get("text") or turn.get("content") or ""
    body = f"{speaker}: {text}".strip()
    # Prepend session datetime so temporal QA can retrieve by date cues.
    if session_date:
        return f"[{session_date}] {body}"
    return body


def convert(raw: list) -> dict:
    memories: list[dict] = []
    questions: list[dict] = []
    mem_by_dia: dict[str, str] = {}  # dia_id -> memory id

    for conv in raw:
        sample_id = conv.get("sample_id") or conv.get("id") or "conv"
        conversation = conv.get("conversation") or {}
        # Index turns by dia_id
        for key, val in conversation.items():
            if not key.startswith("session_") or key.endswith("_date_time"):
                continue
            if not isinstance(val, list):
                continue
            session_num = key.replace("session_", "")
            session_id = f"{sample_id}_s{session_num}"
            # LoCoMo stores session dates as conversation["session_N_date_time"]
            date_key = f"session_{session_num}_date_time"
            session_date = ""
            raw_date = conversation.get(date_key)
            if isinstance(raw_date, str):
                session_date = raw_date.strip()
            for turn in val:
                if not isinstance(turn, dict):
                    continue
                dia_id = turn.get("dia_id") or ""
                content = turn_text(turn, session_date)
                if not content or content.endswith(":"):
                    continue
                mid = f"{sample_id}_{dia_id.replace(':', '_')}" if dia_id else f"{sample_id}_{len(memories)}"
                memories.append(
                    {
                        "id": mid,
                        "content": content,
                        "user_id": sample_id,
                        "session_id": session_id,
                        "metadata": {
                            "dia_id": dia_id,
                            "source": "locomo",
                            "session_date": session_date,
                            "conversation_id": sample_id,
                        },
                    }
                )
                if dia_id:
                    # Always namespace by conversation — bare dia_ids collide across convs.
                    mem_by_dia[f"{sample_id}:{dia_id}"] = mid

        for i, qa in enumerate(conv.get("qa") or []):
            if not isinstance(qa, dict):
                continue
            q = qa.get("question") or ""
            ans = qa.get("answer")
            if isinstance(ans, (int, float)):
                ans = str(ans)
            ans = (ans or "").strip()
            cat = CAT_MAP.get(qa.get("category"), "single_hop")
            # Adversarial (cat 5) may have empty gold ("not in conversation").
            if not q:
                continue
            if not ans and cat != "adversarial":
                continue
            if not ans and cat == "adversarial":
                ans = "Not mentioned in the conversation"
            evidence = qa.get("evidence") or []
            # Prefer first evidence turn as expected memory id when present
            memory_id = ""
            evidence_ids: list[str] = []
            for ev in evidence:
                key = f"{sample_id}:{ev}"
                if key in mem_by_dia:
                    mid = mem_by_dia[key]
                    evidence_ids.append(mid)
                    if not memory_id:
                        memory_id = mid
            questions.append(
                {
                    "id": f"{sample_id}_q{i:04d}",
                    "question": q,
                    "session_id": sample_id,
                    "memory_id": memory_id,
                    "evidence_ids": evidence_ids,
                    "category": cat,
                    "ground_truth": ans,
                    "evidence": evidence,
                }
            )

    return {
        "name": "locomo",
        "description": "Official LoCoMo (snap-research/locomo) — full 10 conversations converted for Hystersis retrieval eval",
        "source": "https://github.com/snap-research/locomo",
        "source_file": "data/locomo10.json",
        "questions": questions,
        "memories": memories,
    }


def main() -> None:
    inp = Path(sys.argv[1]) if len(sys.argv) > 1 else ensure_raw(RAW)
    out = Path(sys.argv[2]) if len(sys.argv) > 2 else OUT
    if not inp.exists():
        ensure_raw(inp)
    raw = json.loads(inp.read_text())
    if not isinstance(raw, list):
        raise SystemExit("expected locomo10.json to be a JSON array")
    ds = convert(raw)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(ds, indent=2, ensure_ascii=False) + "\n")
    print(
        f"wrote {out} memories={len(ds['memories'])} questions={len(ds['questions'])}",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
