# Benchmarks — real datasets only for competitive claims

## Mock vs real (do not confuse these)

| Flag / mode | What runs | Use for |
|-------------|-----------|---------|
| `--mock` | In-memory lexical fake store | **Plumbing only** — never cite as product quality |
| **no `--mock`** (default) | **Live Neo4j + Qdrant + embeddings** | Real retrieval measurement |
| Official LoCoMo JSON | Real dialog turns + QA from [snap-research/locomo](https://github.com/snap-research/locomo) | Primary retrieval suite |

`--mock` does **not** mean “fake dataset file”. It means a **fake memory backend**. Always omit `--mock` for readiness and product comparison.

## Datasets

| Dataset | Source | Location after setup |
|---------|--------|----------------------|
| **locomo** (full) | Official `locomo10.json` (~5.8k turns, ~1.5k QA) | `data/benchmarks/locomo/dataset.json` via `scripts/convert_locomo.py` |
| **locomo** raw | GitHub | `data/benchmarks/raw/locomo10.json` |
| **longmemeval** | Packaged fixture (expand when full dump available) | `data/benchmarks/longmemeval/` or `internal/evaluation/longmemeval/` |
| **beam_1m / beam_10m** | Packaged real-format BEAM fixtures | `internal/evaluation/beam/` |

### Install official LoCoMo

```bash
mkdir -p data/benchmarks/raw
curl -fsSL -o data/benchmarks/raw/locomo10.json \
  https://raw.githubusercontent.com/snap-research/locomo/main/data/locomo10.json
python3 scripts/convert_locomo.py \
  data/benchmarks/raw/locomo10.json \
  data/benchmarks/locomo/dataset.json
```

## Live run (required for product readiness)

```bash
# Stores
docker compose up -d neo4j qdrant redis

# Env: NEO4J_*, QDRANT_*, OPENAI_API_KEY (embeddings + judge fallback)
# LLM-as-judge (required for publishable=true):
export EVALUATOR_API_KEY="$OPENAI_API_KEY"   # or a dedicated judge key
export EVALUATOR_PROVIDER=openai
export EVALUATOR_MODEL=gpt-4o-mini

# Full official LoCoMo (no mock)
./scripts/run-real-benchmarks.sh locomo

# Faster slice (still real data + live stores + LLM judge)
LIMIT=50 ./scripts/run-real-benchmarks.sh locomo
```

Results: `docs/benchmarks/results/*-live-*.json`

## Publishability

A result is **`publishable: true`** only when:

1. Not run with `--mock`
2. Zero ingest/search errors
3. Every question scored by a configured **LLM judge** (`score_method: "llm_judge"`, `evaluator_configured: true`)

Token-F1 / lexical scores without an LLM judge are useful for regression, **not** competitive marketing.

### Judge env priority

`EVALUATOR_API_KEY` → `COMPRESSION_VERIFY_API_KEY` → `OPENAI_API_KEY` / `LLM_API_KEY`  
Provider defaults to **openai** / **gpt-4o-mini** when only an OpenAI key is set.

## Latest live LoCoMo snapshot (this repo)

See `docs/benchmarks/results/locomo-live-real.json` — live Neo4j+Qdrant, real LoCoMo-derived questions (limit 30), `publishable: false` until LLM-judged.
