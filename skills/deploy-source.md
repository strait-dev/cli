# Skill: Deploy Source Code

Deploy source code directly to a Strait job using the code-first deployment pipeline.

## When to use

Use this skill when you need to build and deploy application code from a local directory
to a Strait job, triggering a server-side BuildKit build.

## Prerequisites

- `STRAIT_API_KEY` set (or `--api-key` flag)
- `STRAIT_PROJECT` set (or `--project` flag)
- `STRAIT_SERVER` set (or `--server` flag)
- A Strait job with a known slug
- Source code in a directory with a recognised runtime marker file

## Steps

### 1. Detect runtime (optional — auto-detected if omitted)

```
strait schema runtimes
```

Returns a JSON list of supported runtimes and their marker files.

### 2. Dry-run to verify what will be packed

```
strait deployments create-from-source --job <job-slug> --dry-run --dir <path>
```

Prints the list of files that would be included without uploading anything.
Use this to verify `.straitignore` is excluding unnecessary files.

### 3. Deploy

```
strait deployments create-from-source \
  --job <job-slug> \
  --dir <path> \
  --runtime <runtime>
```

Omit `--runtime` to auto-detect from the source directory.

> The legacy `strait deploy source` form continues to work but prints a
> deprecation warning. New code paths should use
> `strait deployments create-from-source`.

### 4. Watch the build (if you need to wait for completion)

```
strait deployments watch <deployment-id> --job <job-slug>
```

Streams build logs and exits 0 when the deployment is ready, or exits 1 on failure.

### 5. Rollback if needed

```
strait deployments rollback <deployment-id> --job <job-slug> --yes
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Deployment succeeded |
| 1 | General error |
| 3 | Missing required flag or invalid argument |
| 4 | Authentication error |
| 5 | Job not found |
| 7 | Validation error (e.g. unsupported runtime) |

## Example (non-interactive / agent mode)

```bash
strait deployments create-from-source \
  --job payment-processor \
  --dir ./services/payments \
  --runtime go \
  --no-stream \
  --format json
```

The `--format json` flag produces machine-readable output on stdout.
The `--no-stream` flag disables build log streaming (logs available via `strait deployments logs`).

## Checking server support

```
strait doctor
```

Look for `code_deploy_supported: pass` in the output.
