<p align="center">
  <img src=".github/header.png" alt="Strait" width="100%" />
</p>

<p align="center">
  <strong>Command-line interface for the Strait platform</strong>
</p>

<p align="center">
  <a href="https://github.com/strait-dev/cli/actions/workflows/ci.yml"><img src="https://github.com/strait-dev/cli/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://github.com/strait-dev/cli/releases"><img src="https://img.shields.io/github/v/release/strait-dev/cli" alt="Release" /></a>
  <a href="https://github.com/strait-dev/cli/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License" /></a>
  <img src="https://img.shields.io/badge/go-1.25-00ADD8?logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey" alt="Platform" />
</p>

---

The official CLI for [Strait](https://strait.dev) -- an open-source job execution and workflow orchestration platform for humans and AI agents. A single Go binary with 55+ commands covering job management, workflow orchestration, deployment, declarative GitOps, local development, and real-time monitoring.

[Website](https://strait.dev) | [Platform Repo](https://github.com/strait-dev/strait) | [Documentation](https://docs.strait.dev) | [Releases](https://github.com/strait-dev/cli/releases)

## Installation

### From source

```bash
go install github.com/strait-dev/cli/cmd/strait@latest
```

### Pre-built binaries

Download from [GitHub Releases](https://github.com/strait-dev/cli/releases).

### Homebrew (coming soon)

```bash
brew install strait-dev/tap/strait
```

## Quick Start

```bash
# Initialize a new project
strait init

# Authenticate (opens browser for OAuth)
strait login

# Create a job
strait create job

# Trigger it
strait trigger my-job --payload '{"id": "123"}'

# Watch the run
strait runs watch <run-id>

# Open the dashboard
strait tui
```

## Authentication

Strait CLI supports two authentication methods:

### Browser-based login (recommended)

```bash
strait login
# Opens browser -> approve -> CLI receives token automatically
```

The device code flow works like `gh auth login`: the CLI generates a code, opens your browser to the approval page, and polls until you confirm. The API key is stored in your system keyring.

### Direct token

```bash
# Paste an API key directly
strait login --token strait_abc123...

# Read from stdin (CI/CD)
echo $STRAIT_API_KEY | strait login --with-token
```

### Contexts

Manage multiple environments with contexts:

```bash
strait context create prod --server https://api.strait.dev --project proj-1
strait context create staging --server https://staging.strait.dev --project proj-2
strait context use prod
strait context current
```

## Configuration

### Config files

Strait looks for configuration in this order:
1. `.strait.yaml` in the current directory (local project config)
2. `~/.config/strait/config.yaml` (global user config)

### Environment variables

| Variable | Description |
|----------|-------------|
| `STRAIT_SERVER` | API server URL |
| `STRAIT_API_KEY` | API key for authentication |
| `STRAIT_PROJECT` | Default project ID |
| `STRAIT_FORMAT` | Output format (table, json, yaml, csv) |
| `STRAIT_CONTEXT` | Active context name |
| `NO_COLOR` | Disable color output |

### Global flags

Every command accepts these flags:

```
--server          API server URL
--api-key         API key
--project         Project ID
-o, --format      Output format (table, json, yaml, csv, wide, go-template, jsonpath)
--no-color        Disable color output
-q, --quiet       Minimal output
-v, --verbose     Verbose output
--context         Context name override
--config          Config file path
--timeout         Request timeout (default: 30s)
--ci              CI mode (no color, no interactive prompts)
```

## Commands

### Job Management

```bash
strait jobs list --project proj-1
strait create job --name my-job --endpoint http://localhost:3000/jobs/my-job
strait trigger my-job --payload '{"key": "value"}'
strait trigger my-job --payload-file input.json --wait
strait jobs trigger-bulk my-job --items-json '[{"payload":{"id":"1"}}]'
strait jobs get my-job
strait jobs describe my-job
strait jobs edit my-job --field "cron=*/10 * * * *"
strait jobs versions my-job
strait jobs delete my-job --yes
```

### Run Management

```bash
strait runs list --project proj-1 --status failed --limit 10
strait runs get run_abc123
strait runs last --project proj-1
strait runs watch run_abc123 --timeout 5m
strait runs cancel run_abc123
strait runs cancel --all --status executing --project proj-1 --yes
strait runs replay run_abc123
strait runs diff run_abc123 run_def456 --show-events
```

### Workflow Orchestration

```bash
strait workflows list --project proj-1
strait create workflow
strait workflows describe data-pipeline
strait workflows visualize data-pipeline --run wfr_abc123
strait workflows trigger data-pipeline --payload '{"date": "2026-03-21"}'
strait workflow-runs list --project proj-1
strait workflow-runs get wfr_abc123
strait workflow-runs steps wfr_abc123
strait workflow-runs cancel wfr_abc123
```

### Deployment

```bash
strait deploy --job my-job --dockerfile ./Dockerfile
strait deploy --config strait.config.json
strait deploy --config strait.config.json --strategy canary --canary-percent 10
strait deploy create --config strait.config.json --artifact-uri registry.example.com/app:v1.0
strait deploy finalize dep_abc123
strait deploy promote dep_abc123
strait deploy rollback --to dep_abc123
strait deploy list --project proj-1
strait deploy --config strait.config.json --dry-run
```

### Logs and Events

```bash
strait logs --run run_abc123
strait logs --follow --run run_abc123
strait logs --run run_abc123 --level error --search "timeout"
strait logs --run run_abc123 --since 1h
strait logs --run run_abc123 --output ndjson | jq '.message'
strait events --run run_abc123
```

### Secrets Management

```bash
strait secrets list --project proj-1
strait secrets create --project proj-1 --name STRIPE_KEY --value sk_live_xxx
strait secrets delete secret_abc123
strait secrets local set my-secret
strait secrets local get my-secret
```

### API Keys

```bash
strait api-keys list --project proj-1
strait api-keys create --project proj-1 --name "CI Deploy Key" --scopes "jobs:read,jobs:write"
strait api-keys rotate key_abc123
strait api-keys revoke key_abc123
```

### Team Management

```bash
strait team list --project proj-1
strait team add --user user_abc --role operator --project proj-1
strait team remove user_abc --project proj-1
strait team roles --project proj-1
```

### Event Triggers

```bash
strait triggers list --project proj-1
strait triggers get my-event-key
strait triggers send my-event-key --payload '{"data": "value"}'
strait triggers purge --older-than 30 --dry-run
```

### Declarative GitOps

Manage infrastructure as code with declarative YAML definitions:

```bash
# Validate definition files (syntax, required fields, cron expressions, DAG acyclicity)
strait validate -f jobs.yaml
strait validate -f ./definitions/

# Deep validation with endpoint reachability checks
strait check -f jobs.yaml --check-endpoints

# Preview what would change on the server
strait diff -f jobs.yaml

# Apply definitions to the server
strait apply -f jobs.yaml
strait apply -f ./definitions/ --dry-run

# Export current server state as declarative YAML
strait export all --project proj-1 --output-dir ./definitions/
strait export jobs --project proj-1 --name-contains "payment"
strait export workflows --project proj-1

# Compile project config into a deployment manifest
strait build --config strait.config.json --out-dir .strait
strait build --config strait.config.json --dry-run --json
```

### Command Aliases

```bash
strait alias set trig "trigger"
strait alias set rl "runs list --status failed"
strait alias list
strait alias delete trig
```

### Audit Log

```bash
strait audit --project proj-1
strait audit --project proj-1 --actor-id user_abc --limit 20
strait audit --project proj-1 --resource-type job --from 2026-03-01T00:00:00Z
```

### Raw API Access

```bash
strait api GET /v1/projects/proj-1/jobs
strait api POST /v1/projects/proj-1/jobs --field 'name=my-job' --field 'endpoint=http://example.com'
strait api DELETE /v1/projects/proj-1/jobs/my-job --header 'X-Custom:value'
```

### Account and Config

```bash
strait whoami                    # Show authenticated user, context, server, project
strait config path               # Print config file location
strait config edit               # Open config in $EDITOR
strait config edit --editor vim  # Open config in specific editor
strait open                      # Open dashboard in browser
strait open run_abc123           # Open specific resource in browser
```

## Local Development

```bash
# Test a job handler locally
strait dev test process-payment --payload '{"id": "123"}'
strait dev test --all --config strait.config.json

# Expose local server via Cloudflare tunnel
strait dev tunnel --port 3000

# Check local dev status
strait dev status

# Run a command with strait context env vars injected
strait run -- node worker.js
strait run --context staging -- python process.py
```

## Interactive Features

### TUI Dashboard

```bash
strait tui --project proj-1
```

Live terminal dashboard with queue metrics, run explorer, and event timeline. Navigate with keyboard shortcuts to browse jobs, runs, and events in real time.

### Interactive Wizards

When run without flags in a TTY, `strait init`, `strait create job`, and `strait create workflow` launch interactive wizards that walk through required fields with validation.

## CI/CD Integration

```bash
strait ci setup          # Auto-detect CI provider and generate workflow
strait ci check          # Validate CI readiness
strait deploy --ci       # Non-interactive deployment
```

## Monitoring and Diagnostics

```bash
# Health and status
strait doctor            # Comprehensive health check
strait status            # System status overview
strait health            # Server health

# Real-time monitoring
strait listen --project proj-1                    # Watch for new runs as they appear
strait listen --project proj-1 --status failed    # Filter to failed runs only
strait top                                        # Live queue depth monitoring
strait top queue                                  # Queue-specific stats
strait top jobs                                   # Job-specific stats

# Run analysis
strait trace run_abc123                           # ASCII timeline for a run
strait perf --project proj-1                      # Performance analytics

# Waiting and draining
strait wait run run_abc123 --for "status=completed" --timeout 5m
strait wait queue --empty --timeout 10m
strait drain --timeout 5m                         # Wait for all executing runs to finish

# Cleanup
strait cleanup --project proj-1 --runs-older-than 720h --dry-run
strait cleanup --project proj-1 --runs-older-than 720h --status failed --yes

# Debugging
strait debug bundle run_abc123                    # Collect diagnostics into shareable archive
strait debug bundle run_abc123 --no-events        # Exclude events from bundle
strait diagnose run_abc123                        # Run troubleshooting diagnostics
strait profile --type cpu --duration 30s          # Capture pprof profile from server
strait profile --type heap --output heap.prof
```

## Extensions

Extend the CLI with custom executables:

```bash
strait extension list                             # Discover extensions in PATH
strait extension install github.com/user/strait-myext
strait extension run myext --flag value
strait extension create my-extension              # Scaffold a new extension
strait extension remove myext
```

Extensions are executables named `strait-<name>` in your PATH, invoked as `strait extension run <name> [args...]`.

## Fixtures

Manage fixture data for demos and testing:

```bash
strait fixtures create --template full --project proj-1
strait fixtures create --template minimal
strait fixtures clean --project proj-1
```

## Backup and Restore

```bash
strait backup create --output backup.sql --database-url $DATABASE_URL
strait backup create --format custom --output backup.dump
strait backup restore -i backup.sql --database-url $DATABASE_URL --yes
strait backup restore -i backup.dump --clean --yes
```

## Self-Update

```bash
strait upgrade           # Check for new CLI version
strait upgrade --apply   # Download and install the latest version
strait version           # Print current version
```

## Output Formats

```bash
strait jobs list -o json
strait jobs list -o yaml
strait jobs list -o csv
strait jobs list -o wide
strait jobs list -o go-template --output-template '{{.ID}} {{.Status}}'
strait jobs list -o jsonpath --output-jsonpath '{.items[*].name}'
```

## Shell Completion

```bash
strait completion bash > /etc/bash_completion.d/strait
strait completion zsh > "${fpath[1]}/_strait"
strait completion fish > ~/.config/fish/completions/strait.fish
```

## Architecture

```
cmd/strait/              Command definitions (55+ commands)
internal/
  types/                 CLI-own types matching REST API JSON contract
  client/                HTTP API client (50+ methods)
  auth/                  Keyring credential storage + OAuth device flow
  config/                Config file management and context resolution
  styles/                Terminal color and formatting (lipgloss)
  output/                Multi-format output rendering
  dag/                   Workflow DAG visualization
  deploy/                Docker build/push and manifest deployment
  devtest/               Local job testing engine
  extension/             Plugin manifest, hooks, and lifecycle
  manifest/              Project config loading and compilation
  ci/                    CI provider detection and config generation
  tui/                   TUI dashboard components
  tunnel/                Cloudflare tunnel integration
  wizard/                Interactive form validation
  bundle/                GitOps bundle export/import/diff
```

Built with [Cobra](https://github.com/spf13/cobra), [charmbracelet/huh](https://github.com/charmbracelet/huh), [rivo/tview](https://github.com/rivo/tview), and [lipgloss](https://github.com/charmbracelet/lipgloss).

## Development

```bash
make build         # Build binary to bin/strait
make test          # Run tests with race detector
make lint          # Run golangci-lint
make install       # Install to $GOPATH/bin
make check         # vet + lint + test
```

## License

MIT
