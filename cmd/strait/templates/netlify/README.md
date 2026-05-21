# STRAIT_TEMPLATE_PROJECT_NAME

Strait + Netlify Functions starter. Strait orchestrates job execution by
calling your function's signed HTTPS endpoint.

## Setup

```bash
npm install
netlify login
netlify env:set STRAIT_SIGNING_SECRET <your-secret>
npm run dev
```

## Deploy

1. `npm run deploy` and copy the production URL.
2. `strait endpoint set hello https://<your-site>.netlify.app/.netlify/functions/strait`
3. `strait sync` to sync the `strait.json` orchestration definitions.
4. `strait endpoint verify hello` to round-trip a signed canary payload.
