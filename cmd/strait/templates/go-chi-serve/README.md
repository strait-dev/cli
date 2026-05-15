# STRAIT_TEMPLATE_PROJECT_NAME

Strait + chi (Go) starter. Strait orchestrates job execution by calling your
chi router's signed `/api/strait` route.

## Setup

```bash
go mod tidy
export STRAIT_SIGNING_SECRET=<your-secret>
go run .
```

## Deploy

1. Build: `go build -o bin/server .` and deploy the binary or container to
   your host of choice.
2. `strait endpoint set hello https://<your-host>/api/strait`
3. `strait deploy push` to upsert the job definitions.
4. `strait endpoint verify hello` to round-trip a signed canary payload.
