<p align="center">
  <img src=".github/header.png" alt="Strait" width="100%" />
</p>

<p align="center">
  <strong>Command-line interface for the Strait orchestration platform</strong>
</p>

<p align="center">
  <a href="https://github.com/strait-dev/cli/actions/workflows/ci.yml"><img src="https://github.com/strait-dev/cli/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://github.com/strait-dev/cli/releases"><img src="https://img.shields.io/github/v/release/strait-dev/cli" alt="Release" /></a>
  <a href="https://github.com/strait-dev/cli/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License" /></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/strait-dev/cli"><img src="https://api.scorecard.dev/projects/github.com/strait-dev/cli/badge" alt="OpenSSF Scorecard" /></a>
  <img src="https://img.shields.io/badge/go-1.26.3-00ADD8?logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey" alt="Platform" />
</p>

<p align="center">
  <a href="https://strait.dev">Website</a>
  ·
  <a href="https://docs.strait.dev">Documentation</a>
  ·
  <a href="https://github.com/strait-dev/strait">Platform repo</a>
  ·
  <a href="https://github.com/strait-dev/cli/releases">Releases</a>
</p>

---

## What is Strait?

[Strait](https://strait.dev) is an open-source job orchestration platform. **Your code stays on your infrastructure** — Vercel, Cloudflare Workers, AWS Lambda, Netlify, Express, Kubernetes, or a long-lived Go worker. Strait schedules, retries, fans out, and observes — it does not host your code.

Two transports, one mental model:

- **`strait.serve`** — Strait calls a signed HTTPS endpoint you expose (push).
- **`strait.worker`** — your process holds a long-lived gRPC stream and pulls tasks (pull).

This repo ships the CLI: scaffold projects, register jobs, manage endpoints, run a local tunnel, inspect runs, and operate workers.

## Installation

```bash
# Homebrew (macOS / Linux)
brew install strait-dev/tap/strait

# From source (requires Go 1.26+)
go install github.com/strait-dev/cli/cmd/strait@latest
```

Pre-built binaries for darwin/linux/windows on [GitHub Releases](https://github.com/strait-dev/cli/releases).

## Quick start

```bash
# 1. Scaffold a starter project (pick any template — see `strait init --list`)
strait init --template vercel --name my-app

# 2. Authenticate with the orchestrator (opens a browser)
strait auth login

# 3. Sync strait.json orchestration definitions
cd my-app && strait sync

# 4. Point Strait at your deployed endpoint and round-trip a signed canary
strait endpoint set hello https://my-app.vercel.app/api/strait
strait endpoint verify hello

# 5. Watch a run end-to-end
strait runs watch <run-id>
```

### Run a worker instead

Workers are SDK programs, not CLI processes — scaffold one and run it on your own infrastructure:

```bash
strait init --template go-worker --name my-worker
cd my-worker && go run .          # opens a gRPC stream to the orchestrator
strait worker status              # confirm it's connected
```

### Local development with a tunnel

`strait dev` launches a Cloudflare Quick Tunnel, rewrites each job's `endpoint_url` to point at the tunnel, and restores the original URLs on exit. Drop into any scaffold:

```bash
strait dev
```

## Templates

`strait init --template <name>` scaffolds a working starter. Currently:

| Template       | Stack                              |
|----------------|------------------------------------|
| `vercel`       | Next.js App Router + `serve()`     |
| `cloudflare`   | Cloudflare Workers + `serve()`     |
| `lambda`       | AWS Lambda (Function URL) + `serve()` |
| `netlify`      | Netlify Functions + `serve()`      |
| `express`      | Node.js Express + `serve()`        |
| `go-chi-serve` | Go + chi router + `serve.Serve`    |
| `go-worker`    | Go worker holding a gRPC stream    |
| `k8s-worker`   | TypeScript worker for Kubernetes   |

Each scaffold ships a starter `strait.json` so `strait sync` works immediately.

Example `strait.json`:

```json
{
  "$schema": "https://schemas.strait.dev/v1/strait.json",
  "version": "1",
  "jobs": [
    {
      "slug": "hello",
      "name": "Hello",
      "endpoint_url": "https://my-app.example.com/api/strait",
      "max_attempts": 3,
      "timeout_secs": 30
    }
  ],
  "workflows": []
}
```

## Commands

`strait --help` shows the full tree. Canonical groups:

| Group           | Commands                                                              |
|-----------------|-----------------------------------------------------------------------|
| Scaffolding     | `init --template <name>`                                              |
| Migration       | `migrate inngest\|trigger\|hatchet --input <path>`                    |
| Sync            | `sync` (`strait.json` orchestration upsert; supports `--dry-run`, `--prune`) |
| Endpoint        | `endpoint set/get/verify`                                             |
| Worker          | `worker status/drain` (workers run on customer infra via `strait-go/worker`) |
| Dev             | `dev` (Cloudflare Tunnel + auto-register)                             |
| Jobs            | `jobs list/get/create/update/delete/clone/trigger/health/versions/dependencies/add-dependency/batch` |
| Job groups      | `job-groups list/get/create/update/delete/jobs/pause/resume/stats` |
| Runs            | `runs list/get/logs/cancel/replay/reschedule/dlq/dlq-replay/outputs/tool-calls/checkpoints/watch` |
| Workflows       | `workflows list/get/create/update/delete/clone/trigger/dry-run/plan/simulate/versions/diff/policy/visualize` |
| Workflow runs   | `workflow-runs list/get/pause/resume/retry/approve-step/retry-step/skip-step/force-complete-step`, `workflow-runs steps {list\|approve\|retry\|skip\|force-complete}` |
| Triggers        | `triggers list/get/send/stream/purge`                                 |
| Webhooks        | `webhooks list/get/create/delete/deliveries/retry/test`               |
| Event sources   | `event-sources list/get/create/update/delete`                         |
| Notifications   | `notifications list/get/create/update/delete`                          |
| Log drains      | `log-drains list/get/create/update/delete`                            |
| Logs            | `logs`                                                                |
| Secrets         | `secrets list/create/delete`, `api-keys list/create/rotate/revoke`    |
| Team            | `team list/add/remove/roles/policies/audit`                           |
| Projects / Env  | `projects list/switch/get/create/delete/export/import`, `env`/`environments list/get/create/update/delete/variables` |
| Analytics       | `analytics costs/reliability/top-failing/performance`                 |
| Billing         | `usage current/history/forecast`                                      |
| Auth            | `auth login/logout/whoami`, `context`, `alias`, `completion`, `config` |
| Dashboard       | `tui` (interactive jobs/runs/workflows pane switcher)                 |
| Diagnostics     | `debug bundle/profile/request`, `version`, `upgrade`                  |
| Extensions      | `extension list/install/run/create/remove`                            |

`strait deploy push` is deprecated and currently delegates to `strait sync` for compatibility. New scripts should use `strait sync`.

## Configuration

Strait CLI reads configuration from (in order of precedence):

1. Command-line flags (`--server`, `--project`, `--api-key`)
2. Environment variables (`STRAIT_SERVER`, `STRAIT_PROJECT`, `STRAIT_API_KEY`)
3. Per-project file: `./.strait.yaml`
4. User-global file: `~/.config/strait/config.yaml`

Credentials from `strait auth login` are stored in the OS keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager).

Every command supports `--format json|yaml|csv|table|jsonpath|go-template` for machine output and `--quiet` to suppress styling.

## Development

```bash
make build         # build binary to bin/strait
make test          # run tests with race detector
make lint          # golangci-lint
make check         # vet + lint + test
make hooks         # install lefthook pre-commit hooks
make mutation-dry  # run Gremlins coverage analysis without mutating code
make mutation      # run Gremlins mutation testing → bin/gremlins-report.json
```

Mutation testing is pinned to `go-gremlins/gremlins` `v0.6.0` via `go run`, so no separate install step is needed. Scope local runs with `MUTATION_ARGS`, e.g. `make mutation MUTATION_ARGS="--diff origin/main"`.

See [`AGENTS.md`](AGENTS.md) for the contributor operating guide.

## Reporting security issues

Email **security@strait.dev** — see [`SECURITY.md`](SECURITY.md).

## License

[MIT](LICENSE)
