# Configuration

## Config files

Strait looks for configuration in this order:

1. `.strait.yaml` in the current directory (local project config)
2. `~/.config/strait/config.yaml` (global user config)

### Managing config

```bash
strait config path               # Print config file location
strait config edit               # Open config in $EDITOR
strait config edit --editor vim  # Open in a specific editor
```

## Environment variables

| Variable | Description |
|---|---|
| `STRAIT_SERVER` | API server URL |
| `STRAIT_API_KEY` | API key for authentication |
| `STRAIT_PROJECT` | Default project ID |
| `STRAIT_FORMAT` | Output format (table, json, yaml, csv) |
| `STRAIT_CONTEXT` | Active context name |
| `NO_COLOR` | Disable color output |

## Global flags

Every command accepts these flags:

```
--server          API server URL
--api-key         API key
--project         Project ID
-o, --format      Output format (table, json, yaml, csv, wide, go-template, jsonpath)
--no-color        Disable color output
--no-headers      Omit headers for table output
-q, --quiet       Minimal output
-v, --verbose     Verbose output
--context         Context name override
--config          Config file path
--timeout         Request timeout (default: 30s)
--ci              CI mode (no color, no interactive prompts)
```

## Output formats

All list/get commands support multiple output formats:

```bash
strait jobs list -o json
strait jobs list -o yaml
strait jobs list -o csv
strait jobs list -o wide
strait jobs list -o go-template --output-template '{{.ID}} {{.Status}}'
strait jobs list -o jsonpath --output-jsonpath '{.items[*].name}'
```

## Shell completion

```bash
strait completion bash > /etc/bash_completion.d/strait
strait completion zsh > "${fpath[1]}/_strait"
strait completion fish > ~/.config/fish/completions/strait.fish
```

## Command aliases

Create shortcuts for frequently used commands:

```bash
strait alias set trig "trigger"
strait alias set rl "runs list --status failed"
strait alias list
strait alias delete trig
```
