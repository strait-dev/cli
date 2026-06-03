//go:build e2e

// Package main's e2e suite exercises the built CLI against a live Strait server.
// It is excluded from normal `go test` by the e2e build tag and run via
// `make e2e`. Required environment:
//
//	STRAIT_SERVER   base URL of a running server (e.g. http://localhost:8080)
//	STRAIT_API_KEY  a valid API key (Bearer strait_...)
//	STRAIT_PROJECT  a project ID the key can access
//
// The suite builds the binary once and runs read + write-lifecycle commands,
// asserting the corrected paths actually reach the server (no 404 drift).
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireE2EEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"STRAIT_SERVER", "STRAIT_API_KEY", "STRAIT_PROJECT"} {
		if strings.TrimSpace(os.Getenv(k)) == "" {
			t.Skipf("%s not set; skipping e2e", k)
		}
	}
}

func buildE2EBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "strait")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

// runCLI runs the built binary with args and returns combined output + error.
func runCLI(t *testing.T, bin string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bin, append(args, "--format", "json")...)
	cmd.Env = append(os.Environ(), "STRAIT_NON_INTERACTIVE=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestE2EReadCommands(t *testing.T) {
	requireE2EEnv(t)
	bin := buildE2EBinary(t)

	// Each of these previously regressed on path/contract drift. They must all
	// reach the server and return success (cloud-only analytics may 402, which
	// is surfaced as a non-nil error and is therefore excluded here).
	reads := [][]string{
		{"jobs", "list"},
		{"runs", "list"},
		{"workflows", "list"},
		{"workflow-runs", "list"},
		{"webhooks", "list"},
		{"event-sources", "list"},
		{"log-drains", "list"},
		{"notifications", "list"},
		{"secrets", "list"},
		{"env", "list"},
		{"job-groups", "list"},
		{"api-keys", "list"},
		{"worker", "status"},
		{"triggers", "list"},
		{"runs", "dlq"},
	}
	for _, args := range reads {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, err := runCLI(t, bin, args...)
			if err != nil {
				t.Fatalf("%v failed: %v\n%s", args, err, out)
			}
		})
	}
}

func TestE2EJobLifecycle(t *testing.T) {
	requireE2EEnv(t)
	bin := buildE2EBinary(t)

	slug := "e2e-lifecycle-job"
	// Best-effort cleanup from a prior run.
	_, _ = runCLI(t, bin, "jobs", "delete", slug, "--yes")

	// create -> get -> delete must round-trip cleanly (delete exercises the
	// 204 No-Content path that previously surfaced a spurious EOF error).
	if out, err := runCLI(t, bin, "jobs", "create", "--name", "e2e lifecycle", "--slug", slug, "--endpoint", "https://example.com/strait"); err != nil {
		t.Fatalf("create: %v\n%s", err, out)
	}
	if out, err := runCLI(t, bin, "jobs", "get", slug); err != nil {
		t.Fatalf("get: %v\n%s", err, out)
	}
	if out, err := runCLI(t, bin, "jobs", "delete", slug, "--yes"); err != nil {
		t.Fatalf("delete: %v\n%s", err, out)
	}
}

func TestE2EJobTrigger(t *testing.T) {
	requireE2EEnv(t)
	bin := buildE2EBinary(t)

	slug := "e2e-trigger-job"
	if out, err := runCLI(t, bin, "jobs", "create", "--name", "e2e trigger", "--slug", slug, "--endpoint", "https://example.com/strait"); err != nil {
		// A pre-existing job from a prior run is fine.
		if !strings.Contains(out, "conflict") {
			t.Fatalf("create: %v\n%s", err, out)
		}
	}
	if out, err := runCLI(t, bin, "jobs", "trigger", slug); err != nil {
		t.Fatalf("trigger: %v\n%s", err, out)
	}
	// Leave the job in place: deleting with an active run is a legitimate 409.
}
