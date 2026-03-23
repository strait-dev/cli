# Strait CLI

The standalone command-line interface for the [Strait](https://strait.dev) job orchestration platform. A single Go binary with 50+ commands covering job management, workflow orchestration, deployment, local development, and real-time monitoring.

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

## Local Development

```bash
strait dev test process-payment --payload '{"id": "123"}'
strait dev test --all --config strait.config.json
strait dev tunnel --port 3000
strait dev status
```

## Interactive Features

### TUI Dashboard

```bash
strait tui --project proj-1
```

Live terminal dashboard with queue metrics, run explorer, and event timeline.

### Interactive Wizards

When run without flags in a TTY, `strait init`, `strait create job`, and `strait create workflow` launch interactive wizards.

## CI/CD Integration

```bash
strait ci setup          # Auto-detect CI provider and generate workflow
strait ci check          # Validate CI readiness
strait deploy --ci       # Non-interactive deployment
```

## Diagnostics

```bash
strait doctor            # Comprehensive health check
strait status            # System status overview
strait health            # Server health
strait trace run_abc123  # Trace run execution
strait perf --project p  # Performance analytics
strait top               # Queue depth monitoring
```

## Output Formats

```bash
strait jobs list -o json
strait jobs list -o yaml
strait jobs list -o csv
strait jobs list -o go-template --output-template '{{.ID}} {{.Status}}'
```

## Shell Completion

```bash
strait completion bash > /etc/bash_completion.d/strait
strait completion zsh > "${fpath[1]}/_strait"
strait completion fish > ~/.config/fish/completions/strait.fish
```

## Architecture

```
cmd/strait/              Command definitions (50+ commands)
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

See [LICENSE](LICENSE) for details.
