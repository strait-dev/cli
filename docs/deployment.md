# Deployment

Deploy managed job images or full manifests with optional canary strategies.

## Single job deploy

```bash
strait deploy --job my-job --dockerfile ./Dockerfile
```

Options:

```
--dockerfile        Path to Dockerfile (default: ./Dockerfile)
--tag               Image tag (default: git SHA or 'latest')
--registry          Container registry (default: registry.fly.io)
--image             Pre-built image URI (skip build)
--region            Region override
--preset            Machine preset override
--env               Deployment environment (default: production)
--strategy          Deployment strategy: direct, canary (default: direct)
--canary-percent    Percentage of traffic for canary (1-99)
--canary-duration   Duration to run canary before full rollout (e.g. 10m, 1h)
--build-arg         Docker build args (repeatable)
--cache             Enable Docker layer caching (default: true)
--push              Push image after build (default: true)
--dry-run           Print plan without executing
```

## Manifest deploy

Deploy all jobs defined in a config file:

```bash
strait deploy --config strait.config.json
strait deploy --config strait.config.json --dry-run
```

## Canary deployments

```bash
strait deploy --config strait.config.json --strategy canary --canary-percent 10
```

## Draft deployments

```bash
strait deploy create --config strait.config.json --artifact-uri registry.example.com/app:v1.0
strait deploy finalize dep_abc123
```

## Promote and rollback

```bash
strait deploy promote dep_abc123
strait deploy rollback --to dep_abc123
```

## List deployments

```bash
strait deploy list --project proj-1
```

## Post-deploy verification

```bash
strait verify
```
