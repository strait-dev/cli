# Skill: Initialize a Project

Set up a new Strait project configuration file and create starter files.

## When to use

Use this skill when setting up a new workspace that will use the Strait CLI.
It creates a `strait.json` (or `strait.yaml`) config file and a `.straitignore`
file appropriate for the detected runtime.

## Prerequisites

- `STRAIT_API_KEY` and `STRAIT_SERVER` set (for server-connected init)
- Or run with `--yes` and flags for fully offline scaffolding

## Non-interactive init (agent/CI mode)

```
strait init \
  --yes \
  --name my-project \
  --server https://your.strait.server \
  --runtime go
```

All values come from flags. No prompts are shown.

### Available flags

| Flag | Description |
|------|-------------|
| `--name` | Project name (lowercase, hyphens only) |
| `--server` | Strait server URL |
| `--runtime` | Language runtime for `.straitignore` scaffolding |
| `--yes` | Non-interactive mode |
| `--format json` | Print created files as JSON |

## What gets created

- `strait.json` — project config with server URL and project name
- `.straitignore` — runtime-appropriate ignore rules for source packing
- `.gitignore` entry for `strait.json` if the file is not already ignored

## Check created files

```
strait init --yes --name test-project --format json
```

The JSON output lists each file and its status (`created` or `skipped`).

## Validate the config

```
strait doctor
```

Runs all health checks including config validity, connectivity, and authentication.

## Supported runtimes

```
strait schema runtimes
```
