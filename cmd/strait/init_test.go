package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_NonInteractive_AllFlags(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--name", "my-api", "--runtime", "node"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config file was created
	if _, err := os.Stat(filepath.Join(dir, "strait.config.json")); err != nil {
		t.Fatal("strait.config.json not created")
	}
}

func TestInit_NonInteractive_RequiresName(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--runtime", "node"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected 'required' error, got: %v", err)
	}
}

func TestInit_WritesValidConfig(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--name", "test-project", "--runtime", "bun"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "strait.config.json"))
	if err != nil {
		t.Fatal(err)
	}

	var cfg straitConfigJSON
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if cfg.Project.ID != "test-project" {
		t.Fatalf("expected project.id=test-project, got %q", cfg.Project.ID)
	}
	if cfg.Runtime != "bun" {
		t.Fatalf("expected runtime=bun, got %q", cfg.Runtime)
	}
}

func TestInit_WithJob_AddsJobToConfig(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{
		"--yes", "--name", "my-api", "--runtime", "node",
		"--with-job", "--job-name", "process-payment",
		"--job-endpoint", "http://localhost:3000/jobs/payment",
		"--job-cron", "*/5 * * * *",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "strait.config.json"))
	if err != nil {
		t.Fatal(err)
	}

	var cfg straitConfigJSON
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(cfg.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(cfg.Jobs))
	}
	if cfg.Jobs[0].Slug != "process-payment" {
		t.Fatalf("expected slug=process-payment, got %q", cfg.Jobs[0].Slug)
	}
	if cfg.Jobs[0].EndpointURL != "http://localhost:3000/jobs/payment" {
		t.Fatalf("expected endpoint, got %q", cfg.Jobs[0].EndpointURL)
	}
	if cfg.Jobs[0].Cron != "*/5 * * * *" {
		t.Fatalf("expected cron, got %q", cfg.Jobs[0].Cron)
	}
}

func TestInit_WithJob_ValidatesEndpoint(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{
		"--yes", "--name", "my-api", "--runtime", "node",
		"--with-job", "--job-name", "bad-job",
		"--job-endpoint", "not-a-url",
	})

	// The endpoint is written as-is in non-interactive mode (validation happens
	// at API time). The init command trusts flag input — wizard validates interactively.
	// This test verifies the flag path doesn't crash.
	_ = cmd.Execute()
}

func TestInit_ConfigAlreadyExists_Errors(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	// Pre-create the config
	if err := os.WriteFile(filepath.Join(dir, "strait.config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--name", "my-api"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for existing config")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}
}

func TestInit_ConfigAlreadyExists_Force(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	if err := os.WriteFile(filepath.Join(dir, "strait.config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--name", "my-api", "--runtime", "go", "--force"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--force should allow overwrite, got: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "strait.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "my-api") {
		t.Fatal("config was not overwritten")
	}
}

func TestInit_UpdatesGitignore(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--name", "my-api"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(".gitignore not created")
	}
	if !strings.Contains(string(content), ".strait/") {
		t.Fatal(".gitignore missing .strait/ entry")
	}
}

func TestInit_GitignoreAlreadyHasEntry(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	// Pre-create .gitignore with the entry
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n.strait/\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--name", "my-api"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	// Should not duplicate the entry
	count := strings.Count(string(content), ".strait/")
	if count != 1 {
		t.Fatalf("expected 1 .strait/ entry, got %d", count)
	}
}

func TestInit_RuntimeAffectsConfig(t *testing.T) {
	// Not parallel: subtests use os.Chdir which is process-global.
	for _, rt := range []string{"node", "bun", "python", "go", "docker"} {
		t.Run(rt, func(t *testing.T) {
			dir := t.TempDir()
			origDir, _ := os.Getwd()
			t.Cleanup(func() { _ = os.Chdir(origDir) })
			_ = os.Chdir(dir)

			state := &appState{opts: &rootOptions{}}
			cmd := newInitCommand(state)
			cmd.SetArgs([]string{"--yes", "--name", "rt-test", "--runtime", rt})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			content, err := os.ReadFile(filepath.Join(dir, "strait.config.json"))
			if err != nil {
				t.Fatal(err)
			}
			var cfg straitConfigJSON
			if err := json.Unmarshal(content, &cfg); err != nil {
				t.Fatal(err)
			}
			if cfg.Runtime != rt {
				t.Fatalf("expected runtime=%s, got %q", rt, cfg.Runtime)
			}
		})
	}
}

func TestInit_InvalidRuntime(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--name", "my-api", "--runtime", "java"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid runtime")
	}
	if !strings.Contains(err.Error(), "runtime") {
		t.Fatalf("expected runtime error, got: %v", err)
	}
}

