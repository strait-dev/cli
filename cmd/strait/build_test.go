package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCommand_JSONEmitsSingleDocumentAndDoesNotWriteFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "strait.json")
	if err := os.WriteFile(configPath, []byte(`{"project":{"id":"proj-1"},"runtime":"node","jobs":[{"slug":"job-1","name":"Job 1"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{}, stdout: &bytes.Buffer{}}
	cmd := newBuildCommand(state)
	cmd.SetArgs([]string{"--config", configPath, "--json"})

	output := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("build --json: %v", err)
		}
	})

	var manifest map[string]any
	if err := json.Unmarshal([]byte(output), &manifest); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, output)
	}
	if strings.Count(strings.TrimSpace(output), "\n{") > 0 {
		t.Fatalf("expected a single JSON document, got:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(dir, ".strait", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("build --json should not write manifest.json, stat err=%v", err)
	}
}

func TestBuildCommand_DryRunDoesNotWriteFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "strait.config.yml")
	if err := os.WriteFile(configPath, []byte("project:\n  id: proj-1\nruntime: node\njobs:\n  - slug: job-1\n    name: Job 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{}, stdout: &bytes.Buffer{}}
	cmd := newBuildCommand(state)
	cmd.SetArgs([]string{"--config", configPath, "--dry-run"})

	output := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("build --dry-run: %v", err)
		}
	})

	if !strings.Contains(output, `"checksum"`) {
		t.Fatalf("expected manifest JSON output, got:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(dir, ".strait", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("build --dry-run should not write manifest.json, stat err=%v", err)
	}
}
