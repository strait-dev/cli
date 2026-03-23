# Jobs

Jobs are the core unit of work in Strait. Each job has an endpoint URL that Strait calls when the job is triggered.

## List jobs

```bash
strait jobs list --project proj-1
strait jobs list -o json
```

## Create a job

Interactive wizard (TTY):

```bash
strait create job
```

Non-interactive:

```bash
strait create job \
  --name my-job \
  --slug my-job \
  --endpoint http://localhost:3000/jobs/my-job \
  --project proj-1 \
  --timeout-secs 60 \
  --max-attempts 3
```

Optional flags: `--cron`, `--description`, `--run-ttl-secs`.

## Get and describe

```bash
strait jobs get my-job              # Basic info
strait jobs describe my-job         # Rich details with recent runs
```

## Trigger

```bash
# Simple trigger
strait trigger my-job --payload '{"key": "value"}'

# From file
strait trigger my-job --payload-file input.json

# With options
strait trigger my-job \
  --payload '{"id": "123"}' \
  --idempotency-key "unique-key" \
  --priority 10 \
  --scheduled-at "2026-04-01T00:00:00Z"
```

`strait trigger` is a top-level shortcut for `strait jobs trigger`.

## Bulk trigger

```bash
strait jobs trigger-bulk my-job --items-json '[{"payload":{"id":"1"}},{"payload":{"id":"2"}}]'
strait jobs trigger-bulk my-job --items-file items.json
```

## Edit

```bash
strait jobs edit my-job --field "cron=*/10 * * * *"
```

Without `--field`, opens an interactive editor.

## Versions

```bash
strait jobs versions my-job
```

## Delete

```bash
strait jobs delete my-job --yes
```