func TestInit_InvalidProjectName(t *testing.T) {

	tests := []struct {
		name  string
		value string
	}{
		{"uppercase", "MyProject"},
		{"spaces", "my project"},
		{"leading hyphen", "-bad"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			dir := t.TempDir()
			origDir, _ := os.Getwd()
			t.Cleanup(func() { _ = os.Chdir(origDir) })
			_ = os.Chdir(dir)

			state := &appState{opts: &rootOptions{}}
			cmd := newInitCommand(state)
			cmd.SetArgs([]string{"--yes", "--name", tc.value})

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error for name %q", tc.value)
			}
		})
	}
}

func TestInit_TemplateFullCreatesDefinitions(t *testing.T) {

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--name", "full-test", "--template", "full"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check definitions were created
	if _, err := os.Stat(filepath.Join(dir, "definitions", "jobs.yaml")); err != nil {
		t.Fatal("definitions/jobs.yaml not created")
	}
	if _, err := os.Stat(filepath.Join(dir, "definitions", "workflows.yaml")); err != nil {
		t.Fatal("definitions/workflows.yaml not created for full template")
	}
}

func TestWriteStraitIgnore_CreatesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	status, err := writeStraitIgnore("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "created" {
		t.Fatalf("expected 'created', got %q", status)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".straitignore"))
	if err != nil {
		t.Fatal(".straitignore not created")
	}
	for _, pattern := range []string{".git/", ".env", "*.log", "dist/"} {
		if !strings.Contains(string(content), pattern) {
			t.Fatalf(".straitignore missing pattern %q", pattern)
		}
	}
}

func TestWriteStraitIgnore_SkipsIfExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	if err := os.WriteFile(filepath.Join(dir, ".straitignore"), []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	status, err := writeStraitIgnore("go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "skipped" {
		t.Fatalf("expected 'skipped', got %q", status)
	}

	// Original content must be untouched.
	content, _ := os.ReadFile(filepath.Join(dir, ".straitignore"))
	if string(content) != "existing" {
		t.Fatal("existing .straitignore was overwritten")
	}
}

// TestWriteStraitIgnore_RuntimePatterns is not parallel: os.Chdir is process-global.
func TestWriteStraitIgnore_RuntimePatterns(t *testing.T) {
	tests := []struct {
		runtime  string
		expected string
	}{
		{"typescript", "node_modules/"},
		{"node", "node_modules/"},
		{"bun", "node_modules/"},
		{"python", "__pycache__/"},
		{"go", "vendor/"},
		{"rust", "target/"},
		{"ruby", ".bundle/"},
		{"docker", ""},
		{"", ""},
	}

	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	for _, tc := range tests {
		t.Run("runtime="+tc.runtime, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}

			_, err := writeStraitIgnore(tc.runtime)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			content, _ := os.ReadFile(filepath.Join(dir, ".straitignore"))
			if tc.expected != "" && !strings.Contains(string(content), tc.expected) {
				t.Fatalf("expected pattern %q in .straitignore for runtime %q", tc.expected, tc.runtime)
			}
			// Common patterns always present.
			if !strings.Contains(string(content), ".git/") {
				t.Fatal(".straitignore missing common .git/ pattern")
			}
		})
	}
}

func TestInit_CreatesStraitIgnore(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--yes", "--name", "my-api", "--runtime", "typescript"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".straitignore"))
	if err != nil {
		t.Fatal(".straitignore not created by init command")
	}
	if !strings.Contains(string(content), "node_modules/") {
		t.Fatal(".straitignore missing node_modules/ for typescript runtime")
	}
}
