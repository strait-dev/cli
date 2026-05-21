# STRAIT_TEMPLATE_PROJECT_NAME

Strait + Express starter. Strait orchestrates job execution by calling your
Express app's signed `/api/strait` route.

## Setup

```bash
npm install
export STRAIT_SIGNING_SECRET=<your-secret>
npm run dev
```

## Deploy

1. Run `npm run build && npm start` on your host of choice (Render, Fly,
   Railway, EC2, etc.).
2. `strait endpoint set hello https://<your-host>/api/strait`
3. `strait sync` to sync the `strait.json` orchestration definitions.
4. `strait endpoint verify hello` to round-trip a signed canary payload.
