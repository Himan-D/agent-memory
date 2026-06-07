#!/usr/bin/env node

import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const mode = process.env.BOT_MODE || "monitor";
const task = process.env.BOT_TASK || "Review the repository for production-readiness gaps.";
const model = process.env.BOT_MODEL || "gpt-5";
const baseBranch = process.env.BOT_BASE_BRANCH || "master";
const botBranch = process.env.BOT_BRANCH || "cursor/openai-repo-bot-6161";
const repo = process.env.GITHUB_REPOSITORY || "";
const runURL = process.env.GITHUB_RUN_URL || "";

const allowedContributionFiles = [
  /^docs\/.+/,
  /^README\.md$/,
  /^AGENTS\.md$/,
  /^\.github\/AGENTS\.md$/,
];

function run(command, args, options = {}) {
  return execFileSync(command, args, {
    cwd: root,
    encoding: "utf8",
    stdio: options.stdio || ["ignore", "pipe", "pipe"],
    maxBuffer: options.maxBuffer || 20 * 1024 * 1024,
  }).trim();
}

function runOptional(command, args, fallback = "") {
  try {
    return run(command, args);
  } catch {
    return fallback;
  }
}

function requireEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

function readIfExists(path, maxChars = 12000) {
  const absolute = resolve(root, path);
  if (!existsSync(absolute)) return "";
  return readFileSync(absolute, "utf8").slice(0, maxChars);
}

function collectRepoContext() {
  const trackedFiles = run("git", ["ls-files"]).split("\n").filter(Boolean);
  const fileSummary = trackedFiles
    .filter((file) => !file.includes("node_modules/") && !file.includes("/build/lib/"))
    .slice(0, 700)
    .join("\n");

  const keyFiles = {
    "AGENTS.md": readIfExists("AGENTS.md"),
    ".github/AGENTS.md": readIfExists(".github/AGENTS.md"),
    "README.md": readIfExists("README.md"),
    "go.mod": readIfExists("go.mod", 6000),
    ".github/workflows/ci.yml": readIfExists(".github/workflows/ci.yml", 10000),
    "docs/architecture.md": readIfExists("docs/architecture.md", 12000),
  };

  return {
    task,
    mode,
    baseBranch,
    repo,
    runURL,
    status: runOptional("git", ["status", "--short"]),
    recentCommits: runOptional("git", ["log", "--oneline", "-8"]),
    fileSummary,
    keyFiles,
  };
}

function buildPrompt(context) {
  return `You are OpenAI Repo Bot for the Hystersis agent-memory repository.

Repository policy:
- Base branch is master.
- Automation must use cursor/<name>-6161 branches and PRs.
- Never commit secrets, binaries, node_modules, dist, .open-next, or generated docs output.
- Go changes must pass go build ./... and focused tests.
- Prefer narrow, production-grade changes over broad rewrites.

Task:
${context.task}

Mode:
${context.mode}

Return strict JSON only. Do not wrap it in markdown.

For monitor mode, return:
{
  "title": "short issue title",
  "summary": "short executive summary",
  "findings": [{"severity":"high|medium|low","area":"...","detail":"...","recommendation":"..."}],
  "nextActions": ["..."]
}

For contribute mode, only propose documentation/governance changes in docs/, README.md, AGENTS.md, or .github/AGENTS.md.
Return:
{
  "title": "conventional commit PR title",
  "summary": "what will change",
  "changes": [{"path":"docs/example.md","content":"full replacement file content"}],
  "verification": ["commands the workflow should run"]
}

Repository context:
${JSON.stringify(context, null, 2)}`;
}

async function callOpenAI(prompt) {
  const apiKey = requireEnv("OPENAI_API_KEY");
  const response = await fetch("https://api.openai.com/v1/responses", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${apiKey}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      model,
      input: [
        {
          role: "developer",
          content:
            "You are a careful repository automation agent. Return only valid JSON. Do not request secrets. Do not propose direct pushes to protected branches.",
        },
        { role: "user", content: prompt },
      ],
      text: {
        format: {
          type: "json_object",
        },
      },
    }),
  });

  const body = await response.text();
  if (!response.ok) {
    throw new Error(`OpenAI API error ${response.status}: ${body}`);
  }

  const parsed = JSON.parse(body);
  const outputText =
    parsed.output_text ||
    parsed.output?.flatMap((item) => item.content || []).find((part) => part.type === "output_text")?.text;

  if (!outputText) {
    throw new Error(`OpenAI response did not include output text: ${body}`);
  }
  return JSON.parse(outputText);
}

function assertSafePath(path) {
  const normalized = relative(root, resolve(root, path));
  if (normalized.startsWith("..") || normalized === "") {
    throw new Error(`Unsafe output path: ${path}`);
  }
  if (!allowedContributionFiles.some((pattern) => pattern.test(normalized))) {
    throw new Error(`Contribution path is outside the allowlist: ${normalized}`);
  }
  return normalized;
}

