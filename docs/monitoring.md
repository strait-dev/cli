# Monitoring and Diagnostics

## Health and status

```bash
strait doctor            # Comprehensive health check
strait status            # Project status overview
strait health            # Server health
```

## Real-time monitoring

```bash
strait listen --project proj-1                    # Watch for new runs
strait listen --project proj-1 --status failed    # Filter to failed runs
strait top                                        # Live queue depth monitoring
strait top queue                                  # Queue-specific stats
strait top jobs                                   # Job-specific stats
```

## Run analysis

```bash
strait trace run_abc123                           # ASCII timeline for a run
strait perf --project proj-1                      # Performance analytics
strait stats                                      # Queue statistics
```

## Waiting and draining

```bash
strait wait run run_abc123 --for "status=completed" --timeout 5m
strait wait queue --empty --timeout 10m
strait drain --timeout 5m                         # Wait for all executing runs to finish
```

## Cleanup

```bash
strait cleanup --project proj-1 --runs-older-than 720h --dry-run
strait cleanup --project proj-1 --runs-older-than 720h --status failed --yes
```

## Debugging

```bash
strait debug bundle run_abc123                    # Collect diagnostics into archive
strait debug bundle run_abc123 --no-events        # Exclude events from bundle
strait diagnose                                   # Run troubleshooting diagnostics
strait diagnose run run_abc123                    # Diagnose a specific run
strait profile --type cpu --duration 30s          # Capture pprof profile
strait profile --type heap --output heap.prof
```
