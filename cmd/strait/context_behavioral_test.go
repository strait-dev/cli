package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cliconfig "github.com/strait-dev/cli/internal/config"
)

func TestContextCreate_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{
		opts:       &rootOptions{serverURL: "http://localhost:8080", outputFormat: "json", ciMode: true},
		configPath: configPath,
	}

	// Load the config so state.config is populated
	loaded, err := cliconfig.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	state.config = loaded.Data

	cmd := newContextCreateCommand(state)
	cmd.SetArgs([]string{"staging", "--server", "https://staging.example.com", "--project", "proj-staging"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "staging") {
		t.Fatalf("expected context name in output, got: %s", out)
	}

	// Verify config was written
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "staging") {
		t.Fatalf("expected staging context in config file, got: %s", string(data))
	}
}

func TestContextUse_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{
		opts:       &rootOptions{outputFormat: "json", ciMode: true},
		configPath: configPath,
		config:     &cliconfig.File{},
	}

	cmd := newContextUseCommand(state)
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected 'does not exist' error, got: %v", err)
	}
}

func TestContextList_Empty(t *testing.T) {
	t.Parallel()

	state := &appState{
		opts:   &rootOptions{outputFormat: "json", ciMode: true},
		config: &cliconfig.File{},
	}

	cmd := newContextListCommand(state)

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Empty contexts should produce empty JSON array
	if !strings.Contains(out, "[]") {
		t.Fatalf("expected empty array output, got: %s", out)
	}
}

func TestContextCurrent_NoActive(t *testing.T) {
	t.Parallel()

	state := &appState{
		opts:   &rootOptions{outputFormat: "json", ciMode: true},
		config: &cliconfig.File{},
	}

	cmd := newContextCurrentCommand(state)

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, `"active_context": ""`) {
		t.Fatalf("expected empty active_context, got: %s", out)
	}
}

func TestContextCurrent_WithActive(t *testing.T) {
	t.Parallel()

	state := &appState{
		opts: &rootOptions{outputFormat: "json", ciMode: true},
		config: &cliconfig.File{
			ActiveContext: "prod",
			Contexts: map[string]cliconfig.Context{
				"prod": {Server: "https://api.example.com", Project: "proj-1"},
			},
		},
	}

	cmd := newContextCurrentCommand(state)

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "prod") {
		t.Fatalf("expected prod in output, got: %s", out)
	}
	if !strings.Contains(out, "api.example.com") {
		t.Fatalf("expected server URL in output, got: %s", out)
	}
}
