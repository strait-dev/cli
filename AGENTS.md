# AGENTS.md

This file is the operating guide for contributors and AI agents working on this repository.

**Read this document before making any change.**
If instructions conflict, use this priority order:
1. Direct user request
2. Repository conventions in this file
3. Existing code patterns
4. Personal preference

---

## 1) What this project is

- **Project**: Strait CLI
- **Language**: Go 1.26
- **Module**: `github.com/strait-dev/cli`
- **Purpose**: Standalone command-line interface for the Strait orchestration platform. Strait pivoted to **orchestration-only** — customer code runs on customer infrastructure (Vercel, Cloudflare Workers, AWS Lambda, Netlify, Express, Kubernetes, Go) and Strait orchestrates execution via signed HTTPS push (`strait.serve`) or a long-lived gRPC worker stream (`strait.worker`).
  - Canonical surface: jobs, runs, workflows, workflow-runs, triggers, webhooks, event-sources, log-drains, secrets, team, projects, env, analytics, usage, auth, debug, extension, init, migrate, deploy push, endpoint, worker, dev
  - REST API client (`internal/client/`) for existing commands; forward-looking SDK shim (`internal/sdk/`) for new commands. The shim becomes a thin wrapper around `strait-go` once v0.2.0 is published.
  - OAuth device code flow + keychain credential storage
  - Config file management (`~/.config/strait/config.yaml`, `.strait.yaml`)
  - Multi-format output (table, JSON, YAML, CSV, go-template, jsonpath)
  - Embedded starter templates for `strait init --template <name>` (8 runtimes)
  - Best-effort migration converters (`strait migrate inngest|trigger|hatchet`)
  - Extension/plugin system: `extension list/install/run/create/remove`

This repo was extracted from `strait-dev/strait` (monorepo `apps/strait/`) to ship CLI updates independently of the server.

Core technical model:
- The CLI defines its own response types in `internal/types/` matching the REST API JSON contract
- No shared Go module dependency between CLI and server
- The REST API is the single source of truth for type shapes

---

## 2) Repository map (how to navigate)

```
cmd/strait/              CLI commands and app entrypoint
cmd/strait/templates/    Embedded starter projects for `strait init --template`
internal/
  types/                 CLI-own types matching REST API JSON contract
  client/                HTTP API client (existing commands)
  sdk/                   Forward-looking shim around strait-go (new commands)
  auth/                  Keyring credential storage + OAuth device flow
  config/                Config file management and context resolution
  styles/                Terminal color and formatting (lipgloss)
  output/                Multi-format output rendering
  dag/                   Workflow DAG visualization (box-drawing)
  extension/             Plugin manifest, hooks, and lifecycle
  ci/                    CI provider detection and config generation
  tui/                   TUI dashboard components
  tunnel/                Cloudflare tunnel integration
  wizard/                Interactive form validation
  bundle/                GitOps bundle export/import/diff
```

See `internal/AGENTS.md` for the `internal/client/` vs `internal/sdk/` migration policy.

Support files:
- `.github/workflows/ci.yml` — CI (test + lint + goreleaser check)
- `.golangci.yml` — linter config
- `.goreleaser.yaml` — cross-platform binary releases
- `Makefile` — build/test/lint targets

---

## 3) Engineering rules (non-negotiable)

1. **No server dependencies**
   - This is a pure REST API client. Never import server-internal packages (store, queue, worker, api, etc.).
   - Types come from `internal/types/`, not from the server's `domain` package.

2. **Error handling**
   - Wrap with `%w` and include contextual message.

3. **Collection helpers**
   - Prefer `samber/lo` where it improves readability.

4. **Testing style**
   - Table-driven tests with `t.Parallel()`.
   - Use `httptest.NewServer` for API client tests.
   - Every new functionality needs tests. Every bug fix needs a regression test.

5. **No emojis**
   - In code, comments, logs, docs, commits, PR text.

