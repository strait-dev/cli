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
  <a href="https://scorecard.dev/viewer/?uri=github.com/strait-dev/cli"><img src="https://api.scorecard.dev/projects/github.com/strait-dev/cli/badge" alt="OpenSSF Scorecard" /></a>
  <img src="https://img.shields.io/badge/go-1.26-00ADD8?logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey" alt="Platform" />
</p>

---

The official CLI for [Strait](https://strait.dev) -- an open-source job execution and workflow orchestration platform. A single Go binary with 55+ commands covering job management, workflow orchestration, deployment, declarative GitOps, local development, and real-time monitoring.

[Website](https://strait.dev) | [Platform Repo](https://github.com/strait-dev/strait) | [Documentation](https://docs.strait.dev) | [Releases](https://github.com/strait-dev/cli/releases)

## Installation

```bash
# Homebrew
brew install strait-dev/tap/strait

# From source
go install github.com/strait-dev/cli/cmd/strait@latest
```

Pre-built binaries available on [GitHub Releases](https://github.com/strait-dev/cli/releases).

## Quick start

```bash
strait init                                        # Initialize a project
strait login                                       # Authenticate (opens browser)
strait create job                                  # Create a job
strait trigger my-job --payload '{"id": "123"}'    # Trigger it
strait runs watch <run-id>                         # Watch the run
strait tui                                         # Open the dashboard
```

## Commands

| Category | Commands | Docs |
|---|---|---|
| Jobs | `create job`, `trigger`, `jobs list/get/describe/edit/delete` | [docs/jobs.md](docs/jobs.md) |
| Runs | `runs list/get/watch/cancel/replay/diff`, `wait run` | [docs/runs.md](docs/runs.md) |
| Workflows | `create workflow`, `workflows list/describe/visualize/trigger` | [docs/workflows.md](docs/workflows.md) |
| Deployment | `deploy`, `deploy create/finalize/promote/rollback/list`, `verify` | [docs/deployment.md](docs/deployment.md) |
| Logs | `logs`, `events`, `send` | [docs/logs-and-events.md](docs/logs-and-events.md) |
| GitOps | `validate`, `check`, `diff`, `apply`, `export`, `build`, `project` | [docs/gitops.md](docs/gitops.md) |
| Secrets | `secrets list/create/delete`, `api-keys list/create/rotate/revoke` | [docs/secrets-and-api-keys.md](docs/secrets-and-api-keys.md) |
| Team | `team list/add/remove/roles`, `audit` | [docs/team-and-audit.md](docs/team-and-audit.md) |
| Triggers | `triggers list/get/send/purge` | [docs/event-triggers.md](docs/event-triggers.md) |
| Monitoring | `doctor`, `status`, `health`, `listen`, `top`, `trace`, `perf`, `stats` | [docs/monitoring.md](docs/monitoring.md) |
| Local dev | `dev test`, `dev tunnel`, `dev status`, `run` | [docs/local-development.md](docs/local-development.md) |
| CI/CD | `ci setup`, `ci check` | [docs/ci-cd.md](docs/ci-cd.md) |
| Extensions | `extension list/install/run/create/remove` | [docs/extensions.md](docs/extensions.md) |
| Auth | `login`, `logout`, `whoami`, `context`, `auth` | [docs/authentication.md](docs/authentication.md) |
| Config | `config`, `alias`, `completion` | [docs/configuration.md](docs/configuration.md) |
| Backup | `backup create`, `backup restore` | [docs/backup.md](docs/backup.md) |
| Fixtures | `fixtures create/clean` | [docs/fixtures.md](docs/fixtures.md) |
| Raw API | `api GET/POST/DELETE ...` | [docs/raw-api.md](docs/raw-api.md) |
| Other | `open`, `cleanup`, `drain`, `diagnose`, `debug`, `profile`, `upgrade` | [docs/monitoring.md](docs/monitoring.md) |

## Development

```bash
make build         # Build binary to bin/strait
make test          # Run tests with race detector
make lint          # Run golangci-lint
make check         # vet + lint + test
make hooks         # Install lefthook pre-commit hooks
```

See [docs/architecture.md](docs/architecture.md) for project structure and design.

## License

MIT
