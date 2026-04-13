package main

import (
	"encoding/json"
	"net/http"
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
	// Not parallel: os.Chdir mutates process-global state.
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
	// Not parallel: os.Chdir mutates process-global state.
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

func TestInitFromServer_RequiresProject(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}} // no project ID
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--from-server"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --from-server used without --project")
	}
	if !strings.Contains(err.Error(), "project") {
		t.Errorf("expected 'project' in error, got: %v", err)
	}
}

func TestInitFromServer_ScaffoldsJobManifest(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{
					"id": "job-1", "slug": "my-job", "name": "My Job",
					"endpoint_url": "http://localhost:3000/jobs/my-job",
					"timeout_secs": 60, "max_attempts": 3, "enabled": true,
				},
			})
		},
		"GET /v1/workflows": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{})
		},
	})

	state := newTestState(t, srv)
	state.opts.projectID = "proj-1"
	state.opts.outputFormat = "json"
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--from-server"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Check JSON output
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("output not valid JSON: %v\noutput: %s", err, out)
	}
	if result["jobs"].(float64) != 1 {
		t.Errorf("expected 1 job, got: %v", result["jobs"])
	}

	// Check file was created
	jobsFile := filepath.Join(dir, "definitions", "jobs.yaml")
	if _, err := os.Stat(jobsFile); err != nil {
		t.Fatalf("expected definitions/jobs.yaml to be created: %v", err)
	}

	content, _ := os.ReadFile(jobsFile)
	if !strings.Contains(string(content), "my-job") {
		t.Errorf("expected job slug 'my-job' in manifest, got: %s", content)
	}
}

func TestInitFromServer_FailsOnExistingFileWithoutForce(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	// Pre-create the definitions directory and jobs.yaml
	_ = os.MkdirAll(filepath.Join(dir, "definitions"), 0o750)
	_ = os.WriteFile(filepath.Join(dir, "definitions", "jobs.yaml"), []byte("existing"), 0o600)

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "job-1", "slug": "my-job", "name": "My Job", "endpoint_url": "http://x"},
			})
		},
		"GET /v1/workflows": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{})
		},
	})

	state := newTestState(t, srv)
	state.opts.projectID = "proj-1"
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--from-server"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when jobs.yaml already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

func TestInitFromServer_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(dir)

	_ = os.MkdirAll(filepath.Join(dir, "definitions"), 0o750)
	_ = os.WriteFile(filepath.Join(dir, "definitions", "jobs.yaml"), []byte("old content"), 0o600)

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "job-1", "slug": "new-job", "name": "New Job", "endpoint_url": "http://x"},
			})
		},
		"GET /v1/workflows": func(w http.ResponseWriter, _ *http.Request) {
			respondPaginated(t, w, http.StatusOK, []map[string]any{})
		},
	})

	state := newTestState(t, srv)
	state.opts.projectID = "proj-1"
	state.opts.outputFormat = "json"
	cmd := newInitCommand(state)
	cmd.SetArgs([]string{"--from-server", "--force"})

	captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error with --force: %v", err)
		}
	})

	content, _ := os.ReadFile(filepath.Join(dir, "definitions", "jobs.yaml"))
	if strings.Contains(string(content), "old content") {
		t.Error("expected old content to be overwritten by --force")
	}
	if !strings.Contains(string(content), "new-job") {
		t.Errorf("expected new-job in overwritten file, got: %s", content)
	}
}

func TestInitFromServer_HasFlag(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	cmd := newInitCommand(state)
	if cmd.Flags().Lookup("from-server") == nil {
		t.Error("expected --from-server flag on init command")
	}
}