6. **Output discipline**
   - TTY: rich styled output to stderr via `styles` package.
   - Non-TTY/piped: machine-readable JSON to stdout via `printData(state, ...)` or `state.out()`. Never use `os.Stdout`, `fmt.Print`/`Println`/`Printf`, or the `print`/`println` builtins for primary output — `TestRunEHandlersDoNotBypassPrintData` will fail the build.
   - If a bypass is genuinely necessary (e.g. subprocess stdout passthrough, top-level fatal handler that runs without `appState`), suppress the audit on that line with `// printdata-ok: <reason>`. The reason must explain why routing through `state.out()` is impossible, not why it would be inconvenient.
   - All commands must support `--format json` and `--quiet` modes.

---

## 4) Validation commands (before proposing merge)

```bash
go build ./...
go test ./...
go test -race ./...
golangci-lint run --timeout=5m ./...
```

Or via Makefile:
```bash
make check    # vet + lint + test
make build    # build binary to bin/strait
make install  # install to $GOPATH/bin
```

---

## 5) Commit and PR conventions

### 5.1 Conventional Commits (mandatory)

Every commit must follow Conventional Commits. No exceptions.

```text
type(scope): short summary
```

Examples:
- `feat(client): add bulk cancel API method`
- `fix(auth): handle expired keychain tokens`
- `docs(cli): clarify runs watch output modes`
- `test(client): add regression for pagination edge case`

Allowed types:
- `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `build`, `ci`, `chore`, `revert`

Rules:
1. lowercase type/scope
2. imperative summary
3. include scope when useful
4. use `!` for breaking changes and explain in body
5. avoid vague messages (`update`, `misc`, `fix stuff`)
6. **never** add "Co-Authored-By" lines to commit messages
7. **never** add AI attribution of any kind to commit messages or PR descriptions

### 5.2 PR expectations

PR descriptions must be clear, complete, and easy to review.

Every PR description should include:
- **Summary**: what this PR does
- **Why**: context/problem and motivation
- **What changed**: key implementation points
- **Validation**: exact commands run + outcomes
- **Testing impact**: what tests were added/updated

---

## 6) Workflow for implementing changes

1. Understand request and constraints.
2. Read relevant code paths.
3. If ambiguous, ask user and wait.
4. Share a concise implementation plan for non-trivial work.
5. Implement minimal targeted change.
6. Add/update tests (mandatory for new functionality and bug fixes).
7. Run validations and report results.

Keep scope narrow: one logical change per PR.

---

## 7) DOs and DON'Ts

### DO
- Do confirm assumptions when requirements are ambiguous.
- Do follow existing package boundaries and naming patterns.
- Do keep changes small, focused, and reversible.
- Do add tests for every new functionality.
- Do include regression tests for bug fixes.
- Do maintain backward compatibility unless user requests breakage.
- Do update README for user-facing changes.

### DON'T
- Don't guess API contracts or type shapes — check the server repo.
- Don't make unrelated refactors in the same PR.
- Don't ship new behavior without tests.
- Don't bypass failing tests/lint without explicit user approval.
- Don't import server-internal packages.
- Don't mark work as complete without validation evidence.

---

## 8) Testing quality bar (mandatory)

- Every behavior change should be protected by tests.
- New functionality should include meaningful assertions (not only "no error").
- For critical paths (client, auth, config), prefer multiple tests covering:
  - success path
  - validation/error path
  - edge cases
- When fixing defects, add a regression test that would fail before the fix.

---

## 9) Definition of done

A change is done only when all apply:

1. Code compiles (`go build ./...`)
2. All tests pass (`go test ./...`)
3. Race detector passes (`go test -race ./...`)
4. Lint passes (`golangci-lint run ./...`)
5. README/docs updated for user-facing behavior changes
6. Summary provided: what changed, why, and how validated

---

When in doubt, prefer established project patterns over novelty, ask clarifying questions early, and keep changes explicit and verifiable.
