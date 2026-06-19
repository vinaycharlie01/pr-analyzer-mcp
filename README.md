# PR Analyzer MCP

> A Model Context Protocol (MCP) server that gives AI assistants deep insight into pull requests — comprehensive analysis, step-by-step migration plans, dependency graphs, architecture impact maps, and full documentation, all through natural language.

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-87%25-brightgreen)](coverage.out)
[![MCP](https://img.shields.io/badge/MCP-compatible-purple)](https://modelcontextprotocol.io)

---

## What Is This?

PR Analyzer MCP exposes **12 specialized tools** over the [Model Context Protocol](https://modelcontextprotocol.io) so that AI assistants (Claude, GitHub Copilot, ChatGPT, Cursor, Windsurf) can answer deep questions about pull requests without needing direct repository access.

Ask your AI assistant things like:

- *"Analyze PR #142 in myorg/backend and give me a migration checklist"*
- *"Why was `internal/service/auth.go` changed in that PR?"*
- *"Generate migration documentation to port PR #88 from the monolith to the new microservice"*
- *"What architecture layers does this PR touch?"*

The server connects to **GitHub** and **Bitbucket Cloud / Data Center** and works entirely over `stdio` — no web server, no open ports, no credentials in prompts.

---

## Features

| Tool | Description |
|------|-------------|
| `analyze_pr` | Full PR analysis: executive summary, business & technical purpose, architecture impact, DB impact, K8s impact, rollback strategy |
| `explain_change` | Explains WHY a file changed, WHAT changed, HOW to migrate it, and what can break |
| `generate_migration_plan` | Step-by-step migration plan to port PR changes to a target repository |
| `dependency_analysis` | Identifies all package, module, service, and config dependencies introduced by a PR |
| `architecture_map` | Maps which architectural layers (domain, application, adapter, port, service) are affected |
| `find_related_files` | Finds files likely impacted by the PR that were not explicitly changed |
| `find_required_dependencies` | Lists Go packages and modules required to migrate PR changes |
| `compare_repositories` | Diffs two repositories and suggests migration actions |
| `generate_migration_checklist` | Safety checklist covering preparation, testing, deployment, and rollback |
| `explain_code_flow` | Traces execution flow from an entry point through the changed code |
| `generate_feature_summary` | Business and technical feature summary for stakeholders |
| `generate_migration_documentation` | Complete migration doc with overview, steps, checklist, and validation |

---

## Architecture

The project follows **Hexagonal Architecture** (Ports & Adapters) with Domain-Driven Design:

```
┌─────────────────────────────────────────────────────────┐
│                    MCP Adapter (stdio)                  │
│              12 tools via JSON-RPC 2.0                  │
└──────────────────────┬──────────────────────────────────┘
                       │ inbound port
┌──────────────────────▼──────────────────────────────────┐
│               Application Use Case                      │
│            PRAnalyzerUseCase (orchestrator)             │
└──┬───────────────────┬──────────────────┬───────────────┘
   │                   │                  │
   ▼                   ▼                  ▼
analysis.Analyzer  migration.Planner  architecture.Mapper
   │
   │  outbound port
┌──▼──────────────────────────────────────────────────────┐
│                  VCS Adapters                           │
│        GitHub (go-github v66)  │  Bitbucket Cloud/DC   │
└─────────────────────────────────────────────────────────┘
```

**Key packages:**

```
cmd/server/                  → entrypoint, cobra CLI, DI wiring
internal/
  adapters/
    mcp/                     → MCP server, tool registration, formatters
    github/                  → GitHub REST client adapter
    bitbucket/               → Bitbucket REST client adapter
    config/                  → Wire DI providers
  application/usecase/       → PRAnalyzerUseCase (core orchestration)
  domain/
    entity/                  → PullRequest, AnalysisResult, MigrationPlan …
    valueobject/             → RepositoryRef, PRNumber
  port/
    inbound/                 → PRAnalyzerPort interface
    outbound/                → VCSPort interface
  service/
    analysis/                → Static analysis (go/ast, go/parser)
    architecture/            → Layer detection, K8s/config impact
    dependency/              → Dependency graph (dominikbraun/graph)
    migration/               → Migration step generation
pkg/
  config/                    → Viper configuration loader
  errors/                    → Typed application errors
  logger/                    → slog JSON logger
  observability/             → OpenTelemetry tracing + metrics
```

---

## Prerequisites

- **Go 1.24+**
- A **GitHub Personal Access Token** (for GitHub repositories)
- A **Bitbucket Token or App Password** (for Bitbucket repositories)
- One of the supported AI clients listed below

---

## Installation

```bash
# Clone
git clone https://github.com/vinaycharlie01/pr-analyzer-mcp.git
cd pr-analyzer-mcp

# Build
go build -o pr-analyzer-mcp ./cmd/server

# Or run directly
go run ./cmd/server
```

---

## Configuration

### Environment Variables (Recommended)

Copy `.env.example` and fill in your credentials:

```bash
cp .env.example .env
```

```env
# GitHub
GITHUB_TOKEN=ghp_your_personal_access_token

# Bitbucket Cloud (use Token OR Username + App Password)
BITBUCKET_TOKEN=your_bitbucket_token
BITBUCKET_USERNAME=your_username
BITBUCKET_APP_PASSWORD=your_app_password

# Logging
LOG_LEVEL=info          # debug | info | warn | error
LOG_FORMAT=json         # json | text

# Observability (optional)
OTEL_ENABLED=false
OTEL_SERVICE_NAME=pr-analyzer-mcp
```

### Config File

Alternatively, edit `configs/config.yaml`:

```yaml
github:
  token: "${GITHUB_TOKEN}"
  base_url: "https://api.github.com"   # override for GitHub Enterprise

bitbucket:
  base_url: "https://api.bitbucket.org/2.0"
  datacenter_url: "https://bitbucket.example.com"  # for Data Center
  token: "${BITBUCKET_TOKEN}"
  username: "${BITBUCKET_USERNAME}"
  app_password: "${BITBUCKET_APP_PASSWORD}"

logging:
  level: "info"
  format: "json"

analysis:
  max_depth: 10
  timeout_seconds: 30
```

Pass a custom config path with `--config`:

```bash
./pr-analyzer-mcp --config /path/to/my-config.yaml
```

---

## AI Client Setup

### Claude (Anthropic)

**Claude Desktop** — edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "pr-analyzer": {
      "command": "/absolute/path/to/pr-analyzer-mcp",
      "args": [],
      "env": {
        "GITHUB_TOKEN": "ghp_your_token_here",
        "BITBUCKET_TOKEN": "your_bitbucket_token",
        "LOG_LEVEL": "warn"
      }
    }
  }
}
```

**Claude Code (CLI)** — add to your project's `.claude/settings.json`:

```json
{
  "mcpServers": {
    "pr-analyzer": {
      "command": "/absolute/path/to/pr-analyzer-mcp",
      "env": {
        "GITHUB_TOKEN": "ghp_your_token_here"
      }
    }
  }
}
```

Or register it globally:

```bash
claude mcp add pr-analyzer /absolute/path/to/pr-analyzer-mcp \
  -e GITHUB_TOKEN=ghp_your_token_here \
  -e LOG_LEVEL=warn
```

Restart Claude and ask:

```
Analyze PR #42 in myorg/myrepo — give me a full migration plan.
```

---

### GitHub Copilot (VS Code)

1. Install the **GitHub Copilot Chat** extension (`ms-vscode.copilot-chat`) version 0.22+.
2. Open **VS Code Settings** (`Ctrl+,`) → search for `mcp` → click **Edit in settings.json**.
3. Add the server under `mcp.servers`:

```json
{
  "mcp": {
    "servers": {
      "pr-analyzer": {
        "type": "stdio",
        "command": "/absolute/path/to/pr-analyzer-mcp",
        "args": [],
        "env": {
          "GITHUB_TOKEN": "ghp_your_token_here",
          "BITBUCKET_TOKEN": "your_bitbucket_token",
          "LOG_LEVEL": "warn"
        }
      }
    }
  }
}
```

4. Open Copilot Chat (`Ctrl+Alt+I`) and select **Agent mode** (`@workspace`). The tools appear automatically.

```
@workspace use analyze_pr to review PR #15 in myorg/api-service
```

> **Tip:** You can also place `mcp.json` at the workspace root for project-scoped configuration:
> ```json
> {
>   "servers": {
>     "pr-analyzer": {
>       "type": "stdio",
>       "command": "${workspaceFolder}/pr-analyzer-mcp"
>     }
>   }
> }
> ```

---

### ChatGPT (OpenAI)

ChatGPT supports MCP through compatible desktop clients. Use **[OpenAI Desktop App](https://openai.com/chatgpt/desktop)** (macOS/Windows) version 1.2024.352+:

1. Open **Settings → Developer → MCP Servers → Add**.
2. Fill in the form:

| Field | Value |
|-------|-------|
| Name | `pr-analyzer` |
| Command | `/absolute/path/to/pr-analyzer-mcp` |
| Environment | `GITHUB_TOKEN=ghp_xxx` |

3. Save and start a new chat. The tools are available immediately.

**Alternative — Cursor IDE:**

Add to `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "pr-analyzer": {
      "command": "/absolute/path/to/pr-analyzer-mcp",
      "env": {
        "GITHUB_TOKEN": "ghp_your_token_here"
      }
    }
  }
}
```

**Alternative — Windsurf:**

Add to `~/.codeium/windsurf/mcp_config.json`:

```json
{
  "mcpServers": {
    "pr-analyzer": {
      "command": "/absolute/path/to/pr-analyzer-mcp",
      "env": {
        "GITHUB_TOKEN": "ghp_your_token_here"
      }
    }
  }
}
```

---

## Tool Reference

### `analyze_pr`

Full analysis of a pull request.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `platform` | string | yes | `github` or `bitbucket` |
| `repository` | string | yes | `owner/name` |
| `pr_number` | number | yes | PR number |

**Returns:** Executive summary, business purpose, technical purpose, changed files, dependencies, migration plan, architecture impact, DB impact, config impact, K8s impact, validation steps, rollback strategy.

---

### `explain_change`

Explains a single file change inside a PR.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `platform` | string | yes | `github` or `bitbucket` |
| `repository` | string | yes | `owner/name` |
| `pr_number` | number | yes | PR number |
| `file_path` | string | yes | Path of the file to explain |

---

### `generate_migration_plan`

Step-by-step plan to port PR changes to a target repository.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `platform` | string | yes | Source platform |
| `source_repository` | string | yes | Source `owner/name` |
| `pr_number` | number | yes | PR number |
| `target_repository` | string | yes | Target `owner/name` |

---

### `compare_repositories`

Diffs two repositories structure.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `source_platform` | string | yes | `github` or `bitbucket` |
| `source_repository` | string | yes | `owner/name` |
| `target_platform` | string | yes | `github` or `bitbucket` |
| `target_repository` | string | yes | `owner/name` |

---

All other tools (`dependency_analysis`, `architecture_map`, `find_related_files`, `find_required_dependencies`, `generate_migration_checklist`, `explain_code_flow`, `generate_feature_summary`, `generate_migration_documentation`) accept the same three common parameters: `platform`, `repository`, `pr_number`.

---

## Development

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -5

# Run a specific package
go test ./internal/adapters/mcp/... -v

# Benchmark
go test ./internal/application/usecase/... -bench=. -benchmem

# Lint (requires golangci-lint)
golangci-lint run ./...

# Build for all platforms
GOOS=linux  GOARCH=amd64 go build -o dist/pr-analyzer-mcp-linux-amd64  ./cmd/server
GOOS=darwin GOARCH=arm64 go build -o dist/pr-analyzer-mcp-darwin-arm64  ./cmd/server
GOOS=windows GOARCH=amd64 go build -o dist/pr-analyzer-mcp-windows.exe ./cmd/server
```

