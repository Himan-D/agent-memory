# GitHub Copilot Coding Agent

This repository is configured for GitHub Copilot coding agent without an OpenAI API key.

## How It Works

Copilot coding agent runs inside GitHub's own agent environment. It can be delegated tasks from GitHub Issues, the GitHub Agents panel, VS Code, or other supported GitHub entry points. It explores the repository, edits code, runs commands, and opens a pull request for review.

Repository-specific guidance is provided by:

- `.github/copilot-instructions.md` for repository-wide Copilot instructions.
- `AGENTS.md` and `.github/AGENTS.md` for agent workflow and repository automation rules.
- `.github/workflows/copilot-setup-steps.yml` to preinstall Go and Node dependencies in Copilot's coding-agent environment.

## No OpenAI API Secret Required

This setup does not use `OPENAI_API_KEY`. Access is controlled by GitHub Copilot licensing, repository settings, and GitHub permissions.

## Delegating Work

Use one of the supported GitHub Copilot entry points:

1. Create a focused GitHub Issue with acceptance criteria.
2. Assign the Issue to Copilot, or start a Copilot task from the GitHub agent UI.
3. Copilot should create a branch and PR.
4. Review the PR and let CI enforce quality.

Good issue prompts include:

```text
Implement durable wiki persistence.

Acceptance criteria:
- Store wiki pages, sources, logs, and index outside process memory.
- Preserve current API behavior.
- Add focused tests for save/load behavior.
- Run go build ./... and go test ./cmd/... ./internal/...
```

## Guardrails

- Copilot must not push directly to `master`.
- Copilot PRs should use branch names compatible with existing automation: `cursor/<task>-6161`.
- CI must pass before auto-merge.
- Secrets, binaries, dependency folders, and generated build output must not be committed.

## References

- GitHub Docs: Assigning tasks to Copilot coding agent
- GitHub Docs: Repository custom instructions via `.github/copilot-instructions.md`
- GitHub Docs: Copilot coding agent setup steps via `.github/workflows/copilot-setup-steps.yml`

