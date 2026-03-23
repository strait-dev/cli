# CI/CD Integration

## Setup

Auto-detect your CI provider and generate a workflow file:

```bash
strait ci setup
```

## Check

Validate CI readiness:

```bash
strait ci check
```

## Deploy in CI

Use `--ci` flag for non-interactive mode:

```bash
strait deploy --config strait.config.json --ci
```

## Environment variables for CI

Set these in your CI environment:

| Variable | Description |
|---|---|
| `STRAIT_API_KEY` | API key for authentication |
| `STRAIT_SERVER` | API server URL |
| `STRAIT_PROJECT` | Default project ID |

Authenticate via stdin in CI pipelines:

```bash
echo $STRAIT_API_KEY | strait login --with-token
```
