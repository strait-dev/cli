package main

import (
	"net/http"
	"strings"
	"testing"
)

// Roles list.

func TestRolesList_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/roles": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []map[string]any{
				{"id": "role-1", "name": "admin"},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newRolesListCommand(state)

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "role-1") {
		t.Fatalf("expected role id in output: %s", out)
	}
}

// Roles create — asserts name and permissions in request body.

func TestRolesCreate_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/roles": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			var got struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Permissions []string `json:"permissions"`
			}
			readJSONBody(t, r, &got)
			if got.Name != "ops-lead" {
				t.Errorf("name: got %q, want %q", got.Name, "ops-lead")
			}
			if len(got.Permissions) != 2 {
				t.Errorf("permissions count: got %d, want 2", len(got.Permissions))
			}
			wantPerms := map[string]bool{"jobs:read": true, "runs:write": true}
			for _, p := range got.Permissions {
				if !wantPerms[p] {
					t.Errorf("unexpected permission %q", p)
				}
			}
			respondJSON(t, w, http.StatusCreated, map[string]any{"id": "role-new", "name": got.Name})
		},
	})

	state := newTestState(t, srv)
	cmd := newRolesCreateCommand(state)
	cmd.SetArgs([]string{
		"--name", "ops-lead",
		"--description", "Operations lead role",
		"--permission", "jobs:read",
		"--permission", "runs:write",
	})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRolesCreate_RequiresName(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newRolesCreateCommand(state)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("expected --name error, got: %v", err)
	}
}

func TestRolesDelete_WithYes(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/roles/role-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newRolesDeleteCommand(state)
	cmd.SetArgs([]string{"role-1", "--yes"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Tag-policies create — asserts tag_key and actions in request body.

func TestTagPoliciesCreate_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/tag-policies": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			var got struct {
				ProjectID    string   `json:"project_id"`
				ResourceType string   `json:"resource_type"`
				UserID       string   `json:"user_id"`
				TagKey       string   `json:"tag_key"`
				TagValue     string   `json:"tag_value"`
				Actions      []string `json:"actions"`
			}
			readJSONBody(t, r, &got)
			if got.TagKey != "env" {
				t.Errorf("tag_key: got %q, want %q", got.TagKey, "env")
			}
			if len(got.Actions) != 2 {
				t.Errorf("actions count: got %d, want 2", len(got.Actions))
			}
			wantActions := map[string]bool{"read": true, "write": true}
			for _, a := range got.Actions {
				if !wantActions[a] {
					t.Errorf("unexpected action %q", a)
				}
			}
			if got.ProjectID != "proj-test" {
				t.Errorf("project_id: got %q, want %q", got.ProjectID, "proj-test")
			}
			respondJSON(t, w, http.StatusCreated, map[string]any{"id": "policy-new"})
		},
	})

	state := newTestState(t, srv)
	cmd := newTagPoliciesCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--resource-type", "job",
		"--user-id", "user-abc",
		"--tag-key", "env",
		"--tag-value", "production",
		"--action", "read",
		"--action", "write",
	})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTagPoliciesCreate_RequiresTagKey(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach the server")
	}))

	state := newTestState(t, srv)
	cmd := newTagPoliciesCreateCommand(state)
	cmd.SetArgs([]string{
		"--project", "proj-test",
		"--resource-type", "job",
		"--user-id", "user-abc",
		"--action", "read",
		// --tag-key intentionally omitted
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--tag-key is required") {
		t.Fatalf("expected --tag-key error, got: %v", err)
	}
}

func TestTagPoliciesDelete_WithYes(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/tag-policies/policy-1": func(w http.ResponseWriter, _ *http.Request) {
			respondJSON(t, w, http.StatusOK, map[string]string{})
		},
	})

	state := newTestState(t, srv)
	cmd := newTagPoliciesDeleteCommand(state)
	cmd.SetArgs([]string{"policy-1", "--yes"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Usage anomalies.

func TestUsageAnomalies_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/usage/anomalies": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondJSON(t, w, http.StatusOK, map[string]any{
				"anomalies": []map[string]any{
					{"type": "spike", "detected_at": "2026-06-01T00:00:00Z"},
				},
			})
		},
	})

	state := newTestState(t, srv)
	cmd := newUsageAnomaliesCommand(state)

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "spike") {
		t.Fatalf("expected anomaly type in output: %s", out)
	}
}

// Usage email-preferences-set — asserts monthly_usage_email in request body.

func TestUsageEmailPreferencesSet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"PUT /v1/usage/email-preferences": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "org_id", "org-xyz")
			var got struct {
				MonthlyUsageEmail bool `json:"monthly_usage_email"`
			}
			readJSONBody(t, r, &got)
			if !got.MonthlyUsageEmail {
				t.Errorf("monthly_usage_email: got false, want true")
			}
			respondJSON(t, w, http.StatusOK, map[string]any{"monthly_usage_email": true})
		},
	})

	state := newTestState(t, srv)
	cmd := newUsageEmailPreferencesSetCommand(state)
	cmd.SetArgs([]string{"--org", "org-xyz", "--monthly"})

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsageEmailPreferencesSet_MonthlyFalse(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"PUT /v1/usage/email-preferences": func(w http.ResponseWriter, r *http.Request) {
			var got struct {
				MonthlyUsageEmail bool `json:"monthly_usage_email"`
			}
			readJSONBody(t, r, &got)
			if got.MonthlyUsageEmail {
				t.Errorf("monthly_usage_email: got true, want false")
			}
			respondJSON(t, w, http.StatusOK, map[string]any{"monthly_usage_email": false})
		},
	})

	state := newTestState(t, srv)
	cmd := newUsageEmailPreferencesSetCommand(state)
	// --monthly not passed, defaults to false

	if err := captureAndExec(t, state, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUsageEmailPreferencesGet_Success(t *testing.T) {
	t.Parallel()

	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/usage/email-preferences": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "org_id", "org-xyz")
			respondJSON(t, w, http.StatusOK, map[string]any{"monthly_usage_email": true})
		},
	})

	state := newTestState(t, srv)
	cmd := newUsageEmailPreferencesGetCommand(state)
	cmd.SetArgs([]string{"--org", "org-xyz"})

	out := captureStateOutput(t, state, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "monthly_usage_email") {
		t.Fatalf("expected preference field in output: %s", out)
	}
}
