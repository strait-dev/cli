package ci

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// GenerateConfig holds template values for CI config generation.
type GenerateConfig struct {
	ProjectID   string
	Environment string
}

// Generate creates a CI workflow file for the given provider.
func Generate(provider string, cfg GenerateConfig) (string, error) {
	if containsUnsafeChars(cfg.ProjectID) || containsUnsafeChars(cfg.Environment) {
		return "", fmt.Errorf("unsafe characters in project ID or environment")
	}

	tplStr, ok := templates[provider]
	if !ok {
		return "", fmt.Errorf("unsupported CI provider: %s", provider)
	}

	tpl, err := template.New(provider).Parse(tplStr)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

func containsUnsafeChars(s string) bool {
	for _, ch := range []string{"${", "`", "{{", "}}", "$(", ";", "|", "&", "\n", "\r"} {
		if strings.Contains(s, ch) {
			return true
		}
	}
	return false
}

var templates = map[string]string{
	"github": `name: Strait Sync

on:
  push:
    branches: [main, master]

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Strait CLI
        run: |
          curl -fsSL https://get.strait.run | sh
          echo "$HOME/.strait/bin" >> $GITHUB_PATH

      - name: Sync orchestration definitions
        run: strait sync --file strait.json
        env:
          STRAIT_API_KEY: ${{"{{"}} secrets.STRAIT_API_KEY {{"}}"}}
          STRAIT_PROJECT: {{.ProjectID}}
          STRAIT_ENVIRONMENT: {{.Environment}}
`,
	"gitlab": `stages:
  - validate
  - sync

sync:
  stage: sync
  script:
    - curl -fsSL https://get.strait.run | sh
    - export PATH="$HOME/.strait/bin:$PATH"
    - strait sync --file strait.json
  only:
    - main
    - master
  variables:
    STRAIT_API_KEY: $STRAIT_API_KEY
    STRAIT_PROJECT: {{.ProjectID}}
    STRAIT_ENVIRONMENT: {{.Environment}}
`,
	"generic": `#!/bin/bash
set -euo pipefail

# Install Strait CLI
curl -fsSL https://get.strait.run | sh
export PATH="$HOME/.strait/bin:$PATH"

# Sync orchestration definitions
export STRAIT_API_KEY="${STRAIT_API_KEY}"
export STRAIT_PROJECT="{{.ProjectID}}"
export STRAIT_ENVIRONMENT="{{.Environment}}"
strait sync --file strait.json
`,
}
