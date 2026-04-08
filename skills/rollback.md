# Skill: Rollback a Deployment

Roll back a code-first job to a previously successful deployment.

## When to use

Use this skill when a new code deployment is causing failures and you need to
revert the job to an earlier known-good image.

## Prerequisites

- `STRAIT_API_KEY`, `STRAIT_PROJECT`, `STRAIT_SERVER` set
- The job must use code-first deployments (`source_type: code`)
- The target deployment must have `status: ready`

## Steps

### 1. List available deployments for the job

```
strait deployments list --job <job-slug> --format json
```

Look for deployments with `"status": "ready"`. Note the `id` and `version` of
the deployment you want to roll back to.

### 2. Inspect the target deployment (optional)

```
strait deployments get <deployment-id> --job <job-slug> --format json
```

Verify the `built_image_uri` matches what you expect.

### 3. Roll back

```
strait deployments rollback <deployment-id> --job <job-slug> --yes
```

The `--yes` flag is required in non-interactive mode.

### 4. Verify

```
strait jobs get <job-slug> --format json | jq '.active_deployment_id'
```

The `active_deployment_id` should now match the deployment you rolled back to.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Rollback succeeded |
| 3 | Missing required flag |
| 4 | Authentication error |
| 5 | Deployment not found |
| 6 | Conflict (deployment not in ready state) |

## Field schema

```
strait schema deployment
```

## Notes

- Rollback does not rebuild the image — it switches the active pointer to an existing image.
- A rollback is instantaneous; existing in-flight runs are not affected.
- To undo a rollback, deploy again or roll back to a more recent deployment.
