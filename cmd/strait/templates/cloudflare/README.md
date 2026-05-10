# STRAIT_TEMPLATE_PROJECT_NAME

Strait + Cloudflare Workers starter. Strait orchestrates job execution by
calling your Worker's signed HTTPS endpoint.

## Setup

```bash
npm install
npx wrangler login
npx wrangler secret put STRAIT_SIGNING_SECRET
npm run dev
```

## Deploy

1. `npm run deploy` and copy the published Workers URL.
2. `strait endpoint set hello https://<your-worker>.workers.dev`
3. `strait deploy push` to upsert the job definitions in `src/jobs.ts`.
4. `strait endpoint verify hello` to round-trip a signed canary payload.
