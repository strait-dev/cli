# STRAIT_TEMPLATE_PROJECT_NAME

Strait + AWS Lambda starter. Strait orchestrates job execution by calling your
Lambda's signed HTTPS endpoint (Function URL or API Gateway).

## Setup

```bash
npm install
npm run build
```

## Deploy

1. `npm run package` produces `function.zip`. Upload via the AWS Console, SAM,
   or your IaC of choice.
2. Set the `STRAIT_SIGNING_SECRET` environment variable on the Lambda.
3. Configure a Function URL or API Gateway route to the handler.
4. `strait endpoint set hello https://<function-url>`
5. `strait deploy push` to upsert the job definitions in `src/jobs.ts`.
6. `strait endpoint verify hello` to round-trip a signed canary payload.
