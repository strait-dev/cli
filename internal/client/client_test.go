package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "valid http", url: "http://localhost:8080", wantErr: false},
		{name: "valid https", url: "https://api.example.com", wantErr: false},
		{name: "trailing slash", url: "http://localhost:8080/", wantErr: false},
		{name: "empty url", url: "", wantErr: true},
		{name: "whitespace", url: "   ", wantErr: true},
		{name: "invalid scheme", url: "ftp://example.com", wantErr: true},
		{name: "no scheme", url: "example.com", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := New(tc.url, "test-key", 10*time.Second)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c == nil {
				t.Fatal("expected client, got nil")
			}
		})
	}
}

func TestNew_DefaultTimeout(t *testing.T) {
	t.Parallel()

	c, err := New("http://localhost:8080", "key", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.http.Timeout != 30*time.Second {
		t.Fatalf("expected 30s timeout, got %v", c.http.Timeout)
	}
}

func TestNew_StreamHTTPHasNoTimeout(t *testing.T) {
	t.Parallel()

	c, err := New("http://localhost:8080", "key", 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.streamHTTP.Timeout != 0 {
		t.Fatalf("expected 0 timeout for streamHTTP, got %v", c.streamHTTP.Timeout)
	}
}

func TestListJobs(t *testing.T) {
	t.Parallel()

	jobs := []types.Job{
		{ID: "job-1", ProjectID: "proj-1", Name: "Test Job", Slug: "test-job", Enabled: true},
		{ID: "job-2", ProjectID: "proj-1", Name: "Other Job", Slug: "other-job", Enabled: false},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/jobs")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Errorf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		respondPaginated(t, w, http.StatusOK, jobs)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListJobs(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(got))
	}
	if got[0].ID != "job-1" {
		t.Fatalf("expected job-1, got %s", got[0].ID)
	}
}

func TestGetJob(t *testing.T) {
	t.Parallel()

	job := types.Job{ID: "job-1", ProjectID: "proj-1", Name: "Test Job", Slug: "test-job"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/jobs/job-1")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, job)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.GetJob(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.ID != "job-1" {
		t.Fatalf("expected job-1, got %s", got.ID)
	}
}

func TestCreateJob(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/jobs")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req CreateJobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.Name != "New Job" {
			t.Fatalf("expected name=New Job, got %q", req.Name)
		}

		respondJSON(t, w, http.StatusOK, types.Job{ID: "job-new", Name: req.Name})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.CreateJob(context.Background(), CreateJobRequest{
		ProjectID:   "proj-1",
		Name:        "New Job",
		Slug:        "new-job",
		EndpointURL: "https://example.com/hook",
	}, "")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if got.ID != "job-new" {
		t.Fatalf("expected job-new, got %s", got.ID)
	}
}

func TestCreateJob_SendsIdempotencyKeyHeader(t *testing.T) {
	t.Parallel()

	var capturedKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedKey = r.Header.Get("X-Idempotency-Key")
		respondJSON(t, w, http.StatusOK, types.Job{ID: "job-1"})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.CreateJob(context.Background(), CreateJobRequest{
		ProjectID: "proj-1", Name: "J", Slug: "j", EndpointURL: "http://x",
	}, "idem-key-1")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if capturedKey != "idem-key-1" {
		t.Errorf("X-Idempotency-Key: got %q, want %q", capturedKey, "idem-key-1")
	}
}

func TestCreateJob_EmptyIdempotencyKeyOmitsHeader(t *testing.T) {
	t.Parallel()

	var capturedKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedKey = r.Header.Get("X-Idempotency-Key")
		respondJSON(t, w, http.StatusOK, types.Job{ID: "job-1"})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.CreateJob(context.Background(), CreateJobRequest{
		ProjectID: "proj-1", Name: "J", Slug: "j", EndpointURL: "http://x",
	}, "")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if capturedKey != "" {
		t.Errorf("expected empty X-Idempotency-Key, got %q", capturedKey)
	}
}

func TestCreateWorkflow_SendsIdempotencyKeyHeader(t *testing.T) {
	t.Parallel()

	var capturedKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedKey = r.Header.Get("X-Idempotency-Key")
		respondJSON(t, w, http.StatusOK, WorkflowResponse{Workflow: types.Workflow{ID: "wf-1"}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.CreateWorkflow(context.Background(), CreateWorkflowRequest{ProjectID: "p", Name: "W", Slug: "w"}, "wf-idem-key")
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if capturedKey != "wf-idem-key" {
		t.Errorf("X-Idempotency-Key: got %q, want %q", capturedKey, "wf-idem-key")
	}
}

func TestDeleteJob(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/v1/jobs/job-1")
		assertAuth(t, r, "test-key")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if err := c.DeleteJob(context.Background(), "job-1"); err != nil {
		t.Fatalf("DeleteJob: %v", err)
	}
}

func TestListRuns(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)
	runs := []types.JobRun{
		{ID: "run-1", JobID: "job-1", ProjectID: "proj-1", Status: types.StatusCompleted, CreatedAt: now},
		{ID: "run-2", JobID: "job-1", ProjectID: "proj-1", Status: types.StatusFailed, CreatedAt: now.Add(-time.Minute)},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/runs")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Errorf("expected project_id=proj-1")
		}
		if r.URL.Query().Get("limit") != "50" {
			t.Errorf("expected limit=50, got %q", r.URL.Query().Get("limit"))
		}
		respondPaginated(t, w, http.StatusOK, runs)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListRuns(context.Background(), "proj-1", "", 50, nil)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(got))
	}
}

func TestListRuns_WithCursor(t *testing.T) {
	t.Parallel()

	cursor := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		cursorParam := r.URL.Query().Get("cursor")
		if cursorParam == "" {
			t.Fatal("expected cursor query parameter")
		}
		parsed, err := time.Parse(time.RFC3339, cursorParam)
		if err != nil {
			t.Fatalf("cursor not RFC3339: %v", err)
		}
		if !parsed.Equal(cursor) {
			t.Fatalf("cursor mismatch: got %v, want %v", parsed, cursor)
		}
		respondPaginated(t, w, http.StatusOK, []types.JobRun{})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.ListRuns(context.Background(), "proj-1", "", 50, &cursor)
	if err != nil {
		t.Fatalf("ListRuns with cursor: %v", err)
	}
}

func TestListRuns_WithStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("status") != "executing" {
			t.Fatalf("expected status=executing, got %q", r.URL.Query().Get("status"))
		}
		respondPaginated(t, w, http.StatusOK, []types.JobRun{})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.ListRuns(context.Background(), "proj-1", "executing", 50, nil)
	if err != nil {
		t.Fatalf("ListRuns with status: %v", err)
	}
}

func TestListAllRuns_Pagination(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)
	var callCount atomic.Int32

	// First page: 100 runs, second page: 5 runs
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		switch n {
		case 1:
			// First page: no cursor, return 100 runs
			if r.URL.Query().Get("cursor") != "" {
				t.Error("first request should not have cursor")
			}
			runs := make([]types.JobRun, 100)
			for i := range runs {
				runs[i] = types.JobRun{
					ID:        fmt.Sprintf("run-%d", i),
					ProjectID: "proj-1",
					CreatedAt: now.Add(-time.Duration(i) * time.Second),
				}
			}
			respondPaginated(t, w, http.StatusOK, runs)
		case 2:
			// Second page: should have cursor
			if r.URL.Query().Get("cursor") == "" {
				t.Error("second request should have cursor")
			}
			runs := make([]types.JobRun, 5)
			for i := range runs {
				runs[i] = types.JobRun{
					ID:        fmt.Sprintf("run-page2-%d", i),
					ProjectID: "proj-1",
					CreatedAt: now.Add(-time.Duration(100+i) * time.Second),
				}
			}
			respondPaginated(t, w, http.StatusOK, runs)
		default:
			t.Fatalf("unexpected call #%d", n)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListAllRuns(context.Background(), "proj-1", "")
	if err != nil {
		t.Fatalf("ListAllRuns: %v", err)
	}
	if len(got) != 105 {
		t.Fatalf("expected 105 total runs, got %d", len(got))
	}
	if callCount.Load() != 2 {
		t.Fatalf("expected 2 HTTP calls, got %d", callCount.Load())
	}
}

func TestTriggerJob(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/jobs/job-1/trigger")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		respondJSON(t, w, http.StatusOK, TriggerJobResponse{
			ID: "run-1",
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.TriggerJob(context.Background(), "job-1", TriggerJobRequest{}, "")
	if err != nil {
		t.Fatalf("TriggerJob: %v", err)
	}
	if got.ID != "run-1" {
		t.Fatalf("expected run-1, got %s", got.ID)
	}
}

func TestTriggerJob_IdempotencyKey(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Idempotency-Key")
		if key != "my-key-123" {
			t.Fatalf("expected idempotency key my-key-123, got %q", key)
		}
		respondJSON(t, w, http.StatusOK, TriggerJobResponse{ID: "run-1"})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.TriggerJob(context.Background(), "job-1", TriggerJobRequest{}, "my-key-123")
	if err != nil {
		t.Fatalf("TriggerJob with idempotency key: %v", err)
	}
}

