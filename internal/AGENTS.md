# internal/ — agent guide

## SDK migration policy

The CLI uses two transports side-by-side:

- `internal/sdk/` — wrapper around `github.com/strait-dev/strait-go` (the
  canonical Go SDK, pinned at v0.2.0). New commands target this.
- `internal/client/` — the original hand-rolled HTTP client. Existing commands
  stay on it until they are individually migrated.

**Rules of thumb:**

1. **New CLI commands** consume `internal/sdk/`. Construct with `sdk.New(resolved)`
   and reach the SDK services directly: `c.Jobs.Get(ctx, slug)`,
   `c.Workflows.List(ctx, ...)`, etc.
2. **Existing commands** stay on `internal/client/`. Don't rewrite a working
   command just to switch transport — that's its own per-command project.
3. **API surface the SDK doesn't expose** (worker admin: `ListWorkers`,
   `DisconnectWorker`; device-code auth in `cli_auth.go`) stays in
   `internal/client/` permanently. The SDK intentionally omits CLI-shaped
   helpers; don't try to push them upstream.
4. **Don't double-implement.** If a call exists on `*strait.Client`, use it via
   `internal/sdk/`. If it doesn't, leave it on `internal/client/`.

## What lives where

| Package              | Purpose                                                  |
|----------------------|----------------------------------------------------------|
| `internal/sdk/`      | Wrapper around `*strait.Client`. New commands use this.  |
| `internal/client/`   | Hand-rolled HTTP client. Hosts CLI-only flows permanently (`cli_auth.go`, worker admin) plus legacy callers pending migration. |
| `internal/config/`   | Config + context resolution (`Load`, `Resolve`).         |
| `internal/auth/`     | Keychain helpers. Stays in CLI; not in SDK.              |
| `internal/types/`    | JSON contract types mirroring the orchestration server.  |
| `internal/styles/`   | Terminal styling (TTY output).                           |
| `internal/output/`   | Multi-format renderer (json/yaml/template/jsonpath).     |
| `internal/validate/` | Slug, cron, endpoint validators.                         |
| `internal/tunnel/`   | Cloudflare Tunnel wrapper used by `strait dev`.          |
| `internal/wizard/`   | Interactive prompts for `jobs create` / `workflows create`. |
| `internal/bundle/`   | Diagnostic bundle producer for `strait debug bundle`.    |
| `internal/extension/`| Extension lifecycle for `strait extension`.              |
| `internal/dag/`      | DAG rendering for workflow visualizations.               |
| `internal/tui/`      | Shared TUI helpers.                                      |
| `internal/ci/`       | CI mode helpers (color, prompts).                        |

## Migrating a command from `internal/client/` to `internal/sdk/`

1. Confirm every method the command calls exists on `*strait.Client` (check
   `client.go` in `strait-go` — 23 services as of v0.2.0).
2. Swap `newAPIClient(state)` for `sdk.New(state.resolved)` and update the
   field/method names to match the SDK surface (`cli.GetJob(...)` →
   `c.Jobs.Get(...)`).
3. Update mocks: SDK services hit real HTTP, so existing `httptest`-backed
   behavioral tests usually work unchanged.
4. Leave the `internal/client/` method in place if anything else still calls it;
   delete it once the call graph is empty.

## Permanent CLI-only flows

These belong in `internal/client/` forever, not in the SDK:

- `cli_auth.go` — device-code flow at `/v1/cli/auth/device-code`. The SDK
  excludes interactive auth (it would couple SDK consumers to terminal I/O).
- `ListWorkers` / `DisconnectWorker` — admin operations on connected workers.
  The SDK exposes `worker.Run` (the runtime), not the admin surface.
