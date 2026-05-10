# internal/ — agent guide

## SDK migration policy

The CLI is moving from a hand-rolled HTTP client (`internal/client/`) to the
canonical Go SDK (`github.com/strait-dev/strait-go`). The two coexist while we
migrate.

**Rules of thumb:**

1. **New CLI commands** (anything added in or after STR-505) consume
   `internal/sdk/`, never `internal/client/` directly. The shim is the single
   point that flips when strait-go v0.2.0 lands.
2. **Existing commands** stay on `internal/client/` until they are individually
   migrated. Don't rewrite a working command just to switch transport.
3. **New API surface** (a method that doesn't exist on `*client.Client` yet)
   stays on `internal/client/` until strait-go ships it. Don't double-implement
   the same call in both places.

## What lives where

| Package              | Purpose                                                  |
|----------------------|----------------------------------------------------------|
| `internal/sdk/`      | Forward-looking shim. New commands use this.             |
| `internal/client/`   | Existing HTTP client. Keep stable; migrate, don't fork.  |
| `internal/config/`   | Config + context resolution (`Load`, `Resolve`).         |
| `internal/auth/`     | Keychain + device-code flow. Stays in CLI; not in SDK.   |
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

## When strait-go v0.2.0 lands

1. Add `github.com/strait-dev/strait-go vX.Y.Z` to `go.mod`.
2. Replace `Client = client.Client` in `internal/sdk/sdk.go` with the SDK
   client type and update `New(...)` to call `strait.NewClient(...)`.
3. Migrate one command at a time off `internal/client/` (start with the
   greenfield commands: `worker`, `endpoint`, `deploy push`).
4. Drop `internal/client/` methods as they are replaced; keep the package
   itself until every consumer is gone.
