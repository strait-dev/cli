import { worker } from "@strait/ts/worker";
import { jobs } from "./jobs";

const w = worker({
  apiKey: process.env.STRAIT_API_KEY!,
  serverUrl: process.env.STRAIT_SERVER ?? "https://api.strait.dev",
  queues: (process.env.STRAIT_QUEUES ?? "default").split(","),
  concurrency: Number(process.env.STRAIT_CONCURRENCY ?? 8),
  jobs,
});

w.run().catch((err) => {
  console.error("worker failed:", err);
  process.exit(1);
});
