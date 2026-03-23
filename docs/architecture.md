# Architecture

## Project structure

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

## Key libraries

- [Cobra](https://github.com/spf13/cobra) -- command framework
- [charmbracelet/huh](https://github.com/charmbracelet/huh) -- interactive forms
- [rivo/tview](https://github.com/rivo/tview) -- TUI dashboard
- [lipgloss](https://github.com/charmbracelet/lipgloss) -- terminal styling

## Design principles

- **Standalone REST client** -- no server, database, queue, or worker dependencies
- **CLI-own types** in `internal/types/` match the REST API JSON contract
- **TTY-aware** -- styled output to stderr, machine-readable output to stdout
- **Error wrapping** with `%w` and context throughout
