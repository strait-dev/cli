# Event Triggers

Manage event-based triggers that fire jobs when specific events arrive.

## List triggers

```bash
strait triggers list --project proj-1
```

## Get a trigger

```bash
strait triggers get my-event-key
```

## Send an event

```bash
strait triggers send my-event-key --payload '{"data": "value"}'
```

## Purge old triggers

```bash
strait triggers purge --older-than 30 --dry-run
strait triggers purge --older-than 30
```
