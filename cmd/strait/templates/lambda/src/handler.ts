import type { APIGatewayProxyHandlerV2 } from "aws-lambda";
import { serve } from "@strait/ts/serve/lambda";
import { jobs } from "./jobs";

export const handler: APIGatewayProxyHandlerV2 = serve({
  jobs,
  signingSecret: process.env.STRAIT_SIGNING_SECRET!,
});
