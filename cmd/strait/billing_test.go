package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestBillingSpendingLimitGet_Success(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"limit_microusd": int64(5000000), "action": "block"}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/spending-limit": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "org_id", "org-abc")
			respondJSON(t, w, http.StatusOK, payload)
		},
	})

	state := newTestState(t, srv)
	cmd := newBillingSpendingLimitGetCommand(state)
	cmd.SetArgs([]string{"--org", "org-abc"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBillingSpendingLimitSet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"PUT /v1/spending-limit": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "org_id", "org-abc")
			var got struct {
				LimitMicroUSD int64  `json:"limit_microusd"`
				Action        string `json:"action"`
			}
			readJSONBody(t, r, &got)
			if got.LimitMicroUSD != 9000000 {
				t.Errorf("limit_microusd: got %d, want 9000000", got.LimitMicroUSD)
			}
			if got.Action != "notify" {
				t.Errorf("action: got %q, want %q", got.Action, "notify")
			}
			respondJSON(t, w, http.StatusOK, map[string]any{"limit_microusd": got.LimitMicroUSD, "action": got.Action})
		},
	})

	state := newTestState(t, srv)
	cmd := newBillingSpendingLimitSetCommand(state)
	cmd.SetArgs([]string{"--org", "org-abc", "--limit-microusd", "9000000", "--action", "notify"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBillingSpendingLimitSet_RequiresAction(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newBillingSpendingLimitSetCommand(state)
	cmd.SetArgs([]string{"--org", "org-abc", "--limit-microusd", "1000000"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--action is required") {
		t.Fatalf("expected --action error, got: %v", err)
	}
}

func TestBillingProjectBudgetGet_Success(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"project_id": "proj-test", "budget_microusd": int64(1000000), "action": "block"}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/project-budget": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondJSON(t, w, http.StatusOK, payload)
		},
	})

	state := newTestState(t, srv)
	cmd := newBillingProjectBudgetGetCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "proj-test") {
		t.Fatalf("expected project_id in output: %s", out)
	}
}

func TestBillingProjectBudgetSet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"PUT /v1/project-budget": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			var got struct {
				ProjectID      string `json:"project_id"`
				BudgetMicroUSD int64  `json:"budget_microusd"`
				Action         string `json:"action"`
			}
			readJSONBody(t, r, &got)
			if got.ProjectID != "proj-test" {
				t.Errorf("project_id: got %q, want %q", got.ProjectID, "proj-test")
			}
			if got.BudgetMicroUSD != 2000000 {
				t.Errorf("budget_microusd: got %d, want 2000000", got.BudgetMicroUSD)
			}
			if got.Action != "notify" {
				t.Errorf("action: got %q, want %q", got.Action, "notify")
			}
			respondJSON(t, w, http.StatusOK, map[string]any{
				"project_id":      got.ProjectID,
				"budget_microusd": got.BudgetMicroUSD,
				"action":          got.Action,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newBillingProjectBudgetSetCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--budget-microusd", "2000000", "--action", "notify"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBillingProjectBudgetSet_RequiresAction(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newBillingProjectBudgetSetCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test", "--budget-microusd", "500000"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--action is required") {
		t.Fatalf("expected --action error, got: %v", err)
	}
}

func TestBillingAnomalyConfigSet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"PUT /v1/anomaly-config": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "org_id", "org-xyz")
			var got struct {
				WarningThreshold  float64 `json:"warning_threshold"`
				CriticalThreshold float64 `json:"critical_threshold"`
			}
			readJSONBody(t, r, &got)
			if got.WarningThreshold != 1.5 {
				t.Errorf("warning_threshold: got %f, want 1.5", got.WarningThreshold)
			}
			if got.CriticalThreshold != 3.0 {
				t.Errorf("critical_threshold: got %f, want 3.0", got.CriticalThreshold)
			}
			respondJSON(t, w, http.StatusOK, map[string]any{
				"warning_threshold":  got.WarningThreshold,
				"critical_threshold": got.CriticalThreshold,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newBillingAnomalyConfigSetCommand(state)
	cmd.SetArgs([]string{"--org", "org-xyz", "--warning", "1.5", "--critical", "3.0"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBillingAnomalyConfigGet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/anomaly-config": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, map[string]any{
				"warning_threshold":  2.0,
				"critical_threshold": 4.0,
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newBillingAnomalyConfigGetCommand(state)
	cmd.SetArgs([]string{})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBillingRegions_Success(t *testing.T) {
	t.Parallel()

	regions := []map[string]string{
		{"id": "us-east-1", "name": "US East (N. Virginia)"},
		{"id": "eu-west-1", "name": "Europe (Ireland)"},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/regions": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, regions)
		},
	})

	state := newTestState(t, srv)
	cmd := newBillingRegionsCommand(state)
	cmd.SetArgs([]string{})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "us-east-1") {
		t.Fatalf("expected region id in output: %s", out)
	}
}

func TestBillingDowngradePreview_Success(t *testing.T) {
	t.Parallel()

	preview := map[string]any{
		"org_id":      "org-abc",
		"target_tier": "starter",
		"impacts":     []string{"reduced seats", "no SSO"},
	}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/downgrade-preview": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "org_id", "org-abc")
			assertQuery(t, r, "target_tier", "starter")
			respondJSON(t, w, http.StatusOK, preview)
		},
	})

	state := newTestState(t, srv)
	cmd := newBillingDowngradePreviewCommand(state)
	cmd.SetArgs([]string{"--org", "org-abc", "--target-tier", "starter"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBillingDowngradePreview_RequiresOrg(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newBillingDowngradePreviewCommand(state)
	cmd.SetArgs([]string{"--target-tier", "starter"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--org is required") {
		t.Fatalf("expected --org error, got: %v", err)
	}
}

func TestBillingDowngradePreview_RequiresTargetTier(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newBillingDowngradePreviewCommand(state)
	cmd.SetArgs([]string{"--org", "org-abc"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--target-tier is required") {
		t.Fatalf("expected --target-tier error, got: %v", err)
	}
}

func TestBillingCheckOrgLimit_Success(t *testing.T) {
	t.Parallel()

	result := map[string]any{"user_id": "user-123", "plan_tier": "pro", "at_limit": false}

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/billing/check-org-limit": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "user_id", "user-123")
			assertQuery(t, r, "plan_tier", "pro")
			respondJSON(t, w, http.StatusOK, result)
		},
	})

	state := newTestState(t, srv)
	cmd := newBillingCheckOrgLimitCommand(state)
	cmd.SetArgs([]string{"--user-id", "user-123", "--plan-tier", "pro"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBillingCheckOrgLimit_RequiresUserID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newBillingCheckOrgLimitCommand(state)
	cmd.SetArgs([]string{"--plan-tier", "pro"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--user-id is required") {
		t.Fatalf("expected --user-id error, got: %v", err)
	}
}

func TestBillingCheckOrgLimit_RequiresPlanTier(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newBillingCheckOrgLimitCommand(state)
	cmd.SetArgs([]string{"--user-id", "user-123"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--plan-tier is required") {
		t.Fatalf("expected --plan-tier error, got: %v", err)
	}
}