function writeContribution(changes) {
  if (!Array.isArray(changes) || changes.length === 0) {
    throw new Error("Contribute mode requires at least one change");
  }

  for (const change of changes) {
    const safePath = assertSafePath(change.path);
    if (typeof change.content !== "string" || change.content.length < 20) {
      throw new Error(`Invalid content for ${safePath}`);
    }
    writeFileSync(resolve(root, safePath), change.content.endsWith("\n") ? change.content : `${change.content}\n`);
  }
}

async function githubRequest(method, path, body) {
  const token = requireEnv("GITHUB_TOKEN");
  const response = await fetch(`https://api.github.com${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
      "X-GitHub-Api-Version": "2022-11-28",
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await response.text();
  if (!response.ok) {
    throw new Error(`GitHub API error ${response.status} ${path}: ${text}`);
  }
  return text ? JSON.parse(text) : null;
}

async function createOrUpdateMonitorIssue(result) {
  const [owner, name] = repo.split("/");
  if (!owner || !name) throw new Error("GITHUB_REPOSITORY must be owner/name");

  const title = result.title || "OpenAI Repo Bot monitor report";
  const body = [
    `## ${title}`,
    "",
    result.summary || "Repository monitor report.",
    "",
    "## Findings",
    ...(result.findings || []).map(
      (finding) =>
        `- **${finding.severity || "medium"}** ${finding.area || "repository"}: ${finding.detail || ""}\n  Recommendation: ${
          finding.recommendation || "Review and prioritize."
        }`,
    ),
    "",
    "## Next Actions",
    ...(result.nextActions || []).map((action) => `- ${action}`),
    "",
    runURL ? `Workflow run: ${runURL}` : "",
  ].join("\n");

  const issues = await githubRequest(
    "GET",
    `/repos/${owner}/${name}/issues?state=open&labels=openai-repo-bot&per_page=20`,
  );
  const existing = issues.find((issue) => issue.title.startsWith("OpenAI Repo Bot"));
  if (existing) {
    await githubRequest("POST", `/repos/${owner}/${name}/issues/${existing.number}/comments`, { body });
    return;
  }

  await githubRequest("POST", `/repos/${owner}/${name}/issues`, {
    title: `OpenAI Repo Bot: ${title}`,
    body,
    labels: ["openai-repo-bot", "agent"],
  });
}

async function createPullRequest(result) {
  const [owner, name] = repo.split("/");
  if (!owner || !name) throw new Error("GITHUB_REPOSITORY must be owner/name");

  run("git", ["fetch", "origin", baseBranch], { stdio: ["ignore", "pipe", "pipe"] });
  runOptional("git", ["branch", "-D", botBranch]);
  run("git", ["checkout", "-B", botBranch, `origin/${baseBranch}`], { stdio: ["ignore", "pipe", "pipe"] });

  writeContribution(result.changes);

  const diff = runOptional("git", ["diff", "--stat"]);
  if (!diff) {
    throw new Error("OpenAI bot produced no file changes");
  }

  run("go", ["build", "./..."], { stdio: "inherit" });
  run("go", ["test", "./cmd/...", "./internal/..."], { stdio: "inherit" });

  run("git", ["add", "-A"]);
  const title = result.title || "docs: update repository guidance";
  const commitTitle = /^(feat|fix|chore|docs|refactor|test|ci|build|perf|style)(\(.+\))?: .+/i.test(title)
    ? title
    : `docs: ${title}`;
  run("git", ["commit", "-m", commitTitle], { stdio: "inherit" });
  run("git", ["push", "--force-with-lease", "-u", "origin", botBranch], { stdio: "inherit" });

  const body = [
    result.summary || "OpenAI Repo Bot contribution.",
    "",
    "## Verification",
    "- `go build ./...`",
    "- `go test ./cmd/... ./internal/...`",
    "",
    runURL ? `Workflow run: ${runURL}` : "",
  ].join("\n");

  const existing = await githubRequest(
    "GET",
    `/repos/${owner}/${name}/pulls?state=open&head=${owner}:${encodeURIComponent(botBranch)}&base=${baseBranch}`,
  );
  if (existing.length) {
    await githubRequest("PATCH", `/repos/${owner}/${name}/pulls/${existing[0].number}`, {
      title: commitTitle,
      body,
    });
    return;
  }

  const pr = await githubRequest("POST", `/repos/${owner}/${name}/pulls`, {
    title: commitTitle,
    head: botBranch,
    base: baseBranch,
    body,
    maintainer_can_modify: true,
  });
  await githubRequest("POST", `/repos/${owner}/${name}/issues/${pr.number}/labels`, {
    labels: ["agent", "automerge", "openai-repo-bot"],
  });
}

async function main() {
  const context = collectRepoContext();
  const result = await callOpenAI(buildPrompt(context));

  if (mode === "monitor") {
    await createOrUpdateMonitorIssue(result);
    return;
  }
  if (mode === "contribute") {
    await createPullRequest(result);
    return;
  }
  throw new Error(`Unsupported BOT_MODE: ${mode}`);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});

