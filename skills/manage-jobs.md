# Skill: Manage Jobs

Create, inspect, update, pause, and delete Strait jobs.

## When to use

Use this skill when you need to query or modify job definitions — not job runs.
For triggering runs, see the `trigger-run` skill.

## Prerequisites

- `STRAIT_API_KEY`, `STRAIT_PROJECT`, `STRAIT_SERVER` set (or equivalent flags)

## Common operations

### List all jobs

```
strait jobs list --format json
```

Returns a JSON array. Each object includes `id`, `slug`, `name`, `source_type`,
`active_deployment_id`, `paused`, and timestamps.

### Get a specific job

```
strait jobs get <job-id-or-slug> --format json
```

### Create a job

```
strait jobs create \
  --name "process-payments" \
  --endpoint https://api.example.com/process \
  --cron "0 * * * *" \
  --format json
```

Use `strait jobs create --help` for all available flags.

### Update a job

```
strait jobs update <job-id-or-slug> --cron "*/15 * * * *"
```

### Pause / resume a job

```
strait jobs pause <job-id-or-slug>
strait jobs resume <job-id-or-slug>
```

### Delete a job (requires `--yes` in non-interactive mode)

```
strait jobs delete <job-id-or-slug> --yes
```

## Field schema

```
strait schema job
```

Returns a JSON object describing all job fields, types, and enum values.

## Tips for agents

- Always use `--format json` to get machine-readable output.
- Use the `slug` field (short, URL-safe) in subsequent commands rather than the UUID `id`.
- `source_type` is `"code"` for code-first jobs and `"endpoint"` for webhook-based jobs.
- `active_deployment_id` is set only for code-first jobs with a successful build.
