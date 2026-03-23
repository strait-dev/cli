# Raw API Access

Call the Strait REST API directly:

```bash
strait api GET /v1/projects/proj-1/jobs
strait api POST /v1/projects/proj-1/jobs --field 'name=my-job' --field 'endpoint=http://example.com'
strait api DELETE /v1/projects/proj-1/jobs/my-job --header 'X-Custom:value'
```

The `api` command uses your current authentication and server context automatically.