### Test Coverage by Package

| Package | Coverage |
|---------|----------|
| `internal/adapters/mcp` | 90.3% |
| `internal/service/analysis` | 97.0% |
| `internal/service/architecture` | 97.3% |
| `internal/service/migration` | 100% |
| `pkg/errors` | 100% |
| `pkg/logger` | 100% |
| `internal/domain/valueobject` | 100% |
| **Total** | **87.3%** |

---

## Token Permissions

### GitHub

Create a token at **Settings → Developer settings → Personal access tokens**:

| Permission | Reason |
|------------|--------|
| `repo` (read) | Read PRs, files, commits |
| `read:org` | Resolve org-scoped repositories |

A **fine-grained token** scoped to specific repositories is recommended.

### Bitbucket Cloud

Create an **App Password** at **Account settings → App passwords**:

| Permission | Reason |
|------------|--------|
| Repositories: Read | Read PRs and file content |
| Pull requests: Read | Read PR metadata, reviews, comments |

### Bitbucket Data Center

Use a **Personal Access Token** with `PROJECT_READ` + `REPO_READ` permissions.

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Commit with a clear message: `git commit -m "feat: add X"`
4. Push and open a pull request

All pull requests must pass `go test ./...` and maintain ≥ 80% coverage on changed packages.

---

## License

[MIT](LICENSE) © 2025 Vinay Charlie
