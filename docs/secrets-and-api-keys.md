# Secrets and API Keys

## Secrets

Manage project-level secrets that are injected into job runs.

```bash
strait secrets list --project proj-1
strait secrets create --project proj-1 --name STRIPE_KEY --value sk_live_xxx
strait secrets delete secret_abc123
```

### Local secrets

Store secrets locally for development:

```bash
strait secrets local set my-secret
strait secrets local get my-secret
```

## API Keys

Manage API keys for programmatic access.

```bash
strait api-keys list --project proj-1
strait api-keys create --project proj-1 --name "CI Deploy Key" --scopes "jobs:read,jobs:write"
strait api-keys rotate key_abc123
strait api-keys revoke key_abc123
```
