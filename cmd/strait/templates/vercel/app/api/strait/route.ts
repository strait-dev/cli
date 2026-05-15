import { serve } from "@strait/ts/serve/vercel";
import { jobs } from "../../../src/jobs";

const handler = serve({
  jobs,
  signingSecret: process.env.STRAIT_SIGNING_SECRET!,
});

export const POST = handler;
export const runtime = "nodejs";
