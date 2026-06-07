# OpenAI Repo Bot

The OpenAI Repo Bot is a guarded GitHub Actions automation that can monitor this repository and create focused documentation/governance contributions.

## What It Can Do

- Run weekly in `monitor` mode and create or update an issue with production-readiness findings.
- Run manually in `contribute` mode with a task prompt.
- Inspect repository status, recent commits, tracked file structure, and key project guidance files.
- Use the OpenAI Responses API to generate a structured plan or a narrow docs contribution.
- Push contributions to `cursor/openai-repo-bot-6161`.
- Open a PR to `master` and label it `agent` and `automerge`.

## What It Must Not Do

- Push directly to `master`.
- Commit secrets, binaries, build outputs, `node_modules`, or generated docs output.
- Modify application code without a dedicated reviewed workflow.
- Bypass CI or branch protection.

Direct writes to `master` are intentionally avoided. The repository already auto-merges eligible `cursor/*-6161` PRs after `CI Success`, which gives the bot a safe path to contribute without weakening release controls.

## Setup

Add this repository secret:

```text
OPENAI_API_KEY
```

The workflow uses `GITHUB_TOKEN` automatically.

## Manual Run

1. Open **Actions**.
2. Select **OpenAI Repo Bot**.
3. Choose `monitor` or `contribute`.
4. Provide a task, for example:

```text
Improve the architecture docs with the next milestone plan for wiki persistence and retrieval quality.
```

## Verification

Contribution mode runs:

```bash
go build ./...
go test ./cmd/... ./internal/...
```

The full integration package under `tests/` still requires Neo4j and Qdrant to be running.

