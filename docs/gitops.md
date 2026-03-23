# Declarative GitOps

Manage infrastructure as code with declarative YAML definitions.

## Validate

Check definition files for syntax errors, required fields, cron expressions, and DAG acyclicity:

```bash
strait validate -f jobs.yaml
strait validate -f ./definitions/
```

## Check

Deep validation with endpoint reachability checks:

```bash
strait check -f jobs.yaml --check-endpoints
```

## Diff

Preview what would change on the server:

```bash
strait diff -f jobs.yaml
```

## Apply

Push definitions to the server:

```bash
strait apply -f jobs.yaml
strait apply -f ./definitions/ --dry-run
```

## Export

Export current server state as declarative YAML:

```bash
strait export all --project proj-1 --output-dir ./definitions/
strait export jobs --project proj-1 --name-contains "payment"
strait export workflows --project proj-1
```

## Build

Compile project config into a deployment manifest:

```bash
strait build --config strait.config.json --out-dir .strait
strait build --config strait.config.json --dry-run --json
```

## Project bundles

```bash
strait project build
strait project diff
strait project apply
```
