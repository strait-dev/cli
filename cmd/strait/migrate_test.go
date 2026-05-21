package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeSlug(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"send-welcome":       "send-welcome",
		"Send_Welcome Email": "send-welcome-email",
		"my.event/handler":   "my-event-handler",
		"  --__leading--  ":  "leading",
		"UPPER":              "upper",
		"a/b/c":              "a-b-c",
	}
	for in, want := range cases {
		if got := normalizeSlug(in); got != want {
			t.Errorf("normalizeSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCamelCase(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":             "migrated",
		"send-welcome": "sendWelcome",
		"a-b-c":        "aBC",
		"single":       "single",
	}
	for in, want := range cases {
		if got := camelCase(in); got != want {
			t.Errorf("camelCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMigrateInngest_GeneratesJobsAndManifest(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp) // t.Chdir is incompatible with t.Parallel.

	input := filepath.Join(tmp, "inngest.json")
	writeFile(t, input, `{
		"functions": [
			{"id": "send-welcome", "name": "Send welcome", "triggers": [{"event": "user/signup"}]},
			{"id": "daily-rollup", "name": "Daily rollup", "triggers": [{"cron": "0 6 * * *"}]}
		]
	}`)

	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newMigrateCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"inngest", "--input", input, "--out", "out"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	src, err := os.ReadFile(filepath.Join(tmp, "out", "jobs", "send-welcome.ts"))
	if err != nil {
		t.Fatalf("read send-welcome.ts: %v", err)
	}
	if !strings.Contains(string(src), `slug: "send-welcome"`) {
		t.Fatalf("missing slug declaration: %s", src)
	}
	if !strings.Contains(string(src), "Listens to event: user/signup") {
		t.Fatalf("missing event-listens comment: %s", src)
	}

	cron, err := os.ReadFile(filepath.Join(tmp, "out", "jobs", "daily-rollup.ts"))
	if err != nil {
		t.Fatalf("read daily-rollup.ts: %v", err)
	}
	if !strings.Contains(string(cron), "TODO: review — cron trigger '0 6 * * *'") {
		t.Fatalf("missing cron TODO: %s", cron)
	}

	manifestRaw, err := os.ReadFile(filepath.Join(tmp, "out", "strait.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest ProjectConfig
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaURL != "https://schemas.strait.dev/v1/strait.json" {
		t.Fatalf("$schema = %q", manifest.SchemaURL)
	}
	if manifest.Metadata["migration_platform"] != "inngest" {
		t.Fatalf("migration_platform = %v, want inngest", manifest.Metadata["migration_platform"])
	}
	if len(manifest.Jobs) != 2 {
		t.Fatalf("jobs = %d, want 2", len(manifest.Jobs))
	}
	if manifest.Jobs[0].EndpointURL == "" {
		t.Fatalf("expected placeholder endpoint_url in migrated strait.json")
	}
}

func TestMigrateTrigger_GeneratesJobs(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	input := filepath.Join(tmp, "trigger.json")
	writeFile(t, input, `{
		"jobs": [
			{"id": "onboarding", "name": "Onboarding", "trigger": {"type": "event", "event": {"name": "user.signup"}}},
			{"id": "weekly", "name": "Weekly", "trigger": {"type": "scheduled", "cron": "0 0 * * 0"}}
		]
	}`)

	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newMigrateCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"trigger", "--input", input, "--out", "out"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	src, err := os.ReadFile(filepath.Join(tmp, "out", "jobs", "onboarding.ts"))
	if err != nil {
		t.Fatalf("read onboarding.ts: %v", err)
	}
	if !strings.Contains(string(src), "Listens to event: user.signup") {
		t.Fatalf("missing event listener: %s", src)
	}
}

func TestMigrateHatchet_GeneratesJobs(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	input := filepath.Join(tmp, "hatchet.yaml")
	writeFile(t, input, `name: example-workflow
triggers:
  events:
    - user:create
  cron:
    - "*/5 * * * *"
jobs:
  step-one:
    description: First step
    timeout: 60s
  step-two:
    description: Second step
`)

	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newMigrateCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"hatchet", "--input", input, "--out", "out"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	for _, slug := range []string{"step-one", "step-two"} {
		path := filepath.Join(tmp, "out", "jobs", slug+".ts")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	step1, err := os.ReadFile(filepath.Join(tmp, "out", "jobs", "step-one.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(step1), "hatchet timeout '60s'") {
		t.Fatalf("missing timeout TODO: %s", step1)
	}
	if !strings.Contains(string(step1), "Listens to event: user:create") {
		t.Fatalf("missing event comment: %s", step1)
	}
}

func TestMigrate_RejectsMissingInput(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newMigrateCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"inngest"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when --input missing")
	}
}

func TestMigrate_RefusesOverwriteWithoutForce(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	input := filepath.Join(tmp, "inngest.json")
	writeFile(t, input, `{"functions":[{"id":"send-welcome","triggers":[{"event":"user/signup"}]}]}`)

	// First run: writes the sources.
	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newMigrateCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"inngest", "--input", input, "--out", "out"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first execute: %v", err)
	}

	// Hand-edit to verify --force is required.
	jobPath := filepath.Join(tmp, "out", "jobs", "send-welcome.ts")
	if err := os.WriteFile(jobPath, []byte("// user edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Second run without --force: must refuse + leave the user edit untouched.
	state2 := &appState{opts: &rootOptions{}}
	state2.stdout = &bytes.Buffer{}
	cmd2 := newMigrateCommand(state2)
	cmd2.SetOut(&bytes.Buffer{})
	cmd2.SetErr(&bytes.Buffer{})
	cmd2.SetArgs([]string{"inngest", "--input", input, "--out", "out"})
	err := cmd2.Execute()
	if err == nil {
		t.Fatal("expected error refusing to overwrite")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected --force hint, got: %v", err)
	}
	body, err := os.ReadFile(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "// user edit") {
		t.Fatalf("user edit was clobbered: %s", body)
	}

	// Third run with --force: should overwrite.
	state3 := &appState{opts: &rootOptions{}}
	state3.stdout = &bytes.Buffer{}
	cmd3 := newMigrateCommand(state3)
	cmd3.SetOut(&bytes.Buffer{})
	cmd3.SetErr(&bytes.Buffer{})
	cmd3.SetArgs([]string{"inngest", "--input", input, "--out", "out", "--force"})
	if err := cmd3.Execute(); err != nil {
		t.Fatalf("force execute: %v", err)
	}
	body, err = os.ReadFile(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "// user edit") {
		t.Fatalf("--force did not overwrite: %s", body)
	}
}

func TestMigrate_RejectsEmptyInput(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	input := filepath.Join(tmp, "empty.json")
	writeFile(t, input, `{"functions": []}`)

	state := &appState{opts: &rootOptions{}}
	state.stdout = &bytes.Buffer{}
	cmd := newMigrateCommand(state)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"inngest", "--input", input, "--out", "out"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for zero jobs")
	}
	if !strings.Contains(err.Error(), "no jobs") {
		t.Fatalf("unexpected error: %v", err)
	}
}
