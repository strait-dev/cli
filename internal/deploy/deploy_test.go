package deploy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
)

func TestDeployJob_MissingImageAndDockerfile(t *testing.T) {
	t.Parallel()
	opts := DeployOptions{
		JobSlug:    "my-job",
		ImageURI:   "",
		Dockerfile: "",
	}
	err := DeployJob(t.Context(), nil, opts)
	if err == nil {
		t.Fatal("expected error for missing --image and empty dockerfile")
	}
}

func TestDeployJob_DryRun_NoAPICalls(t *testing.T) {
	t.Parallel()
	opts := DeployOptions{
		JobSlug:  "my-job",
		ImageURI: "registry.fly.io/my-app:abc123",
		DryRun:   true,
	}
	// Should not panic even with nil client.
	err := DeployJob(t.Context(), nil, opts)
	if err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
}

func TestDeployJob_InvalidPreset(t *testing.T) {
	t.Parallel()
	err := UpdateJobImage(t.Context(), nil, "my-job", "img:latest", "nonexistent", "")
	if err == nil {
		t.Fatal("expected error for invalid preset")
	}
}

func TestValidPresets(t *testing.T) {
	t.Parallel()
	presets := []string{"micro", "small-1x", "small-2x", "medium-1x", "medium-2x", "large-1x", "large-2x"}
	for _, p := range presets {
		if !types.MachinePreset(p).IsValid() {
			t.Errorf("expected %q to be valid", p)
		}
	}
}

func TestGitSHA(t *testing.T) {
	t.Parallel()
	sha, err := gitSHA(t.Context())
	if err != nil {
		t.Skipf("git not available: %v", err)
	}
	if len(sha) < 7 {
		t.Errorf("expected SHA >= 7 chars, got %q", sha)
	}
}

func TestUpdateJobImage_Success(t *testing.T) {
	t.Parallel()

	var patchBody map[string]any
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&patchBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(types.Job{
			ID: "job-1", Name: "test", Slug: "test", EndpointURL: "https://x.com",
			ImageURI: "registry.fly.io/app:v1", CreatedAt: now, UpdatedAt: now,
		})
	}))
	defer srv.Close()

	cli, err := client.New(srv.URL, "key", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	err = UpdateJobImage(t.Context(), cli, "job-1", "registry.fly.io/app:v1", "small-1x", "iad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patchBody["image_uri"] != "registry.fly.io/app:v1" {
		t.Fatalf("expected image_uri in PATCH body, got: %v", patchBody)
	}
	if patchBody["machine_preset"] != "small-1x" {
		t.Fatalf("expected machine_preset in PATCH body, got: %v", patchBody)
	}
	if patchBody["region"] != "iad" {
		t.Fatalf("expected region in PATCH body, got: %v", patchBody)
	}
}

func TestUpdateJobImage_NoPreset(t *testing.T) {
	t.Parallel()

	var patchBody map[string]any
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&patchBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(types.Job{
			ID: "job-1", Name: "test", Slug: "test", EndpointURL: "https://x.com",
			CreatedAt: now, UpdatedAt: now,
		})
	}))
	defer srv.Close()

	cli, err := client.New(srv.URL, "key", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	err = UpdateJobImage(t.Context(), cli, "job-1", "img:latest", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := patchBody["machine_preset"]; ok {
		t.Fatal("expected no machine_preset when preset is empty")
	}
	if _, ok := patchBody["region"]; ok {
		t.Fatal("expected no region when region is empty")
	}
}

func TestDeployJob_WithImage_DryRun(t *testing.T) {
	t.Parallel()

	opts := DeployOptions{
		JobSlug:  "my-job",
		ImageURI: "registry.fly.io/my-app:abc",
		Preset:   "medium-1x",
		Region:   "iad",
		DryRun:   true,
	}
	err := DeployJob(t.Context(), nil, opts)
	if err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
}

func TestDeployJob_WithImage_Success(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(types.Job{
			ID: "job-1", Name: "test", Slug: "my-job", EndpointURL: "https://x.com",
			ImageURI: "registry.fly.io/my-app:abc", CreatedAt: now, UpdatedAt: now,
		})
	}))
	defer srv.Close()

	cli, err := client.New(srv.URL, "key", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	opts := DeployOptions{
		JobSlug:  "my-job",
		ImageURI: "registry.fly.io/my-app:abc",
	}
	err = DeployJob(t.Context(), cli, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
