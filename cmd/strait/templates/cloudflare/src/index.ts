import { serve } from "@strait/ts/serve/cloudflare";
import { jobs } from "./jobs";

interface Env {
  STRAIT_SIGNING_SECRET: string;
}

export default {
  fetch: (request: Request, env: Env) =>
    serve({
      jobs,
      signingSecret: env.STRAIT_SIGNING_SECRET,
    })(request),
};
