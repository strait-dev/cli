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

The official CLI for [Strait](https://strait.dev) — an open-source orchestration platform. Customer code runs on customer infrastructure (Vercel, Cloudflare Workers, AWS Lambda, Netlify, Express, Kubernetes, Go) and Strait orchestrates execution via signed HTTPS push (`strait.serve`) or a long-lived gRPC worker stream (`strait.worker`).

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
strait init --template vercel --name my-app   # Scaffold a starter project
strait auth login                              # Authenticate (opens browser)
strait deploy push                             # Upsert SDK-defined jobs
strait endpoint set hello https://my-app.vercel.app/api/strait
strait endpoint verify hello                   # Round-trip a signed canary
strait runs watch <run-id>                     # Stream a run's logs
```

To run a long-lived worker (gRPC) instead of a serve endpoint:

```bash
strait init --template go-worker --name my-worker
strait deploy push
strait worker start --queue default
```

## Commands

Canonical surface (orchestration-only). `strait --help` lists the full tree.

| Category | Commands |
|---|---|
| Scaffolding | `init --template <vercel\|cloudflare\|lambda\|netlify\|express\|k8s-worker\|go-chi-serve\|go-worker>` |
| Migration | `migrate inngest\|trigger\|hatchet --input <path>` |
| Deploy | `deploy push` |
| Endpoint | `endpoint set/get/verify` |
| Worker | `worker start/status/drain/logs` |
| Dev | `dev` (Cloudflare Tunnel + watch + auto-register) |
| Jobs | `jobs list/get/create/update/delete/clone/trigger/health/versions/dependencies/batch` |
| Runs | `runs list/get/logs/cancel/replay/reschedule/dlq-replay/outputs/checkpoints/events/watch` |
| Workflows | `workflows list/get/create/update/delete/clone/trigger/dry-run/plan/simulate/versions/diff/policy` |
| Workflow runs | `workflow-runs list/get/pause/resume/retry`, `workflow-runs steps {list\|approve\|retry\|skip\|force-complete}` |
| Triggers | `triggers list/get/send/stream/purge` |
| Webhooks | `webhooks list/get/create/delete/deliveries/retry/test` |
| Event sources | `event-sources list/get/create/update/delete` |
| Log drains | `log-drains list/get/create/update/delete` |
| Logs | `logs` |
| Secrets | `secrets list/create/delete`, `api-keys list/create/rotate/revoke` |
| Team | `team list/add/remove/roles/policies/audit` |
| Projects / Env | `projects list/switch/get/create/delete/export/import`, `env list/get/create/update/delete/variables` |
| Analytics | `analytics costs/reliability/top-failing/performance` |
| Billing | `usage current/history/forecast` |
| Auth | `auth login/logout/whoami`, `context`, `alias`, `completion`, `config` |
| Diagnostics | `debug bundle/profile`, `version`, `upgrade` |
| Extensions | `extension list/install/run/create/remove` |

Removed in this minor (managed-mode + non-canonical surfaces): `build`,
`verify`, `deployments`, `code_deploy`, `top`, `tui`, `agent`, `validate`,
`apply`, `diff`, `doctor`, `health`, `api`, `stats`, `perf`, `profile`,
`backup`, `fixtures`, `notifications`, `job-groups`, top-level
`send/listen/drain/events/trigger/login/logout/whoami/audit`. See
[CHANGELOG.md](CHANGELOG.md) for the full mapping.

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
