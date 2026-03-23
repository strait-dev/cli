# Logs and Events

## Logs

View run output logs:

```bash
strait logs --run run_abc123
strait logs --follow --run run_abc123           # Stream logs in real time
strait logs --run run_abc123 --level error      # Filter by level
strait logs --run run_abc123 --search "timeout" # Search log content
strait logs --run run_abc123 --since 1h         # Time-based filter
strait logs --run run_abc123 --output ndjson | jq '.message'
```

You can also access logs through the runs subcommand:

```bash
strait runs logs run_abc123
```

## Events

Inspect structured events for a run:

```bash
strait events --run run_abc123
```

## Send

Send a raw event payload:

```bash
strait send --payload '{"type": "custom", "data": "value"}'
```
