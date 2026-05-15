# STRAIT_TEMPLATE_PROJECT_NAME

Strait worker in Go. Maintains a long-lived gRPC stream to the orchestrator and
pulls tasks for the queues you subscribe to.

## Setup

```bash
go mod tidy
export STRAIT_API_KEY=<your-key>
export STRAIT_QUEUES=default
go run .
```

## Deploy

1. Build: `go build -o bin/worker .`
2. Run on your host (systemd, container, k8s Deployment, etc.) with the
   environment variables set.
3. `strait deploy push` to upsert your job definitions to the orchestrator.
4. `strait worker status` confirms your worker has connected.
