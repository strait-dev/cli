package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

const testJobManifestYAML = `apiVersion: v1
kind: Job
metadata:
  name: test-job
spec:
  project_id: proj-test
  slug: test-job
  endpoint_url: https://example.com/hook
  max_attempts: 3
  timeout_secs: 60
`

var testDeclJob = types.Job{
	ID:          "job-1",
	ProjectID:   "proj-test",
	Name:        "test-job",
	Slug:        "test-job",
	EndpointURL: "https://example.com/hook",
	MaxAttempts: 3,
	TimeoutSecs: 60,
	Enabled:     true,
	Version:     1,
	CreatedAt:   time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
	UpdatedAt:   time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
}

func writeManifestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return p
}

func TestValidate_Success(t *testing.T) {
	t.Parallel()

	path := writeManifestFile(t, testJobManifestYAML)

	state := newTestState(t, newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	})))
	cmd := newValidateCommand(state)
	cmd.SetArgs([]string{"-f", path})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "test-job") {
		t.Fatalf("expected test-job in output, got: %s", out)
	}
}

func TestValidate_InvalidKind(t *testing.T) {
	t.Parallel()

	yaml := `apiVersion: v1
kind: Unknown
metadata:
  name: bad-resource
spec:
  project_id: proj-test
`
	path := writeManifestFile(t, yaml)

	state := newTestState(t, newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	})))
	cmd := newValidateCommand(state)
	cmd.SetArgs([]string{"-f", path})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported kind") {
		t.Fatalf("expected unsupported kind error, got: %v", err)
	}
}

func TestValidate_MissingSpec(t *testing.T) {
	t.Parallel()

	yaml := `apiVersion: v1
kind: Job
metadata:
  name: no-spec-job
`
	path := writeManifestFile(t, yaml)

	state := newTestState(t, newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	})))
	cmd := newValidateCommand(state)
	cmd.SetArgs([]string{"-f", path})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "spec is required") {
		t.Fatalf("expected spec required error, got: %v", err)
	}
}

func TestApply_DryRun(t *testing.T) {
	t.Parallel()

	path := writeManifestFile(t, testJobManifestYAML)

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []types.Job{})
		},
	})

	state := newTestState(t, srv)
	cmd := newApplyCommand(state)
	cmd.SetArgs([]string{"-f", path, "--dry-run"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "dry-run") {
		t.Fatalf("expected dry-run in output, got: %s", out)
	}
}

func TestApply_CreatesJob(t *testing.T) {
	t.Parallel()

	path := writeManifestFile(t, testJobManifestYAML)

	postCalled := false
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []types.Job{})
		},
		"POST /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			postCalled = true
			respondJSON(t, w, http.StatusCreated, testDeclJob)
		},
	})

	state := newTestState(t, srv)
	cmd := newApplyCommand(state)
	cmd.SetArgs([]string{"-f", path})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !postCalled {
		t.Fatal("expected POST /v1/jobs to be called")
	}
	if !strings.Contains(out, "created") {
		t.Fatalf("expected created in output, got: %s", out)
	}
}

func TestApply_UpdatesExistingJob(t *testing.T) {
	t.Parallel()

	path := writeManifestFile(t, testJobManifestYAML)

	patchCalled := false
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []types.Job{testDeclJob})
		},
		"PATCH /v1/jobs/job-1": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			patchCalled = true
			respondJSON(t, w, http.StatusOK, testDeclJob)
		},
	})

	state := newTestState(t, srv)
	cmd := newApplyCommand(state)
	cmd.SetArgs([]string{"-f", path})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !patchCalled {
		t.Fatal("expected PATCH /v1/jobs/job-1 to be called")
	}
	if !strings.Contains(out, "updated") {
		t.Fatalf("expected updated in output, got: %s", out)
	}
}

func TestDiff_NewResource(t *testing.T) {
	t.Parallel()

	path := writeManifestFile(t, testJobManifestYAML)

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []types.Job{})
		},
	})

	state := newTestState(t, srv)
	cmd := newDiffCommand(state)
	cmd.SetArgs([]string{"-f", path})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "create") {
		t.Fatalf("expected create in output, got: %s", out)
	}
}

func TestDiff_ExistingResource(t *testing.T) {
	t.Parallel()

	path := writeManifestFile(t, testJobManifestYAML)

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/jobs": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []types.Job{testDeclJob})
		},
	})

	state := newTestState(t, srv)
	cmd := newDiffCommand(state)
	cmd.SetArgs([]string{"-f", path})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "update") {
		t.Fatalf("expected update in output, got: %s", out)
	}
}
