# Workflows

Workflows orchestrate multiple jobs as a directed acyclic graph (DAG). Steps can depend on other steps, enabling complex pipelines.

## List workflows

```bash
strait workflows list --project proj-1
```

## Create a workflow

Interactive wizard (TTY):

```bash
strait create workflow
```

## Describe

```bash
strait workflows describe data-pipeline
```

## Visualize

Render the DAG as ASCII art:

```bash
strait workflows visualize data-pipeline
strait workflows visualize data-pipeline --run wfr_abc123    # Show run status on nodes
```

## Trigger

```bash
strait workflows trigger data-pipeline --payload '{"date": "2026-03-21"}'
```

## Workflow runs

```bash
strait workflow-runs list --project proj-1
strait workflow-runs get wfr_abc123
strait workflow-runs steps wfr_abc123        # Show step-by-step status
strait workflow-runs cancel wfr_abc123
```
