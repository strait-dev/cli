# Local Development

## Test job handlers locally

```bash
strait dev test process-payment --payload '{"id": "123"}'
strait dev test --all --config strait.config.json
```

## Tunnel

Expose a local server via Cloudflare tunnel so Strait can reach your endpoints:

```bash
strait dev tunnel --port 3000
```

## Dev status

```bash
strait dev status
```

## Run with context

Run a local command with Strait context environment variables injected:

```bash
strait run -- node worker.js
strait run --context staging -- python process.py
```
