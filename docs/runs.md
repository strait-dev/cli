# Runs

A run is a single execution of a job. Each run has a status, payload, and associated logs/events.

## List runs

```bash
strait runs list --project proj-1
strait runs list --status failed --limit 10
strait runs list -o json
```

## Get a run

```bash
strait runs get run_abc123
```

## Last run

```bash
strait runs last --project proj-1
strait runs last --open    # Open in browser
```

## Watch

Follow a run until it reaches a terminal state:

```bash
strait runs watch run_abc123
strait runs watch run_abc123 --timeout 5m
```

## Cancel

```bash
# Cancel a single run
strait runs cancel run_abc123

# Cancel all executing runs
strait runs cancel --all --status executing --project proj-1 --yes
```

## Replay

Re-execute a run using its original payload:

```bash
strait runs replay run_abc123
```

## Diff

Compare two runs side by side:

```bash
strait runs diff run_abc123 run_def456
strait runs diff run_abc123 run_def456 --show-events
```

## Wait

Block until a run reaches a condition:

```bash
strait wait run run_abc123 --for "status=completed" --timeout 5m
strait wait run run_abc123 --interval 5s
```
