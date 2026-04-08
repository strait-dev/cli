# Skill: Troubleshoot the CLI

Diagnose and resolve common Strait CLI issues.

## When to use

Use this skill when commands are failing with unclear errors, when the CLI
cannot connect to the server, or when authentication is not working.

## Step 1: Run doctor

```
strait doctor --format json
```

Runs all health checks in parallel and returns structured JSON. Look for any
check with `"status": "fail"` or `"status": "warn"`.

### Common failures and fixes

| Check | Failure cause | Fix |
|-------|--------------|-----|
| `api_key` | Missing API key | Set `STRAIT_API_KEY` or run `strait login` |
| `server_url` | Server not configured | Set `STRAIT_SERVER` |
| `connectivity_health` | Server unreachable | Verify server URL and network access |
| `connectivity_ready` | Server not ready | Check server database/redis dependencies |
| `auth_stats` | Invalid or expired API key | Run `strait login` or rotate API key |
| `tcp_connectivity` | Port blocked | Check firewall rules |
| `code_deploy_supported` | BuildKit not configured | Set `BUILDKIT_ADDRESS` on server |
| `runtime_detected` | No runtime marker | Add `go.mod`, `package.json`, etc. |

## Step 2: Check environment

```
strait doctor --verbose --format json
```

With `--verbose`, all env var checks are shown (not just failures).

## Step 3: Verify connectivity manually

```
strait health
strait health --ready
```

Returns server health status without authentication.

## Step 4: Check authentication

```
strait whoami --format json
```

Returns the authenticated user or project associated with the API key.

## Step 5: Check version compatibility

```
strait version --check-server --format json
```

## Exit codes

When a command fails, the exit code indicates the error category:

| Exit code | Meaning |
|-----------|---------|
| 1 | General or unexpected error |
| 2 | CLI panic |
| 3 | Configuration/usage error (check flags and env vars) |
| 4 | Authentication error (check `STRAIT_API_KEY`) |
| 5 | Resource not found (check IDs/slugs) |
| 6 | Conflict (resource already exists or state mismatch) |
| 7 | Validation error (invalid input) |
| 8 | Server error (check server logs) |

## Common errors

### "project ID is required"

Set `STRAIT_PROJECT` or pass `--project <id>`.

### "request failed (401): unauthorized"

Your API key is missing or invalid. Run `strait login` or set `STRAIT_API_KEY`.

### "interactive prompt blocked in non-interactive mode"

Pass `--yes` to skip confirmation prompts when running in CI or agent mode.

### "unsupported runtime"

Check `strait schema runtimes` for valid runtime names.

## Getting structured output

For agent use, always pass `--format json` to get machine-readable output on stdout.
Errors are written to stderr and do not affect stdout JSON output.
