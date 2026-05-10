import { defineJob } from "@strait/ts";

export const helloJob = defineJob({
  slug: "hello",
  run: async ({ payload }: { payload: { name?: string } }) => {
    const name = payload.name ?? "world";
    return { greeting: `hello, ${name}` };
  },
});

export const jobs = [helloJob];
