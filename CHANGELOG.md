# Changelog

All notable changes to the Strait CLI are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Orchestration-only refactor (STR-505)

Strait pivoted from a managed-compute platform to **orchestration-only**.
Customer code now runs on customer infrastructure (Vercel, Cloudflare Workers,
AWS Lambda, Netlify, Express, Kubernetes, Go) and Strait orchestrates execution
via signed HTTPS push (`strait.serve`) or a long-lived gRPC worker stream
(`strait.worker`). The CLI surface was rewritten to match.

### Added

- `strait init --template <name>` — template-driven scaffolder backed by
  embedded starter projects. Templates: `vercel`, `cloudflare`, `lambda`,
  `netlify`, `express`, `k8s-worker`, `go-chi-serve`, `go-worker`. Use
  `strait init --list` to see the full set.
- `strait migrate inngest|trigger|hatchet` — best-effort converter that turns
  an Inngest/Trigger.dev/Hatchet export into Strait `defineJob` TypeScript
  sources plus a `strait.deploy.json` manifest. Conversion notes are surfaced
  inline as `// TODO: review` comments.
- `internal/sdk/` — forward-looking shim that new commands consume in place of
  `internal/client/`. The shim becomes a thin wrapper around `strait-go` once
  v0.2.0 is published; existing commands continue to use `internal/client/`
  until they are individually migrated. See `internal/AGENTS.md`.
- `runs events` — moved here from the deleted top-level `events` command.
- `triggers stream` — replaces top-level `listen`.
- `triggers send <key>` now accepts `--raw` to dispatch through `/v1/events`
  in addition to the typed event-trigger endpoint.
- `triggers stream` long-poll adapter prints newly-arrived event triggers as
  JSON lines (replaces top-level `listen`).
- `debug profile` runs round-trip probes against the configured server and
  pairs the timings with the server's performance analytics snapshot.
- `analytics performance` — replaces `perf`/`profile`.
- `team audit` — replaces top-level `audit`.
- `workflow-runs steps {list|approve|retry|skip|force-complete}` — collapses
  the previous flat `approve-step`/`retry-step`/`skip-step`/`force-complete-step`
  set under a single subgroup.
- `debug bundle` / `debug profile` — diagnostics consolidation.
- `strait tui` — interactive dashboard (k9s-style) with read-only panes for
  jobs, runs, workflows, and workflow runs. Tab/arrows switch panes; `r`
  refreshes the active pane; `?` shows keybindings. Write actions remain in
  the regular CLI surface; the TUI is the single GUI surface for the CLI.

### Removed

#### Managed-mode artifacts (STR-516)

- Top-level `deploy` (Docker build+push), `build`, `verify`, `deployments`,
  `code_deploy`.
- Internal packages: `internal/deploy/`, `internal/codedeploy/`,
  `internal/pack/`, `internal/manifest/`, `internal/devtest/`.
- Types: `MachinePreset`, `DeploymentStrategy*`, `DeploymentVersionStatus*`,
  `ExecutionModeManaged`, plus `Job.MachinePreset`/`ImageURI`/`Region`/
  `PreferredRegions`/`MachineID` and the `JobRun` mirrors.
- Client methods: `CreateDeploymentVersion`, `FinalizeDeployment`,
  `PromoteDeployment`, `RollbackDeployment`, `ListDeployments`,
  `CreateCodeDeployment`, `ConfirmCodeDeployment`, `GetCodeDeployment`,
  `ListCodeDeployments`, `RollbackCodeDeployment`, `GetServerCapabilities`.

#### Non-canonical commands (STR-522)

Hard-cut commands that produce a styled migration error pointing at the
canonical replacement:

| Removed | Replacement |
|---|---|
| `deploy` (Docker form) | `strait deploy push` (SDK-defined jobs) |
| `build` | `strait deploy push` |
| `verify` | `strait endpoint verify <slug>` |
| `deployments` | `strait deploy push` |
| `dev` (server-stack form) | `strait dev` (orchestration mode) |
| `init` (interactive wizard) | `strait init --template <name>` |
| `top` / `tui` / `agent` | use the dashboard |
| `validate` / `apply` / `diff` | `strait workflows dry-run` / `strait jobs update --dry-run` |
| `doctor` / `diagnose` / `check` / `status` / `health` / `api` | `strait debug bundle` |
| `stats` / `perf` / `profile` | `strait analytics performance` |
| `send` | `strait triggers send <key>` |
| `listen` | `strait triggers stream` |
| `drain` | `strait worker drain` |
| `events` | `strait runs events` |
| `trigger` (singular) | `strait jobs trigger` |
| `whoami` / `login` / `logout` | `strait auth whoami/login/logout` |
| `audit` | `strait team audit` |
| `schema` / `backup` / `cleanup` / `trace` / `ci` / `open` / `run` / `create` / `docs` / `update` / `fixtures` / `job-groups` / `notifications` | not in canonical surface |

#### Reshaped command tree

- `project` → `projects` with canonical `list/switch/get/create/delete` plus
  the existing `export`/`import`.
- `environments` → `env`. The `environment`/`envs`/`env` aliases were dropped.
- `auth status` → `auth whoami`.
- `jobs` dropped `describe`, `edit`, `add-dependency`; renamed `bulk-trigger`
  → `batch`. Final: `list/get/create/update/delete/clone/trigger/health/versions/dependencies/batch`.
- `runs` dropped `last`, `diff`, `tool-calls`, `usage`. Added `events`.
- `workflows` dropped `describe`, `runs`, `visualize`.
- `secrets` dropped `local`. Final: `list/create/delete` (server-side only).
- `extension` set is unchanged: `list/install/run/create/remove`.

### Migration guide

#### Managed compute → orchestration

If you were using `strait deploy` to build and push a Docker image, that
workflow is gone. Instead:

1. Adopt the Strait SDK in your codebase: `npm install @strait/ts` or
   `go get github.com/strait-dev/strait-go`.
2. Define your jobs with `defineJob({...})` (TypeScript) or `strait.DefineJob`
   (Go). See `strait init --template <name>` for working scaffolds.
3. Deploy your code to the platform of your choice (Vercel, Cloudflare,
   Lambda, your own Kubernetes cluster, etc.).
4. Run `strait deploy push` to upsert your SDK-defined jobs into the
   orchestrator.
5. Run `strait endpoint set <job-slug> <url>` and `strait endpoint verify
   <job-slug>` to wire up the signed-push integration.

#### Common command renames

```
strait login              → strait auth login
strait whoami             → strait auth whoami
strait events             → strait runs events
strait listen             → strait triggers stream
strait send <event>       → strait triggers send <event>
strait drain              → strait worker drain
strait perf               → strait analytics performance
strait audit              → strait team audit
strait workflow-runs approve-step → strait workflow-runs steps approve
```

## v0.1.0

Initial public release of the managed-compute CLI. Superseded by the
orchestration-only refactor in this minor.
