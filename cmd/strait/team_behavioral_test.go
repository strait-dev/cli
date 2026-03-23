package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/strait-dev/cli/internal/types"
)

var testMember = types.ProjectMemberRole{
	ID:        "mem-1",
	ProjectID: "proj-test",
	UserID:    "user-1",
	RoleID:    "role-operator",
	GrantedBy: "user-admin",
	CreatedAt: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
}

var testRole = types.ProjectRole{
	ID:          "role-1",
	ProjectID:   "proj-test",
	Name:        "operator",
	Permissions: []string{"jobs:read", "jobs:write"},
	IsSystem:    true,
	CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	UpdatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
}

func TestTeamList_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/members": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			assertQuery(t, r, "project_id", "proj-test")
			respondPaginated(t, w, http.StatusOK, []types.ProjectMemberRole{testMember})
		},
	})
	state := newTestState(t, srv)
	cmd := newTeamListCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "user-1") {
		t.Fatalf("expected user-1 in output, got: %s", out)
	}
}

func TestTeamAdd_Success(t *testing.T) {
	t.Parallel()
	var receivedBody map[string]any
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"POST /v1/members": func(w http.ResponseWriter, r *http.Request) {
			readJSONBody(t, r, &receivedBody)
			respondJSON(t, w, http.StatusCreated, testMember)
		},
	})
	state := newTestState(t, srv)
	cmd := newTeamAddCommand(state)
	cmd.SetArgs([]string{"user-1", "--role-id", "role-operator"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if receivedBody["user_id"] != "user-1" {
		t.Fatalf("expected user_id=user-1, got: %v", receivedBody)
	}
	if !strings.Contains(out, "user-1") || !strings.Contains(out, "mem-1") {
		t.Fatalf("expected member in output, got: %s", out)
	}
}

func TestTeamAdd_MissingRoleID(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newTeamAddCommand(state)
	cmd.SetArgs([]string{"user-1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--role-id is required") {
		t.Fatalf("expected role-id error, got: %v", err)
	}
}

func TestTeamRemove_WithYes(t *testing.T) {
	t.Parallel()
	removeCalled := false
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"DELETE /v1/members/user-1": func(w http.ResponseWriter, _ *http.Request) {
			removeCalled = true
			w.WriteHeader(http.StatusNoContent)
		},
	})
	state := newTestState(t, srv)
	cmd := newTeamRemoveCommand(state)
	cmd.SetArgs([]string{"user-1", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !removeCalled {
		t.Fatal("expected DELETE to be called")
	}
}

func TestTeamRoles_Success(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t, map[string]http.HandlerFunc{
		"GET /v1/roles": func(w http.ResponseWriter, r *http.Request) {
			assertAuth(t, r, "test-key")
			respondPaginated(t, w, http.StatusOK, []types.ProjectRole{testRole})
		},
	})
	state := newTestState(t, srv)
	cmd := newTeamRolesCommand(state)
	cmd.SetArgs([]string{"--project", "proj-test"})
	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "operator") {
		t.Fatalf("expected operator in output, got: %s", out)
	}
}
