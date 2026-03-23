# Backup and Restore

## Create a backup

```bash
strait backup create --output backup.sql --database-url $DATABASE_URL
strait backup create --format custom --output backup.dump
```

Supported formats: `plain`, `custom`, `directory`, `tar`.

## Restore

```bash
strait backup restore -i backup.sql --database-url $DATABASE_URL --yes
strait backup restore -i backup.dump --clean --yes
```

The `--clean` flag drops database objects before restoring.
