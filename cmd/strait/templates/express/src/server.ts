import express from "express";
import { serve } from "@strait/ts/serve/express";
import { jobs } from "./jobs";

const app = express();

app.use(
  "/api/strait",
  serve({
    jobs,
    signingSecret: process.env.STRAIT_SIGNING_SECRET!,
  }),
);

const port = Number(process.env.PORT ?? 3000);
app.listen(port, () => {
  console.log(`listening on :${port}`);
});
