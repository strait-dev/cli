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

The official CLI for [Strait](https://strait.dev) — an open-source job execution and workflow orchestration platform. A single Go binary with 55+ commands covering job management, workflow orchestration, deployment, declarative GitOps, local development, and real-time monitoring.

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
| Jobs | `create job`, `trigger`, `jobs list/get/describe/edit/delete` | [jobs](docs/cli-reference/jobs.mdx) |
| Runs | `runs list/get/watch/cancel/replay/diff`, `wait run` | [runs](docs/cli-reference/runs.mdx) |
| Workflows | `create workflow`, `workflows list/describe/visualize/trigger` | [workflows](docs/cli-reference/workflows.mdx) |
| Deployment | `deploy`, `deploy create/finalize/promote/rollback/list`, `verify` | [deployment](docs/guides/deployment.mdx) |
| Logs | `logs`, `events`, `send` | [logs](docs/cli-reference/logs.mdx) |
| GitOps | `validate`, `check`, `diff`, `apply`, `export`, `build`, `project` | [gitops](docs/guides/gitops.mdx) |
| Secrets | `secrets list/create/delete`, `api-keys list/create/rotate/revoke` | [secrets](docs/cli-reference/secrets.mdx) |
| Team | `team list/add/remove/roles`, `audit` | [team](docs/cli-reference/team.mdx) |
| Triggers | `triggers list/get/send/purge` | [triggers](docs/cli-reference/triggers.mdx) |
| Monitoring | `doctor`, `status`, `health`, `listen`, `top`, `trace`, `perf`, `stats` | [monitoring](docs/guides/monitoring.mdx) |
| Local dev | `dev test`, `dev tunnel`, `dev status`, `run` | [local dev](docs/guides/local-development.mdx) |
| CI/CD | `ci setup`, `ci check` | [ci-cd](docs/guides/ci-cd.mdx) |
| Extensions | `extension list/install/run/create/remove` | [extensions](docs/guides/extensions.mdx) |
| Auth | `login`, `logout`, `whoami`, `context`, `auth` | [authentication](docs/getting-started/authentication.mdx) |
| Config | `config`, `alias`, `completion` | [configuration](docs/guides/configuration.mdx) |
| Backup | `backup create`, `backup restore` | [backup](docs/cli-reference/backup.mdx) |
| Fixtures | `fixtures create/clean` | [fixtures](docs/cli-reference/fixtures.mdx) |
| Raw API | `api GET/POST/DELETE ...` | [api](docs/cli-reference/api.mdx) |
| Other | `open`, `cleanup`, `drain`, `diagnose`, `debug`, `profile`, `upgrade` | [utilities](docs/cli-reference/utilities.mdx) |

## Development

```bash
make build         # Build binary to bin/strait
make test          # Run tests with race detector
make lint          # Run golangci-lint
make mutation-dry  # Run Gremlins coverage analysis without mutating code
make mutation      # Run Gremlins mutation testing and write bin/gremlins-report.json
make check         # vet + lint + test
make hooks         # Install lefthook pre-commit hooks
```

Mutation testing is pinned to `go-gremlins/gremlins` `v0.6.0` and runs through `go run`, so contributors do not need a separate install step. You can scope local runs with `MUTATION_ARGS`, for example `make mutation MUTATION_ARGS="--diff origin/main"` to mutate only changes against `origin/main`.

## License

MIT