func TestDoJSON_4xxError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid payload"}`))
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.ListJobs(context.Background(), "proj-1")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("error should contain status code: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid payload") {
		t.Fatalf("error should contain message: %v", err)
	}
}

func TestDoJSON_AuthHeader(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret-key-123" {
			t.Fatalf("expected Bearer secret-key-123, got %q", auth)
		}
		respondPaginated(t, w, http.StatusOK, []types.Job{})
	}))
	defer srv.Close()

	c, err := New(srv.URL, "secret-key-123", 10*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = c.ListJobs(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
}

func TestDoJSON_Retry429(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := callCount.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		respondPaginated(t, w, http.StatusOK, []types.Job{{ID: "job-1"}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListJobs(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("expected success after retries: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 job, got %d", len(got))
	}
	if callCount.Load() != 3 {
		t.Fatalf("expected 3 calls (2 retries + 1 success), got %d", callCount.Load())
	}
}

func TestDoJSON_Retry5xx(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := callCount.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		respondPaginated(t, w, http.StatusOK, []types.Job{{ID: "job-1"}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListJobs(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("expected success after retries: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 job, got %d", len(got))
	}
	if callCount.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount.Load())
	}
}

func TestDoJSON_RetryExhausted(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.ListJobs(context.Background(), "proj-1")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if callCount.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", callCount.Load())
	}
}

func TestUpdateJob(t *testing.T) {
	t.Parallel()

	name := "Updated Job"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPatch)
		assertPath(t, r, "/v1/jobs/job-1")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req UpdateJobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.Name == nil || *req.Name != name {
			t.Fatalf("expected name=%q", name)
		}

		respondJSON(t, w, http.StatusOK, types.Job{ID: "job-1", Name: name})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.UpdateJob(context.Background(), "job-1", UpdateJobRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateJob: %v", err)
	}
	if got.Name != name {
		t.Fatalf("expected %q, got %q", name, got.Name)
	}
}

func TestBulkTriggerJob(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/jobs/job-1/trigger/bulk")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req BulkTriggerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if len(req.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(req.Items))
		}

		respondJSON(t, w, http.StatusOK, BulkTriggerResponse{
			Results: []BulkTriggerResult{{ID: "run-1", Status: "queued"}},
			Total:   1,
			Created: 1,
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.BulkTriggerJob(context.Background(), "job-1", BulkTriggerRequest{Items: []BulkTriggerItem{{Priority: 3}}})
	if err != nil {
		t.Fatalf("BulkTriggerJob: %v", err)
	}
	if got.Created != 1 || len(got.Results) != 1 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestListJobVersions(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/jobs/job-1/versions")
		assertAuth(t, r, "test-key")
		respondPaginated(t, w, http.StatusOK, []types.JobVersion{{
			ID:          "jv-1",
			JobID:       "job-1",
			Version:     1,
			Name:        "Job",
			Slug:        "job",
			EndpointURL: "https://example.com/hook",
			MaxAttempts: 3,
			TimeoutSecs: 30,
			CreatedAt:   now,
		}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListJobVersions(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("ListJobVersions: %v", err)
	}
	if len(got) != 1 || got[0].ID != "jv-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetRun(t *testing.T) {
	t.Parallel()

	run := types.JobRun{ID: "run-1", JobID: "job-1", ProjectID: "proj-1", Status: types.StatusExecuting}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/runs/run-1")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, run)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.GetRun(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.ID != "run-1" {
		t.Fatalf("expected run-1, got %s", got.ID)
	}
}

func TestCancelRun(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/v1/runs/run-1")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, types.JobRun{ID: "run-1", Status: types.StatusCanceled})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.CancelRun(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("CancelRun: %v", err)
	}
	if got.Status != types.StatusCanceled {
		t.Fatalf("expected status canceled, got %s", got.Status)
	}
}

func TestBulkCancelRuns(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/runs/bulk-cancel")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req BulkCancelRunsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if len(req.IDs) != 3 {
			t.Fatalf("expected 3 ids, got %d", len(req.IDs))
		}

		respondJSON(t, w, http.StatusOK, BulkCancelRunsResponse{
			Results: []BulkCancelResult{
				{ID: "run-1", Canceled: true, Status: "canceled"},
				{ID: "run-2", Canceled: true, Status: "canceled"},
				{ID: "run-3", Canceled: false, Error: "already terminal"},
			},
			Total:    3,
			Canceled: 2,
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.BulkCancelRuns(context.Background(), []string{"run-1", "run-2", "run-3"})
	if err != nil {
		t.Fatalf("BulkCancelRuns: %v", err)
	}
	if got.Total != 3 || got.Canceled != 2 {
		t.Fatalf("unexpected counters: %+v", got)
	}
	if len(got.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got.Results))
	}
	if got.Results[2].Canceled || got.Results[2].Error == "" {
		t.Fatalf("expected third run to be reported as failed, got %+v", got.Results[2])
	}
}

func TestReplayRun(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/runs/run-1/replay")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, types.JobRun{
			ID:          "run-2",
			JobID:       "job-1",
			ParentRunID: "run-1",
			Status:      types.StatusQueued,
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ReplayRun(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("ReplayRun: %v", err)
	}
	if got.ID != "run-2" {
		t.Fatalf("expected replayed run id run-2, got %s", got.ID)
	}
	if got.ParentRunID != "run-1" {
		t.Fatalf("expected ParentRunID=run-1, got %q", got.ParentRunID)
	}
}

func TestListRunEvents(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/runs/run-1/events")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("level") != "info" {
			t.Fatalf("expected level=info, got %q", r.URL.Query().Get("level"))
		}
		if r.URL.Query().Get("type") != "progress" {
			t.Fatalf("expected type=progress, got %q", r.URL.Query().Get("type"))
		}
		respondPaginated(t, w, http.StatusOK, []types.RunEvent{{
			ID:        "evt-1",
			RunID:     "run-1",
			Type:      types.EventType("progress"),
			Level:     "info",
			Message:   "step done",
			CreatedAt: now,
		}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListRunEvents(context.Background(), "run-1", "info", "progress")
	if err != nil {
		t.Fatalf("ListRunEvents: %v", err)
	}
	if len(got) != 1 || got[0].ID != "evt-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestListWorkflows(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/workflows")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		respondPaginated(t, w, http.StatusOK, []types.Workflow{{ID: "wf-1", ProjectID: "proj-1", Name: "Flow", Slug: "flow", Enabled: true}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListWorkflows(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	if len(got) != 1 || got[0].ID != "wf-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetWorkflow(t *testing.T) {
	t.Parallel()

	resp := WorkflowResponse{
		Workflow: types.Workflow{ID: "wf-1", ProjectID: "proj-1", Name: "Flow", Slug: "flow", Enabled: true},
		Steps:    []types.WorkflowStep{{ID: "step-1", WorkflowID: "wf-1", StepRef: "step_1", DependsOn: []string{}, OnFailure: types.FailWorkflow}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/workflows/wf-1")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, resp)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.GetWorkflow(context.Background(), "wf-1")
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.ID != "wf-1" || len(got.Steps) != 1 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestCreateWorkflow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/workflows")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req CreateWorkflowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.ProjectID != "proj-1" || req.Name != "Flow" {
			t.Fatalf("unexpected request: %+v", req)
		}

		respondJSON(t, w, http.StatusOK, WorkflowResponse{Workflow: types.Workflow{ID: "wf-1", ProjectID: req.ProjectID, Name: req.Name, Slug: req.Slug, Enabled: true}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.CreateWorkflow(context.Background(), CreateWorkflowRequest{ProjectID: "proj-1", Name: "Flow", Slug: "flow"}, "")
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if got.ID != "wf-1" {
		t.Fatalf("expected wf-1, got %s", got.ID)
	}
}

func TestUpdateWorkflow(t *testing.T) {
	t.Parallel()

	name := "Renamed Flow"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPatch)
		assertPath(t, r, "/v1/workflows/wf-1")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req UpdateWorkflowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.Name == nil || *req.Name != name {
			t.Fatalf("expected name=%q", name)
		}

		respondJSON(t, w, http.StatusOK, WorkflowResponse{Workflow: types.Workflow{ID: "wf-1", Name: name}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.UpdateWorkflow(context.Background(), "wf-1", UpdateWorkflowRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}
	if got.Name != name {
		t.Fatalf("expected %q, got %q", name, got.Name)
	}
}

func TestDeleteWorkflow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/v1/workflows/wf-1")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, map[string]string{})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if err := c.DeleteWorkflow(context.Background(), "wf-1"); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
}

func TestTriggerWorkflow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/workflows/wf-1/trigger")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req TriggerWorkflowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.ProjectID != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", req.ProjectID)
		}

		respondJSON(t, w, http.StatusOK, types.WorkflowRun{ID: "wr-1", WorkflowID: "wf-1", ProjectID: "proj-1", Status: types.WfStatusPending})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.TriggerWorkflow(context.Background(), "wf-1", TriggerWorkflowRequest{ProjectID: "proj-1"})
	if err != nil {
		t.Fatalf("TriggerWorkflow: %v", err)
	}
	if got.ID != "wr-1" {
		t.Fatalf("expected wr-1, got %s", got.ID)
	}
}

func TestListWorkflowRuns(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/workflows/wf-1/runs")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("limit") != "20" {
			t.Fatalf("expected limit=20, got %q", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("offset") != "40" {
			t.Fatalf("expected offset=40, got %q", r.URL.Query().Get("offset"))
		}
		respondPaginated(t, w, http.StatusOK, []types.WorkflowRun{{ID: "wr-1", WorkflowID: "wf-1", ProjectID: "proj-1", Status: types.WfStatusRunning}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListWorkflowRuns(context.Background(), "wf-1", 20, 40)
	if err != nil {
		t.Fatalf("ListWorkflowRuns: %v", err)
	}
	if len(got) != 1 || got[0].ID != "wr-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetWorkflowRun(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/workflow-runs/wr-1")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, types.WorkflowRun{ID: "wr-1", WorkflowID: "wf-1", ProjectID: "proj-1", Status: types.WfStatusRunning})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.GetWorkflowRun(context.Background(), "wr-1")
	if err != nil {
		t.Fatalf("GetWorkflowRun: %v", err)
	}
	if got.ID != "wr-1" {
		t.Fatalf("expected wr-1, got %s", got.ID)
	}
}

func TestCancelWorkflowRun(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/v1/workflow-runs/wr-1")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, types.WorkflowRun{ID: "wr-1", WorkflowID: "wf-1", ProjectID: "proj-1", Status: types.WfStatusCanceled})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.CancelWorkflowRun(context.Background(), "wr-1")
	if err != nil {
		t.Fatalf("CancelWorkflowRun: %v", err)
	}
	if got.Status != types.WfStatusCanceled {
		t.Fatalf("expected status canceled, got %s", got.Status)
	}
}

func TestListWorkflowStepRuns(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/workflow-runs/wr-1/steps")
		assertAuth(t, r, "test-key")
		respondPaginated(t, w, http.StatusOK, []types.WorkflowStepRun{{
			ID:             "wsr-1",
			WorkflowRunID:  "wr-1",
			WorkflowStepID: "step-1",
			StepRef:        "step_1",
			Attempt:        1,
			Status:         types.StepRunning,
		}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListWorkflowStepRuns(context.Background(), "wr-1")
	if err != nil {
		t.Fatalf("ListWorkflowStepRuns: %v", err)
	}
	if len(got) != 1 || got[0].ID != "wsr-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/api-keys")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req CreateAPIKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.ProjectID != "proj-1" || req.Name != "cli" {
			t.Fatalf("unexpected request: %+v", req)
		}

		respondJSON(t, w, http.StatusOK, APIKeyCreateResponse{
			ID:        "key-1",
			ProjectID: req.ProjectID,
			Name:      req.Name,
			Key:       "strait_live_123",
			KeyPrefix: "strait_",
			Scopes:    req.Scopes,
			CreatedAt: now,
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.CreateAPIKey(context.Background(), CreateAPIKeyRequest{ProjectID: "proj-1", Name: "cli", Scopes: []string{"jobs:read"}})
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if got.ID != "key-1" || got.Key == "" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestListAPIKeys(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/api-keys")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		respondPaginated(t, w, http.StatusOK, []types.APIKey{{ID: "key-1", ProjectID: "proj-1", Name: "cli", KeyPrefix: "strait_", Scopes: []string{"jobs:read"}}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListAPIKeys(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(got) != 1 || got[0].ID != "key-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestRevokeAPIKey(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/v1/api-keys/key-1")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, map[string]string{})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if err := c.RevokeAPIKey(context.Background(), "key-1"); err != nil {
		t.Fatalf("RevokeAPIKey: %v", err)
	}
}

func TestRotateAPIKey(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/api-keys/key-1/rotate")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req RotateAPIKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.GracePeriodMinutes != 30 {
			t.Fatalf("expected grace_period_minutes=30, got %d", req.GracePeriodMinutes)
		}

		respondJSON(t, w, http.StatusOK, RotateAPIKeyResponse{
			OldKeyID:       "key-1",
			NewKeyID:       "key-2",
			ProjectID:      "proj-1",
			Name:           "cli",
			Key:            "strait_live_456",
			KeyPrefix:      "strait_",
			Scopes:         []string{"jobs:read"},
			CreatedAt:      now,
			GraceExpiresAt: now.Add(30 * time.Minute),
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.RotateAPIKey(context.Background(), "key-1", RotateAPIKeyRequest{GracePeriodMinutes: 30})
	if err != nil {
		t.Fatalf("RotateAPIKey: %v", err)
	}
	if got.NewKeyID != "key-2" {
		t.Fatalf("expected key-2, got %s", got.NewKeyID)
	}
}

func TestListEventTriggers(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/events")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		if r.URL.Query().Get("status") != "waiting" {
			t.Fatalf("expected status=waiting, got %q", r.URL.Query().Get("status"))
		}
		respondPaginated(t, w, http.StatusOK, []types.EventTrigger{{
			ID:          "et-1",
			EventKey:    "payment.received",
			ProjectID:   "proj-1",
			SourceType:  types.EventSourceWorkflowStep,
			Status:      types.EventTriggerStatusWaiting,
			TimeoutSecs: 120,
			RequestedAt: now,
			ExpiresAt:   now.Add(time.Hour),
		}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListEventTriggers(context.Background(), "proj-1", "waiting")
	if err != nil {
		t.Fatalf("ListEventTriggers: %v", err)
	}
	if len(got) != 1 || got[0].ID != "et-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetEventTrigger(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/events/payment.received")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, types.EventTrigger{
			ID:          "et-1",
			EventKey:    "payment.received",
			ProjectID:   "proj-1",
			SourceType:  types.EventSourceWorkflowStep,
			Status:      types.EventTriggerStatusWaiting,
			TimeoutSecs: 120,
			RequestedAt: now,
			ExpiresAt:   now.Add(time.Hour),
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.GetEventTrigger(context.Background(), "payment.received")
	if err != nil {
		t.Fatalf("GetEventTrigger: %v", err)
	}
	if got.EventKey != "payment.received" {
		t.Fatalf("expected payment.received, got %s", got.EventKey)
	}
}

func TestSendEvent(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/events/payment.received/send")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["payload"]["order_id"] != "ord-123" {
			t.Fatalf("unexpected payload: %+v", body)
		}

		respondJSON(t, w, http.StatusOK, types.EventTrigger{
			ID:              "et-1",
			EventKey:        "payment.received",
			ProjectID:       "proj-1",
			SourceType:      types.EventSourceWorkflowStep,
			Status:          types.EventTriggerStatusReceived,
			TimeoutSecs:     120,
			RequestedAt:     now,
			ReceivedAt:      &now,
			ExpiresAt:       now.Add(time.Hour),
			ResponsePayload: mustMarshal(t, map[string]any{"ok": true}),
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.SendEvent(context.Background(), "payment.received", map[string]any{"order_id": "ord-123"})
	if err != nil {
		t.Fatalf("SendEvent: %v", err)
	}
	if got.Status != types.EventTriggerStatusReceived {
		t.Fatalf("expected received, got %s", got.Status)
	}
}

func TestPurgeEventTriggers(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/events/purge")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["older_than_days"] != float64(30) {
			t.Fatalf("expected older_than_days=30, got %+v", body["older_than_days"])
		}
		if body["dry_run"] != false {
			t.Fatalf("expected dry_run=false, got %+v", body["dry_run"])
		}

		respondJSON(t, w, http.StatusOK, map[string]any{"deleted": 3})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.PurgeEventTriggers(context.Background(), 30, false)
	if err != nil {
		t.Fatalf("PurgeEventTriggers: %v", err)
	}
	if got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
}

func TestStats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/stats")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, QueueStats{Queued: 10, Executing: 2, Delayed: 1})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if got.Queued != 10 || got.Executing != 2 || got.Delayed != 1 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestHealth(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/health")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, HealthStatus{Status: "ok"})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if got.Status != "ok" {
		t.Fatalf("expected ok, got %s", got.Status)
	}
}

func TestHealthReady(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/health/ready")
		assertAuth(t, r, "test-key")
		respondJSON(t, w, http.StatusOK, HealthStatus{Status: "ok"})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.HealthReady(context.Background())
	if err != nil {
		t.Fatalf("HealthReady: %v", err)
	}
	if got.Status != "ok" {
		t.Fatalf("expected ok, got %s", got.Status)
	}
}

func TestListWorkflowRunsByProject(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/workflow-runs")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		if r.URL.Query().Get("status") != "running" {
			t.Fatalf("expected status=running, got %q", r.URL.Query().Get("status"))
		}
		if r.URL.Query().Get("limit") != "15" {
			t.Fatalf("expected limit=15, got %q", r.URL.Query().Get("limit"))
		}
		respondPaginated(t, w, http.StatusOK, []types.WorkflowRun{{ID: "wr-1", WorkflowID: "wf-1", ProjectID: "proj-1", Status: types.WfStatusRunning}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListWorkflowRunsByProject(context.Background(), "proj-1", "running", 15)
	if err != nil {
		t.Fatalf("ListWorkflowRunsByProject: %v", err)
	}
	if len(got) != 1 || got[0].ID != "wr-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestPurgeEventTriggers_DryRun(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/events/purge")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["dry_run"] != true {
			t.Fatalf("expected dry_run=true, got %+v", body["dry_run"])
		}

		respondJSON(t, w, http.StatusOK, map[string]any{"would_delete": 7})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.PurgeEventTriggers(context.Background(), 14, true)
	if err != nil {
		t.Fatalf("PurgeEventTriggers dry-run: %v", err)
	}
	if got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
}

func TestEnvironmentCRUD(t *testing.T) {
	t.Parallel()

	env := types.Environment{ID: "env-1", ProjectID: "proj-1", Name: "Production", Slug: "prod", IsStandard: true}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/environments/env-1":
			respondJSON(t, w, http.StatusOK, env)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/environments":
			respondJSON(t, w, http.StatusCreated, env)
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/environments/env-1":
			respondJSON(t, w, http.StatusOK, env)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/environments/env-1":
			respondJSON(t, w, http.StatusOK, map[string]string{})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/environments/env-1/variables":
			respondJSON(t, w, http.StatusOK, map[string]string{"FOO": "bar"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if got, err := c.GetEnvironment(context.Background(), "env-1"); err != nil || got.Slug != "prod" {
		t.Fatalf("GetEnvironment: %v / got=%+v", err, got)
	}
	if got, err := c.CreateEnvironment(context.Background(), CreateEnvironmentRequest{ProjectID: "proj-1", Name: "Production", Slug: "prod"}); err != nil || got.ID != "env-1" {
		t.Fatalf("CreateEnvironment: %v / got=%+v", err, got)
	}
	name := "Prod"
	if got, err := c.UpdateEnvironment(context.Background(), "env-1", UpdateEnvironmentRequest{Name: &name}); err != nil || got.ID != "env-1" {
		t.Fatalf("UpdateEnvironment: %v / got=%+v", err, got)
	}
	if err := c.DeleteEnvironment(context.Background(), "env-1"); err != nil {
		t.Fatalf("DeleteEnvironment: %v", err)
	}
	vars, err := c.ListEnvironmentVariables(context.Background(), "env-1")
	if err != nil || vars["FOO"] != "bar" {
		t.Fatalf("ListEnvironmentVariables: %v / got=%+v", err, vars)
	}
}

func TestWebhookLifecycle(t *testing.T) {
	t.Parallel()

	hook := types.Webhook{ID: "wh-1", ProjectID: "proj-1", URL: "https://example.com/hook", Events: []string{"run.completed"}, Active: true}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/webhooks":
			respondPaginated(t, w, http.StatusOK, []types.Webhook{hook})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/webhooks/wh-1":
			respondJSON(t, w, http.StatusOK, hook)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/webhooks":
			respondJSON(t, w, http.StatusCreated, hook)
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/webhooks/wh-1":
			respondJSON(t, w, http.StatusOK, hook)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/webhooks/wh-1":
			respondJSON(t, w, http.StatusOK, map[string]string{})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/webhooks/wh-1/deliveries":
			respondPaginated(t, w, http.StatusOK, []types.WebhookDelivery{{ID: "wd-1", WebhookID: "wh-1", Status: "ok"}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/webhooks/wh-1/test":
			respondJSON(t, w, http.StatusOK, TestWebhookResponse{DeliveryID: "wd-9", Status: "queued"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/webhook-deliveries/wd-1/retry":
			respondJSON(t, w, http.StatusOK, types.WebhookDelivery{ID: "wd-1", WebhookID: "wh-1", Status: "queued"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if got, err := c.ListWebhooks(context.Background(), "proj-1"); err != nil || len(got) != 1 {
		t.Fatalf("ListWebhooks: %v / got=%+v", err, got)
	}
	if _, err := c.GetWebhook(context.Background(), "wh-1"); err != nil {
		t.Fatalf("GetWebhook: %v", err)
	}
	if _, err := c.CreateWebhook(context.Background(), CreateWebhookRequest{ProjectID: "proj-1", URL: "https://example.com/hook", Events: []string{"run.completed"}}); err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	url := "https://new.example.com"
	if _, err := c.UpdateWebhook(context.Background(), "wh-1", UpdateWebhookRequest{URL: &url}); err != nil {
		t.Fatalf("UpdateWebhook: %v", err)
	}
	if err := c.DeleteWebhook(context.Background(), "wh-1"); err != nil {
		t.Fatalf("DeleteWebhook: %v", err)
	}
	if got, err := c.ListWebhookDeliveries(context.Background(), "wh-1", 10); err != nil || len(got) != 1 {
		t.Fatalf("ListWebhookDeliveries: %v / got=%+v", err, got)
	}
	if got, err := c.TestWebhook(context.Background(), "wh-1"); err != nil || got.DeliveryID != "wd-9" {
		t.Fatalf("TestWebhook: %v / got=%+v", err, got)
	}
	if got, err := c.RetryWebhookDelivery(context.Background(), "wd-1"); err != nil || got.ID != "wd-1" {
		t.Fatalf("RetryWebhookDelivery: %v / got=%+v", err, got)
	}
}

func TestEventSourceCRUD(t *testing.T) {
	t.Parallel()

	src := types.EventSource{ID: "es-1", ProjectID: "proj-1", Name: "Kafka", Slug: "kafka", Type: "kafka", Enabled: true}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/event-sources":
			respondPaginated(t, w, http.StatusOK, []types.EventSource{src})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/event-sources/es-1":
			respondJSON(t, w, http.StatusOK, src)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/event-sources":
			respondJSON(t, w, http.StatusCreated, src)
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/event-sources/es-1":
			respondJSON(t, w, http.StatusOK, src)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/event-sources/es-1":
			respondJSON(t, w, http.StatusOK, map[string]string{})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if _, err := c.ListEventSources(context.Background(), "proj-1"); err != nil {
		t.Fatalf("ListEventSources: %v", err)
	}
	if _, err := c.GetEventSource(context.Background(), "es-1"); err != nil {
		t.Fatalf("GetEventSource: %v", err)
	}
	if _, err := c.CreateEventSource(context.Background(), CreateEventSourceRequest{ProjectID: "proj-1", Name: "Kafka", Slug: "kafka", Type: "kafka"}); err != nil {
		t.Fatalf("CreateEventSource: %v", err)
	}
	name := "Kafka2"
	if _, err := c.UpdateEventSource(context.Background(), "es-1", UpdateEventSourceRequest{Name: &name}); err != nil {
		t.Fatalf("UpdateEventSource: %v", err)
	}
	if err := c.DeleteEventSource(context.Background(), "es-1"); err != nil {
		t.Fatalf("DeleteEventSource: %v", err)
	}
}

func TestJobGroupLifecycle(t *testing.T) {
	t.Parallel()

	group := types.JobGroup{ID: "jg-1", ProjectID: "proj-1", Name: "ETL", Slug: "etl", JobCount: 3}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/job-groups":
			respondPaginated(t, w, http.StatusOK, []types.JobGroup{group})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/job-groups/jg-1":
			respondJSON(t, w, http.StatusOK, group)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/job-groups":
			respondJSON(t, w, http.StatusCreated, group)
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/job-groups/jg-1":
			respondJSON(t, w, http.StatusOK, group)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/job-groups/jg-1":
			respondJSON(t, w, http.StatusOK, map[string]string{})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/job-groups/jg-1/jobs":
			respondPaginated(t, w, http.StatusOK, []types.Job{{ID: "job-1"}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/job-groups/jg-1/pause":
			respondJSON(t, w, http.StatusOK, map[string]string{})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/job-groups/jg-1/resume":
			respondJSON(t, w, http.StatusOK, map[string]string{})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/job-groups/jg-1/stats":
			respondJSON(t, w, http.StatusOK, types.JobGroupStats{GroupID: "jg-1", JobCount: 3, RunsTotal: 100})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if _, err := c.ListJobGroups(context.Background(), "proj-1"); err != nil {
		t.Fatalf("ListJobGroups: %v", err)
	}
	if _, err := c.GetJobGroup(context.Background(), "jg-1"); err != nil {
		t.Fatalf("GetJobGroup: %v", err)
	}
	if _, err := c.CreateJobGroup(context.Background(), CreateJobGroupRequest{ProjectID: "proj-1", Name: "ETL", Slug: "etl"}); err != nil {
		t.Fatalf("CreateJobGroup: %v", err)
	}
	name := "etl-v2"
	if _, err := c.UpdateJobGroup(context.Background(), "jg-1", UpdateJobGroupRequest{Name: &name}); err != nil {
		t.Fatalf("UpdateJobGroup: %v", err)
	}
	if err := c.DeleteJobGroup(context.Background(), "jg-1"); err != nil {
		t.Fatalf("DeleteJobGroup: %v", err)
	}
	if _, err := c.ListJobsInGroup(context.Background(), "jg-1"); err != nil {
		t.Fatalf("ListJobsInGroup: %v", err)
	}
	if err := c.PauseJobGroup(context.Background(), "jg-1"); err != nil {
		t.Fatalf("PauseJobGroup: %v", err)
	}
	if err := c.ResumeJobGroup(context.Background(), "jg-1"); err != nil {
		t.Fatalf("ResumeJobGroup: %v", err)
	}
	if got, err := c.GetJobGroupStats(context.Background(), "jg-1"); err != nil || got.RunsTotal != 100 {
		t.Fatalf("GetJobGroupStats: %v / got=%+v", err, got)
	}
}

func TestNotificationChannelCRUD(t *testing.T) {
	t.Parallel()

	channel := types.NotificationChannel{ID: "nc-1", ProjectID: "proj-1", Name: "oncall", Type: "slack", Enabled: true}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/notification-channels":
			respondPaginated(t, w, http.StatusOK, []types.NotificationChannel{channel})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/notification-channels/nc-1":
			respondJSON(t, w, http.StatusOK, channel)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/notification-channels":
			respondJSON(t, w, http.StatusCreated, channel)
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/notification-channels/nc-1":
			respondJSON(t, w, http.StatusOK, channel)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/notification-channels/nc-1":
			respondJSON(t, w, http.StatusOK, map[string]string{})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if _, err := c.ListNotificationChannels(context.Background(), "proj-1"); err != nil {
		t.Fatalf("ListNotificationChannels: %v", err)
	}
	if _, err := c.GetNotificationChannel(context.Background(), "nc-1"); err != nil {
		t.Fatalf("GetNotificationChannel: %v", err)
	}
	if _, err := c.CreateNotificationChannel(context.Background(), CreateNotificationChannelRequest{
		ProjectID: "proj-1", Name: "oncall", Type: "slack", Config: json.RawMessage(`{"webhook_url":"https://hooks.example"}`),
	}); err != nil {
		t.Fatalf("CreateNotificationChannel: %v", err)
	}
	name := "oncall-v2"
	if _, err := c.UpdateNotificationChannel(context.Background(), "nc-1", UpdateNotificationChannelRequest{Name: &name}); err != nil {
		t.Fatalf("UpdateNotificationChannel: %v", err)
	}
	if err := c.DeleteNotificationChannel(context.Background(), "nc-1"); err != nil {
		t.Fatalf("DeleteNotificationChannel: %v", err)
	}
}

func TestLogDrainCRUD(t *testing.T) {
	t.Parallel()

	drain := types.LogDrain{ID: "ld-1", ProjectID: "proj-1", Name: "datadog-prod", Type: "datadog", Enabled: true}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/log-drains":
			respondPaginated(t, w, http.StatusOK, []types.LogDrain{drain})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/log-drains/ld-1":
			respondJSON(t, w, http.StatusOK, drain)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/log-drains":
			respondJSON(t, w, http.StatusCreated, drain)
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/log-drains/ld-1":
			respondJSON(t, w, http.StatusOK, drain)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/log-drains/ld-1":
			respondJSON(t, w, http.StatusOK, map[string]string{})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if _, err := c.ListLogDrains(context.Background(), "proj-1"); err != nil {
		t.Fatalf("ListLogDrains: %v", err)
	}
	if _, err := c.GetLogDrain(context.Background(), "ld-1"); err != nil {
		t.Fatalf("GetLogDrain: %v", err)
	}
	if _, err := c.CreateLogDrain(context.Background(), CreateLogDrainRequest{
		ProjectID: "proj-1", Name: "datadog-prod", Type: "datadog", Config: json.RawMessage(`{"api_key":"x"}`),
	}); err != nil {
		t.Fatalf("CreateLogDrain: %v", err)
	}
	name := "datadog-prod-v2"
	if _, err := c.UpdateLogDrain(context.Background(), "ld-1", UpdateLogDrainRequest{Name: &name}); err != nil {
		t.Fatalf("UpdateLogDrain: %v", err)
	}
	if err := c.DeleteLogDrain(context.Background(), "ld-1"); err != nil {
		t.Fatalf("DeleteLogDrain: %v", err)
	}
}

// Test helpers.

func mustClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := New(baseURL, "test-key", 10*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func assertMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.Method != want {
		t.Fatalf("expected method %s, got %s", want, r.Method)
	}
}

func assertPath(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.URL.Path != want {
		t.Fatalf("expected path %s, got %s", want, r.URL.Path)
	}
}

func assertAuth(t *testing.T, r *http.Request, key string) {
	t.Helper()
	want := "Bearer " + key
	if r.Header.Get("Authorization") != want {
		t.Fatalf("expected auth %q, got %q", want, r.Header.Get("Authorization"))
	}
}

func assertContentType(t *testing.T, r *http.Request) {
	t.Helper()
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
}

func respondJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

// respondPaginated wraps data in the PaginatedResponse envelope for list endpoints.
func respondPaginated(t *testing.T, w http.ResponseWriter, status int, data any) {
	t.Helper()
	respondJSON(t, w, status, paginatedResponse{
		Data:    mustMarshal(t, data),
		HasMore: false,
	})
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// Phase 0: Deployment API tests.

func TestCreateDeploymentVersion(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/deployments")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req CreateDeploymentVersionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.ProjectID != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", req.ProjectID)
		}
		if req.Environment != "production" {
			t.Fatalf("expected environment=production, got %q", req.Environment)
		}
		if req.Checksum == "" {
			t.Fatal("expected checksum to be set")
		}

		respondJSON(t, w, http.StatusOK, DeploymentVersion{
			ID:          "dep-1",
			ProjectID:   req.ProjectID,
			Environment: req.Environment,
			Status:      "pending",
			Checksum:    req.Checksum,
			CreatedAt:   now,
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.CreateDeploymentVersion(context.Background(), CreateDeploymentVersionRequest{
		ProjectID:   "proj-1",
		Environment: "production",
		Runtime:     "node",
		Checksum:    "sha256:abc123",
	})
	if err != nil {
		t.Fatalf("CreateDeploymentVersion: %v", err)
	}
	if got.ID != "dep-1" {
		t.Fatalf("expected dep-1, got %s", got.ID)
	}
}

func TestCreateDeploymentVersion_MissingFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"project_id is required"}`))
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.CreateDeploymentVersion(context.Background(), CreateDeploymentVersionRequest{})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("error should contain status code: %v", err)
	}
}

func TestFinalizeDeployment(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/deployments/dep-1/finalize")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req FinalizeDeploymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.ProjectID != "proj-1" || req.Environment != "production" {
			t.Fatalf("unexpected request: %+v", req)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	err := c.FinalizeDeployment(context.Background(), "dep-1", FinalizeDeploymentRequest{
		ProjectID:   "proj-1",
		Environment: "production",
	})
	if err != nil {
		t.Fatalf("FinalizeDeployment: %v", err)
	}
}

func TestPromoteDeployment(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/deployments/dep-1/promote")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req PromoteDeploymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.ProjectID != "proj-1" || req.Environment != "production" {
			t.Fatalf("unexpected request: %+v", req)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	err := c.PromoteDeployment(context.Background(), "dep-1", PromoteDeploymentRequest{ProjectID: "proj-1", Environment: "production"})
	if err != nil {
		t.Fatalf("PromoteDeployment: %v", err)
	}
}

func TestRollbackDeployment(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/deployments/dep-1/rollback")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req RollbackDeploymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.ProjectID != "proj-1" || req.Environment != "production" {
			t.Fatalf("unexpected request: %+v", req)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	err := c.RollbackDeployment(context.Background(), "dep-1", RollbackDeploymentRequest{ProjectID: "proj-1", Environment: "production"})
	if err != nil {
		t.Fatalf("RollbackDeployment: %v", err)
	}
}

func TestListDeployments(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/deployments")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		respondPaginated(t, w, http.StatusOK, []DeploymentVersion{
			{ID: "dep-1", ProjectID: "proj-1", Environment: "production", Status: "active", CreatedAt: now},
			{ID: "dep-2", ProjectID: "proj-1", Environment: "staging", Status: "pending", CreatedAt: now},
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListDeployments(context.Background(), "proj-1", 0)
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 deployments, got %d", len(got))
	}
	if got[0].ID != "dep-1" {
		t.Fatalf("expected dep-1, got %s", got[0].ID)
	}
}

func TestListServerSecrets(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/secrets")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		if r.URL.Query().Get("environment") != "production" {
			t.Fatalf("expected environment=production, got %q", r.URL.Query().Get("environment"))
		}
		respondPaginated(t, w, http.StatusOK, []ServerSecret{
			{ID: "sec-1", ProjectID: "proj-1", SecretKey: "DB_PASSWORD", Environment: "production", CreatedAt: now, UpdatedAt: now},
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListServerSecrets(context.Background(), "proj-1", "production")
	if err != nil {
		t.Fatalf("ListServerSecrets: %v", err)
	}
	if len(got) != 1 || got[0].ID != "sec-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestCreateServerSecret(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/secrets")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req CreateServerSecretRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.ProjectID != "proj-1" || req.SecretKey != "API_TOKEN" || req.SecretValue != "secret123" {
			t.Fatalf("unexpected request: %+v", req)
		}

		respondJSON(t, w, http.StatusOK, ServerSecret{
			ID:          "sec-1",
			ProjectID:   req.ProjectID,
			SecretKey:   req.SecretKey,
			Environment: req.Environment,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.CreateServerSecret(context.Background(), CreateServerSecretRequest{
		ProjectID:   "proj-1",
		SecretKey:   "API_TOKEN",
		SecretValue: "secret123",
		Environment: "production",
	})
	if err != nil {
		t.Fatalf("CreateServerSecret: %v", err)
	}
	if got.ID != "sec-1" {
		t.Fatalf("expected sec-1, got %s", got.ID)
	}
}

func TestDeleteServerSecret(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/v1/secrets/sec-1")
		assertAuth(t, r, "test-key")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if err := c.DeleteServerSecret(context.Background(), "sec-1"); err != nil {
		t.Fatalf("DeleteServerSecret: %v", err)
	}
}

func TestGetPerformanceAnalytics(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/analytics/performance")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		if r.URL.Query().Get("period_hours") != "72" {
			t.Fatalf("expected period_hours=72, got %q", r.URL.Query().Get("period_hours"))
		}
		respondJSON(t, w, http.StatusOK, PerformanceAnalytics{
			SlowestJobs: []JobPerformance{
				{JobID: "job-1", JobSlug: "process-payment", TotalRuns: 100, FailedRuns: 5, AvgDurationSecs: 1.5, P95DurationSecs: 2.3},
			},
			Throughput: ThroughputStats{Completed: 95, Failed: 5, PeriodHours: 72},
			HealthSummary: HealthSummary{
				TotalJobs:       10,
				ActiveJobs:      8,
				SuccessRate:     0.95,
				AvgDurationSecs: 1.25,
				QueueDepth:      3,
			},
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.GetPerformanceAnalytics(context.Background(), "proj-1", 72)
	if err != nil {
		t.Fatalf("GetPerformanceAnalytics: %v", err)
	}
	if got.Throughput.PeriodHours != 72 || len(got.SlowestJobs) != 1 || got.SlowestJobs[0].JobID != "job-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestListMembers(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/members")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		respondPaginated(t, w, http.StatusOK, []ProjectMember{
			{ID: "mem-1", ProjectID: "proj-1", UserID: "user-1", RoleID: "role-admin", GrantedBy: "owner-1", CreatedAt: now},
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListMembers(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(got) != 1 || got[0].ID != "mem-1" || got[0].UserID != "user-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestListAuditEvents(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/audit-events")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		if r.URL.Query().Get("actor_id") != "actor-1" {
			t.Fatalf("expected actor_id=actor-1, got %q", r.URL.Query().Get("actor_id"))
		}
		if r.URL.Query().Get("resource_type") != "job" || r.URL.Query().Get("resource_id") != "job-1" {
			t.Fatalf("unexpected resource filters: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("order") != "asc" {
			t.Fatalf("expected order=asc, got %q", r.URL.Query().Get("order"))
		}
		if r.URL.Query().Get("from") == "" || r.URL.Query().Get("to") == "" {
			t.Fatalf("expected from/to filters, got %s", r.URL.RawQuery)
		}
		respondPaginated(t, w, http.StatusOK, []AuditEvent{
			{ID: "ae-1", ProjectID: "proj-1", ActorID: "actor-1", ActorType: "user", Action: "job.created", ResourceType: "job", ResourceID: "job-1", CreatedAt: now},
		})
	}))
	defer srv.Close()

	from := now.Add(-time.Hour)
	to := now.Add(time.Hour)
	c := mustClient(t, srv.URL)
	got, err := c.ListAuditEvents(context.Background(), ListAuditEventsParams{
		ProjectID:    "proj-1",
		ActorID:      "actor-1",
		ResourceType: "job",
		ResourceID:   "job-1",
		Limit:        10,
		From:         &from,
		To:           &to,
		Order:        "asc",
	})
	if err != nil {
		t.Fatalf("ListAuditEvents: %v", err)
	}
	if len(got) != 1 || got[0].ID != "ae-1" || got[0].ActorID != "actor-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestAddMember(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/members")
		assertAuth(t, r, "test-key")
		assertContentType(t, r)

		var req AssignMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.UserID != "user-1" || req.RoleID != "role-admin" {
			t.Fatalf("unexpected request: %+v", req)
		}

		respondJSON(t, w, http.StatusOK, ProjectMember{
			ID:        "mem-1",
			ProjectID: "proj-1",
			UserID:    req.UserID,
			RoleID:    req.RoleID,
			GrantedBy: "owner-1",
			CreatedAt: now,
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.AddMember(context.Background(), AssignMemberRequest{
		UserID: "user-1",
		RoleID: "role-admin",
	})
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if got.ID != "mem-1" || got.UserID != "user-1" || got.RoleID != "role-admin" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestRemoveMember(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodDelete)
		assertPath(t, r, "/v1/members/user-1")
		assertAuth(t, r, "test-key")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if err := c.RemoveMember(context.Background(), "user-1"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
}

func TestListRoles(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/roles")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		respondPaginated(t, w, http.StatusOK, []ProjectRole{
			{ID: "role-1", Name: "admin", Description: "Full access"},
			{ID: "role-2", Name: "viewer", Description: "Read-only access"},
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListRoles(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(got) != 2 || got[0].ID != "role-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestDoListAllJSON_MultiplePages(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/runs/run-1/events")
		assertAuth(t, r, "test-key")

		n := callCount.Add(1)
		cursor := r.URL.Query().Get("cursor")

		if n == 1 && cursor == "" {
			nextCursor := "page2"
			respondJSON(t, w, http.StatusOK, paginatedResponse{
				Data:       mustMarshal(t, []types.RunEvent{{ID: "evt-1"}, {ID: "evt-2"}}),
				HasMore:    true,
				NextCursor: &nextCursor,
			})
		} else if cursor == "page2" {
			respondJSON(t, w, http.StatusOK, paginatedResponse{
				Data:    mustMarshal(t, []types.RunEvent{{ID: "evt-3"}}),
				HasMore: false,
			})
		} else {
			t.Fatalf("unexpected call: n=%d cursor=%q", n, cursor)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListRunEvents(context.Background(), "run-1", "", "")
	if err != nil {
		t.Fatalf("ListRunEvents: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 events, got %d", len(got))
	}
	if got[0].ID != "evt-1" || got[1].ID != "evt-2" || got[2].ID != "evt-3" {
		t.Fatalf("unexpected events: %+v", got)
	}
}

func TestDoListAllJSON_SinglePage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertAuth(t, r, "test-key")
		respondPaginated(t, w, http.StatusOK, []types.RunEvent{{ID: "evt-1"}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListRunEvents(context.Background(), "run-1", "", "")
	if err != nil {
		t.Fatalf("ListRunEvents: %v", err)
	}
	if len(got) != 1 || got[0].ID != "evt-1" {
		t.Fatalf("unexpected events: %+v", got)
	}
}

func TestDoListAllJSON_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertAuth(t, r, "test-key")
		respondPaginated(t, w, http.StatusOK, []types.RunEvent{})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.ListRunEvents(context.Background(), "run-1", "", "")
	if err != nil {
		t.Fatalf("ListRunEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 events, got %d", len(got))
	}
}

func TestDoListAllJSON_TruncationWarning(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertAuth(t, r, "test-key")

		n := callCount.Add(1)
		cursor := fmt.Sprintf("page%d", n+1)
		respondJSON(t, w, http.StatusOK, paginatedResponse{
			Data:       mustMarshal(t, []types.RunEvent{{ID: fmt.Sprintf("evt-%d", n)}}),
			HasMore:    true,
			NextCursor: &cursor,
		})
	}))
	defer srv.Close()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	c := mustClient(t, srv.URL)
	got, err := c.ListRunEvents(context.Background(), "run-1", "", "")

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	stderrOutput := buf.String()

	if err != nil {
		t.Fatalf("ListRunEvents: %v", err)
	}
	// maxPages=100, each page has 1 event
	if len(got) != 100 {
		t.Fatalf("expected 100 events, got %d", len(got))
	}
	if !strings.Contains(stderrOutput, "truncated") {
		t.Fatalf("expected truncation warning on stderr, got: %q", stderrOutput)
	}
}

func TestDoListAllJSON_NoWarningOnNormalPagination(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertAuth(t, r, "test-key")

		n := callCount.Add(1)
		if n == 1 {
			cursor := "page2"
			respondJSON(t, w, http.StatusOK, paginatedResponse{
				Data:       mustMarshal(t, []types.RunEvent{{ID: "evt-1"}}),
				HasMore:    true,
				NextCursor: &cursor,
			})
		} else {
			respondJSON(t, w, http.StatusOK, paginatedResponse{
				Data:    mustMarshal(t, []types.RunEvent{{ID: "evt-2"}}),
				HasMore: false,
			})
		}
	}))
	defer srv.Close()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	c := mustClient(t, srv.URL)
	got, err := c.ListRunEvents(context.Background(), "run-1", "", "")

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	stderrOutput := buf.String()

	if err != nil {
		t.Fatalf("ListRunEvents: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if strings.Contains(stderrOutput, "truncated") {
		t.Fatalf("should not warn on normal pagination, got: %q", stderrOutput)
	}
}

func TestVerifyAuditChain_Passed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/audit-events/verify")
		assertAuth(t, r, "test-key")
		if r.URL.Query().Get("project_id") != "proj-1" {
			t.Fatalf("expected project_id=proj-1, got %q", r.URL.Query().Get("project_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"project_id":"proj-1","valid":true,"events_checked":42,"first_event_id":"ae-a","last_event_id":"ae-z"}`))
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.VerifyAuditChain(context.Background(), VerifyAuditChainParams{ProjectID: "proj-1"})
	if err != nil {
		t.Fatalf("VerifyAuditChain: %v", err)
	}
	if !got.Valid {
		t.Fatalf("expected valid=true, got %+v", got)
	}
	if got.EventsChecked != 42 {
		t.Fatalf("expected 42 events checked, got %d", got.EventsChecked)
	}
	if got.ProjectID != "proj-1" {
		t.Fatalf("unexpected project_id: %q", got.ProjectID)
	}
}

func TestVerifyAuditChain_Failed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/audit-events/verify")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"project_id":"proj-2","valid":false,"events_checked":17,"broken_at_id":"ae-bad","error":"hmac mismatch"}`))
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.VerifyAuditChain(context.Background(), VerifyAuditChainParams{ProjectID: "proj-2"})
	if err != nil {
		t.Fatalf("VerifyAuditChain: %v", err)
	}
	if got.Valid {
		t.Fatalf("expected valid=false, got %+v", got)
	}
	if got.BrokenAtID != "ae-bad" || got.Error != "hmac mismatch" {
		t.Fatalf("unexpected break info: %+v", got)
	}
}

func TestVerifyAuditChain_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	_, err := c.VerifyAuditChain(context.Background(), VerifyAuditChainParams{ProjectID: "proj-3"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestVerifyAuditChain_SincePassedAsQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("since") == "" {
			t.Fatalf("expected since query param, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"project_id":"p","valid":true,"events_checked":0}`))
	}))
	defer srv.Close()

	since := time.Now().Add(-time.Hour).UTC()
	c := mustClient(t, srv.URL)
	if _, err := c.VerifyAuditChain(context.Background(), VerifyAuditChainParams{ProjectID: "p", Since: &since}); err != nil {
		t.Fatalf("VerifyAuditChain: %v", err)
	}
}

func TestCloneJob(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/jobs/job-1/clone")
		respondJSON(t, w, http.StatusOK, types.Job{ID: "job-2", Slug: "job-2"})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.CloneJob(context.Background(), "job-1", CloneJobRequest{Name: "Clone", Slug: "job-2"})
	if err != nil {
		t.Fatalf("CloneJob: %v", err)
	}
	if got.Slug != "job-2" {
		t.Fatalf("unexpected slug: %q", got.Slug)
	}
}

func TestGetJobHealth(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/jobs/job-1/health")
		respondJSON(t, w, http.StatusOK, types.JobHealth{
			JobID: "job-1", Status: "healthy", SuccessRate: 0.99, P95DurationMS: 1200,
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.GetJobHealth(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("GetJobHealth: %v", err)
	}
	if got.Status != "healthy" || got.SuccessRate != 0.99 {
		t.Fatalf("unexpected health: %+v", got)
	}
}

func TestListJobDependencies(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/jobs/job-1/dependencies")
		respondPaginated(t, w, http.StatusOK, []types.JobDependency{{ID: "dep-1", JobID: "job-1", DependsOn: "job-0", Type: "success"}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	deps, err := c.ListJobDependencies(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("ListJobDependencies: %v", err)
	}
	if len(deps) != 1 || deps[0].DependsOn != "job-0" {
		t.Fatalf("unexpected deps: %+v", deps)
	}
}

func TestAddJobDependency(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/jobs/job-1/dependencies")
		var body AddJobDependencyRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.DependsOn != "job-0" || body.Type != "success" {
			t.Fatalf("unexpected body: %+v", body)
		}
		respondJSON(t, w, http.StatusOK, types.JobDependency{ID: "dep-9", JobID: "job-1", DependsOn: body.DependsOn, Type: body.Type})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	dep, err := c.AddJobDependency(context.Background(), "job-1", AddJobDependencyRequest{DependsOn: "job-0", Type: "success"})
	if err != nil {
		t.Fatalf("AddJobDependency: %v", err)
	}
	if dep.ID != "dep-9" {
		t.Fatalf("unexpected dep: %+v", dep)
	}
}

func TestBatchUpdateJobs(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/jobs/batch")
		respondJSON(t, w, http.StatusOK, map[string]any{
			"updated": []string{"job-1", "job-2"},
			"failed": []map[string]string{
				{"id": "job-3", "reason": "not found"},
			},
		})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	resp, err := c.BatchUpdateJobs(context.Background(), BatchUpdateJobsRequest{Updates: []BatchJobUpdate{{ID: "job-1"}}})
	if err != nil {
		t.Fatalf("BatchUpdateJobs: %v", err)
	}
	if len(resp.Updated) != 2 || len(resp.Failed) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCloneWorkflow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/workflows/wf-1/clone")
		respondJSON(t, w, http.StatusOK, map[string]any{"id": "wf-2", "slug": "wf-2"})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	wf, err := c.CloneWorkflow(context.Background(), "wf-1", CloneWorkflowRequest{Name: "Clone", Slug: "wf-2"})
	if err != nil {
		t.Fatalf("CloneWorkflow: %v", err)
	}
	if wf.Slug != "wf-2" {
		t.Fatalf("unexpected slug: %q", wf.Slug)
	}
}

func TestDryRunWorkflow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/workflows/wf-1/dry-run")
		respondJSON(t, w, http.StatusOK, map[string]any{"ok": true})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	out, err := c.DryRunWorkflow(context.Background(), "wf-1", json.RawMessage(`{"x":1}`))
	if err != nil {
		t.Fatalf("DryRunWorkflow: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected payload, got empty")
	}
}

func TestListWorkflowVersions(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/workflows/wf-1/versions")
		respondPaginated(t, w, http.StatusOK, []types.WorkflowVersion{{WorkflowID: "wf-1", Version: 1}})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	versions, err := c.ListWorkflowVersions(context.Background(), "wf-1")
	if err != nil {
		t.Fatalf("ListWorkflowVersions: %v", err)
	}
	if len(versions) != 1 || versions[0].Version != 1 {
		t.Fatalf("unexpected versions: %+v", versions)
	}
}

func TestDiffWorkflowVersions(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertPath(t, r, "/v1/workflows/wf-1/diff")
		if r.URL.Query().Get("from") != "1" || r.URL.Query().Get("to") != "2" {
			t.Fatalf("unexpected query: %q", r.URL.RawQuery)
		}
		respondJSON(t, w, http.StatusOK, types.WorkflowDiff{WorkflowID: "wf-1", From: 1, To: 2})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	diff, err := c.DiffWorkflowVersions(context.Background(), "wf-1", 1, 2)
	if err != nil {
		t.Fatalf("DiffWorkflowVersions: %v", err)
	}
	if diff.From != 1 || diff.To != 2 {
		t.Fatalf("unexpected diff: %+v", diff)
	}
}

func TestWorkflowPolicy(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/v1/workflows/wf-1/policy")
		switch r.Method {
		case http.MethodGet:
			respondJSON(t, w, http.StatusOK, types.WorkflowPolicy{WorkflowID: "wf-1", Policy: json.RawMessage(`{"max":1}`)})
		case http.MethodPut:
			respondJSON(t, w, http.StatusOK, types.WorkflowPolicy{WorkflowID: "wf-1", Policy: json.RawMessage(`{"max":2}`)})
		default:
			t.Fatalf("unexpected method %q", r.Method)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	got, err := c.GetWorkflowPolicy(context.Background(), "wf-1")
	if err != nil {
		t.Fatalf("GetWorkflowPolicy: %v", err)
	}
	if got.WorkflowID != "wf-1" {
		t.Fatalf("unexpected: %+v", got)
	}
	updated, err := c.SetWorkflowPolicy(context.Background(), "wf-1", json.RawMessage(`{"max":2}`))
	if err != nil {
		t.Fatalf("SetWorkflowPolicy: %v", err)
	}
	if updated.WorkflowID != "wf-1" {
		t.Fatalf("unexpected updated: %+v", updated)
	}
}

func TestPauseResumeRetryWorkflowRun(t *testing.T) {
	t.Parallel()

	expected := map[string]string{
		"/v1/workflow-runs/wfr-1/pause":  "paused",
		"/v1/workflow-runs/wfr-1/resume": "running",
		"/v1/workflow-runs/wfr-1/retry":  "queued",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		status, ok := expected[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		respondJSON(t, w, http.StatusOK, map[string]any{"id": "wfr-1", "status": status})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if _, err := c.PauseWorkflowRun(context.Background(), "wfr-1"); err != nil {
		t.Fatalf("PauseWorkflowRun: %v", err)
	}
	if _, err := c.ResumeWorkflowRun(context.Background(), "wfr-1"); err != nil {
		t.Fatalf("ResumeWorkflowRun: %v", err)
	}
	if _, err := c.RetryWorkflowRun(context.Background(), "wfr-1"); err != nil {
		t.Fatalf("RetryWorkflowRun: %v", err)
	}
}

func TestWorkflowStepActions(t *testing.T) {
	t.Parallel()

	expected := map[string]struct{}{
		"/v1/workflow-runs/wfr-1/steps/build/approve":        {},
		"/v1/workflow-runs/wfr-1/steps/build/retry":          {},
		"/v1/workflow-runs/wfr-1/steps/build/skip":           {},
		"/v1/workflow-runs/wfr-1/steps/build/force-complete": {},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		if _, ok := expected[r.URL.Path]; !ok {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		respondJSON(t, w, http.StatusOK, map[string]string{})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if err := c.ApproveWorkflowStep(context.Background(), "wfr-1", "build"); err != nil {
		t.Fatalf("ApproveWorkflowStep: %v", err)
	}
	if err := c.RetryWorkflowStep(context.Background(), "wfr-1", "build"); err != nil {
		t.Fatalf("RetryWorkflowStep: %v", err)
	}
	if err := c.SkipWorkflowStep(context.Background(), "wfr-1", "build"); err != nil {
		t.Fatalf("SkipWorkflowStep: %v", err)
	}
	if err := c.ForceCompleteWorkflowStep(context.Background(), "wfr-1", "build"); err != nil {
		t.Fatalf("ForceCompleteWorkflowStep: %v", err)
	}
}

func TestRescheduleRun(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPath(t, r, "/v1/runs/run-1/reschedule")
		var body RescheduleRunRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !body.ScheduledAt.Equal(at) {
			t.Fatalf("expected %v, got %v", at, body.ScheduledAt)
		}
		respondJSON(t, w, http.StatusOK, types.JobRun{ID: "run-1", Status: "delayed"})
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	run, err := c.RescheduleRun(context.Background(), "run-1", at)
	if err != nil {
		t.Fatalf("RescheduleRun: %v", err)
	}
	if run.ID != "run-1" {
		t.Fatalf("unexpected: %+v", run)
	}
}

func TestDLQListAndReplay(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/runs/dlq":
			if r.URL.Query().Get("project_id") != "proj-1" {
				t.Fatalf("expected project_id query, got %q", r.URL.RawQuery)
			}
			respondPaginated(t, w, http.StatusOK, []types.DLQRun{{ID: "dlq-1", JobID: "job-1", Reason: "max attempts"}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/dlq/dlq-1/replay":
			respondJSON(t, w, http.StatusOK, types.JobRun{ID: "run-2", Status: "queued"})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	items, err := c.ListDLQ(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListDLQ: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected items: %+v", items)
	}
	run, err := c.ReplayDLQ(context.Background(), "dlq-1")
	if err != nil {
		t.Fatalf("ReplayDLQ: %v", err)
	}
	if run.ID != "run-2" {
		t.Fatalf("unexpected replay: %+v", run)
	}
}

func TestRunTelemetryEndpoints(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/runs/run-1/outputs":
			respondPaginated(t, w, http.StatusOK, []types.RunOutput{{ID: "out-1", RunID: "run-1", Key: "result"}})
		case "/v1/runs/run-1/tool-calls":
			respondPaginated(t, w, http.StatusOK, []types.RunToolCall{{ID: "tc-1", RunID: "run-1", Tool: "fetch"}})
		case "/v1/runs/run-1/usage":
			respondJSON(t, w, http.StatusOK, types.RunUsage{RunID: "run-1", DurationMS: 4200, CostUSD: 0.12})
		case "/v1/runs/run-1/checkpoints":
			respondPaginated(t, w, http.StatusOK, []types.RunCheckpoint{{ID: "cp-1", RunID: "run-1", Name: "phase-1"}})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if outputs, err := c.ListRunOutputs(context.Background(), "run-1"); err != nil || len(outputs) != 1 {
		t.Fatalf("ListRunOutputs: %v len=%d", err, len(outputs))
	}
	if calls, err := c.ListRunToolCalls(context.Background(), "run-1"); err != nil || len(calls) != 1 {
		t.Fatalf("ListRunToolCalls: %v len=%d", err, len(calls))
	}
	usage, err := c.GetRunUsage(context.Background(), "run-1")
	if err != nil || usage.DurationMS != 4200 {
		t.Fatalf("GetRunUsage: %v %+v", err, usage)
	}
	if cps, err := c.ListRunCheckpoints(context.Background(), "run-1"); err != nil || len(cps) != 1 {
		t.Fatalf("ListRunCheckpoints: %v len=%d", err, len(cps))
	}
}

func TestUsageEndpoints(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/billing/usage":
			assertMethod(t, r, http.MethodGet)
			respondJSON(t, w, http.StatusOK, types.UsagePeriod{Runs: 42, CostUSD: 12.34})
		case "/v1/billing/usage/history":
			assertMethod(t, r, http.MethodGet)
			respondPaginated(t, w, http.StatusOK, []types.UsagePeriod{{Runs: 10}, {Runs: 20}})
		case "/v1/billing/usage/forecast":
			assertMethod(t, r, http.MethodGet)
			respondJSON(t, w, http.StatusOK, types.UsagePeriod{Runs: 99, CostUSD: 50})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	cur, err := c.GetCurrentUsage(context.Background())
	if err != nil || cur.Runs != 42 {
		t.Fatalf("GetCurrentUsage: %v %+v", err, cur)
	}
	hist, err := c.GetUsageHistory(context.Background())
	if err != nil || len(hist) != 2 {
		t.Fatalf("GetUsageHistory: %v len=%d", err, len(hist))
	}
	fc, err := c.GetUsageForecast(context.Background())
	if err != nil || fc.Runs != 99 {
		t.Fatalf("GetUsageForecast: %v %+v", err, fc)
	}
}

func TestAnalyticsEndpoints(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/analytics/costs":
			assertMethod(t, r, http.MethodGet)
			if got := r.URL.Query().Get("project_id"); got != "proj-1" {
				t.Fatalf("project_id: %q", got)
			}
			if got := r.URL.Query().Get("period_hours"); got != "168" {
				t.Fatalf("period_hours: %q", got)
			}
			respondJSON(t, w, http.StatusOK, types.CostsAnalytics{TotalUSD: 99.95, PeriodHours: 168})
		case "/v1/analytics/reliability":
			assertMethod(t, r, http.MethodGet)
			respondJSON(t, w, http.StatusOK, types.ReliabilityAnalytics{SuccessRate: 0.97, PeriodHours: 168})
		case "/v1/analytics/top-failing":
			assertMethod(t, r, http.MethodGet)
			if got := r.URL.Query().Get("limit"); got != "5" {
				t.Fatalf("limit: %q", got)
			}
			respondPaginated(t, w, http.StatusOK, []types.TopFailingJob{{JobSlug: "x", FailureRate: 0.5}})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	costs, err := c.GetCostsAnalytics(context.Background(), "proj-1", 168)
	if err != nil || costs.TotalUSD != 99.95 {
		t.Fatalf("GetCostsAnalytics: %v %+v", err, costs)
	}
	rel, err := c.GetReliabilityAnalytics(context.Background(), "proj-1", 168)
	if err != nil || rel.SuccessRate < 0.96 {
		t.Fatalf("GetReliabilityAnalytics: %v %+v", err, rel)
	}
	top, err := c.ListTopFailingJobs(context.Background(), "proj-1", 168, 5)
	if err != nil || len(top) != 1 {
		t.Fatalf("ListTopFailingJobs: %v len=%d", err, len(top))
	}
}

func TestTeamPolicyEndpoints(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/team/policies" && r.Method == http.MethodGet:
			respondPaginated(t, w, http.StatusOK, []types.TeamPolicy{{ID: "pol-1", Name: "default"}})
		case r.URL.Path == "/v1/team/policies" && r.Method == http.MethodPost:
			var req CreateTeamPolicyRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Name != "default" || len(req.Permissions) != 1 {
				t.Fatalf("body: %+v", req)
			}
			respondJSON(t, w, http.StatusOK, types.TeamPolicy{ID: "pol-2", Name: req.Name, Permissions: req.Permissions})
		case r.URL.Path == "/v1/team/policies/pol-2" && r.Method == http.MethodDelete:
			respondJSON(t, w, http.StatusOK, map[string]string{"ok": "true"})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := mustClient(t, srv.URL)
	policies, err := c.ListTeamPolicies(context.Background())
	if err != nil || len(policies) != 1 {
		t.Fatalf("ListTeamPolicies: %v len=%d", err, len(policies))
	}
	pol, err := c.CreateTeamPolicy(context.Background(), CreateTeamPolicyRequest{Name: "default", Permissions: []string{"jobs:read"}})
	if err != nil || pol.ID != "pol-2" {
		t.Fatalf("CreateTeamPolicy: %v %+v", err, pol)
	}
	if err := c.DeleteTeamPolicy(context.Background(), "pol-2"); err != nil {
		t.Fatalf("DeleteTeamPolicy: %v", err)
	}
}
