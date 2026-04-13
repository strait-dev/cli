package main

import (
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

	state := &appState{opts: &rootOptions{}}
	cmd := newBuildCommand(state)
	cmd.SetArgs([]string{"--config", configPath, "--json"})

	output := captureCommandOutput(t, func() {
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

	state := &appState{opts: &rootOptions{}}
	cmd := newBuildCommand(state)
	cmd.SetArgs([]string{"--config", configPath, "--dry-run"})

	output := captureCommandOutput(t, func() {
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

func TestBuildCommand_AutoDiscoversConfigAndWritesDefaultOutput(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "strait.json")
	if err := os.WriteFile(configPath, []byte(`{"project":{"id":"proj-1"},"runtime":"node","jobs":[{"slug":"job-1","name":"Job 1"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	state := &appState{opts: &rootOptions{outputFormat: "json", ciMode: true}}
	cmd := newBuildCommand(state)

	output := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("build autodiscovery: %v", err)
		}
	})

	manifestPath := filepath.Join(dir, ".strait", "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected manifest at %s: %v", manifestPath, err)
	}
	if !strings.Contains(output, ".strait/manifest.json") {
		t.Fatalf("expected output path in JSON output, got:\n%s", output)
	}
}

func TestBuildCommand_UsesConfigBuildOutDir(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "strait.config.yaml")
	if err := os.WriteFile(configPath, []byte("project:\n  id: proj-1\nbuild:\n  outDir: compiled\njobs:\n  - slug: job-1\n    name: Job 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	state := &appState{opts: &rootOptions{outputFormat: "json", ciMode: true}}
	cmd := newBuildCommand(state)
	cmd.SetArgs([]string{"--config", "strait.config.yaml"})

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("build with config outDir: %v", err)
		}
	})

	if _, err := os.Stat(filepath.Join(dir, "compiled", "manifest.json")); err != nil {
		t.Fatalf("expected manifest in config outDir: %v", err)
	}
}

func TestBuildCommand_OutDirFlagOverridesConfigBuildOutDir(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "strait.config.yaml")
	if err := os.WriteFile(configPath, []byte("project:\n  id: proj-1\nbuild:\n  outDir: compiled\njobs:\n  - slug: job-1\n    name: Job 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	state := &appState{opts: &rootOptions{outputFormat: "json", ciMode: true}}
	cmd := newBuildCommand(state)
	cmd.SetArgs([]string{"--config", "strait.config.yaml", "--out-dir", "custom-out"})

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("build with explicit outDir: %v", err)
		}
	})

	if _, err := os.Stat(filepath.Join(dir, "custom-out", "manifest.json")); err != nil {
		t.Fatalf("expected manifest in explicit outDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "compiled", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("did not expect manifest in config outDir when --out-dir overrides it, stat err=%v", err)
	}
}

func TestBuildCommand_ErrorsWhenNoConfigCanBeFound(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	state := &appState{opts: &rootOptions{}}
	cmd := newBuildCommand(state)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no config file found") {
		t.Fatalf("expected missing config error, got: %v", err)
	}
}
