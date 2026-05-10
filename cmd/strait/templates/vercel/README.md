# STRAIT_TEMPLATE_PROJECT_NAME

Strait + Vercel starter. Customer code runs on Vercel; Strait orchestrates job
execution by signing HTTPS calls to your `/api/strait` route.

## Setup

```bash
npm install
cp .env.example .env.local  # set STRAIT_SIGNING_SECRET
npm run dev
```

## Deploy

1. `vercel deploy` your app and copy the production URL.
2. `strait endpoint set hello https://<your-app>.vercel.app/api/strait`
3. `strait deploy push` to upsert the job definitions in `src/jobs.ts`.
4. `strait endpoint verify hello` to round-trip a signed canary payload.

## Files

- `src/jobs.ts` — job definitions (`defineJob`, `defineWorkflow`).
- `app/api/strait/route.ts` — Next.js App Router handler that mounts
  `serve({...})` and verifies the HMAC-SHA256 signature.
