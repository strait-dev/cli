# Authentication

## Browser-based login (recommended)

```bash
strait login
```

The device code flow works like `gh auth login`: the CLI generates a code, opens your browser to the approval page, and polls until you confirm. The API key is stored in your system keyring.

## Direct token

```bash
# Paste an API key directly
strait login --token strait_abc123...

# Read from stdin (CI/CD)
echo $STRAIT_API_KEY | strait login --with-token
```

## Logout

```bash
strait logout
```

Removes the stored API key from your system keyring.

## Verify

```bash
strait whoami
```

Shows the authenticated user, active context, server, and project.

## Contexts

Manage multiple environments (production, staging, etc.):

```bash
# Create contexts
strait context create prod --server https://api.strait.dev --project proj-1
strait context create staging --server https://staging.strait.dev --project proj-2

# Switch between them
strait context use prod
strait context current
strait context list

# Delete a context
strait context delete staging
```

You can also override the context per-command:

```bash
strait jobs list --context staging
```

## Auth helpers

```bash
strait auth status    # Show current auth state
strait auth token     # Print current API token
```
