import { serve } from "@strait/ts/serve/netlify";
import { jobs } from "../../src/jobs";

export const handler = serve({
  jobs,
  signingSecret: process.env.STRAIT_SIGNING_SECRET!,
});
