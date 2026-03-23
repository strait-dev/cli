# Team and Audit

## Team management

```bash
strait team list --project proj-1
strait team add --user user_abc --role operator --project proj-1
strait team remove user_abc --project proj-1
strait team roles --project proj-1
```

## Audit log

View a history of actions taken in your project:

```bash
strait audit --project proj-1
strait audit --project proj-1 --actor-id user_abc --limit 20
strait audit --project proj-1 --resource-type job --from 2026-03-01T00:00:00Z
```
