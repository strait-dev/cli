package main

import "testing"

func TestOpen_UsesDashboardRootWhenNoResourceProvided(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{serverURL: "https://api.strait.dev:8080"}}
	var opened string
	prev := openBrowserFunc
	openBrowserFunc = func(target string) error {
		opened = target
		return nil
	}
	t.Cleanup(func() { openBrowserFunc = prev })

	cmd := newOpenCommand(state)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opened != "https://app.strait.dev:5173" {
		t.Fatalf("opened URL = %q, want %q", opened, "https://app.strait.dev:5173")
	}
}

func TestOpen_RunIDTargetsRunsPage(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{serverURL: "http://localhost:8080"}}
	var opened string
	prev := openBrowserFunc
	openBrowserFunc = func(target string) error {
		opened = target
		return nil
	}
	t.Cleanup(func() { openBrowserFunc = prev })

	cmd := newOpenCommand(state)
	cmd.SetArgs([]string{"run-123"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opened != "http://localhost:5173/runs/run-123" {
		t.Fatalf("opened URL = %q, want %q", opened, "http://localhost:5173/runs/run-123")
	}
}

func TestOpen_JobSlugTargetsJobsPageAndEscapesPath(t *testing.T) {
	t.Parallel()

	state := &appState{opts: &rootOptions{serverURL: "https://api.example.com"}}
	var opened string
	prev := openBrowserFunc
	openBrowserFunc = func(target string) error {
		opened = target
		return nil
	}
	t.Cleanup(func() { openBrowserFunc = prev })

	cmd := newOpenCommand(state)
	cmd.SetArgs([]string{"billing/import nightly"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opened != "https://app.example.com/jobs/billing%2Fimport%20nightly" {
		t.Fatalf("opened URL = %q, want %q", opened, "https://app.example.com/jobs/billing%2Fimport%20nightly")
	}
}
