package main

import (
	"strings"
	"testing"
)

func TestRequireProjectID_FromFlag(t *testing.T) {
	t.Parallel()
	state := &appState{opts: &rootOptions{}}
	got, err := requireProjectID(state, "proj-flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "proj-flag" {
		t.Fatalf("expected proj-flag, got %s", got)
	}
}

func TestRequireProjectID_FromState(t *testing.T) {
	t.Parallel()
	state := &appState{opts: &rootOptions{projectID: "proj-state"}}
	got, err := requireProjectID(state, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "proj-state" {
		t.Fatalf("expected proj-state, got %s", got)
	}
}

func TestRequireProjectID_Missing(t *testing.T) {
	t.Parallel()
	state := &appState{opts: &rootOptions{}}
	_, err := requireProjectID(state, "")
	if err == nil || !strings.Contains(err.Error(), "project ID is required") {
		t.Fatalf("expected project ID error, got: %v", err)
	}
}

func TestRequireConfirmation_YesFlag(t *testing.T) {
	t.Parallel()
	state := &appState{opts: &rootOptions{}}
	err := requireConfirmation(state, "Delete?", true)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestRequireConfirmation_CIMode(t *testing.T) {
	t.Parallel()
	state := &appState{opts: &rootOptions{ciMode: true}}
	err := requireConfirmation(state, "Delete?", false)
	if err == nil || !strings.Contains(err.Error(), "CI mode") {
		t.Fatalf("expected CI mode error, got: %v", err)
	}
}

func TestIsTTYRich_JSONFormat(t *testing.T) {
	t.Parallel()
	state := &appState{opts: &rootOptions{outputFormat: "json"}}
	if isTTYRich(state) {
		t.Fatal("expected false for json format")
	}
}

func TestIsTTYRich_TableFormat(t *testing.T) {
	t.Parallel()
	// In test environment, stdout is not a TTY, so isTTYRich returns false
	// regardless of format. This tests the format check path.
	state := &appState{opts: &rootOptions{outputFormat: "table"}}
	// In non-TTY (test environment), isTTYRich returns false.
	if isTTYRich(state) {
		t.Fatal("expected false in non-TTY environment")
	}
}
