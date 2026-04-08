package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAgentCommand_RegisteredOnRoot(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	if !names["agent"] {
		t.Fatal("'agent' command not registered on root")
	}
}

func TestAgentCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newAgentCommand(state)

	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}

	for _, want := range []string{"capabilities", "context", "describe", "skills"} {
		if !names[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestAgentCapabilities_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newAgentCommand(state)
	cmd.SetArgs([]string{"capabilities"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var caps []agentCapability
	if err := json.Unmarshal([]byte(out), &caps); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if len(caps) == 0 {
		t.Fatal("expected non-empty capabilities list")
	}

	// Verify all capabilities have a name and at least one command.
	names := make(map[string]bool)
	for _, c := range caps {
		if c.Name == "" {
			t.Error("capability with empty name")
		}
		if len(c.Commands) == 0 {
			t.Errorf("capability %q has no commands", c.Name)
		}
		names[c.Name] = true
	}

	// Check key capabilities are present.
	for _, want := range []string{"job_management", "code_deployment", "structured_output", "non_interactive_mode"} {
		if !names[want] {
			t.Errorf("missing capability %q", want)
		}
	}
}

func TestAgentContext_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newAgentCommand(state)
	cmd.SetArgs([]string{"context"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var ctx agentContext
	if err := json.Unmarshal([]byte(out), &ctx); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if ctx.ServerURL == "" {
		t.Error("expected server_url to be set")
	}
	if ctx.ProjectID == "" {
		t.Error("expected project_id to be set")
	}
	if !ctx.APIKeySet {
		t.Error("expected api_key_set=true")
	}
	if ctx.CLIVersion == "" {
		t.Error("expected cli_version to be set")
	}
}

func TestAgentDescribe_JobsCommand(t *testing.T) {
	t.Parallel()

	// describe walks cmd.Root(), so it must be wired into the full command tree.
	root := newRootCommand()
	root.SetArgs([]string{"agent", "describe", "jobs"})

	out := captureCommandOutput(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, `"command"`) {
		t.Fatalf("expected JSON output, got: %s", out)
	}
	if !strings.Contains(out, "jobs") {
		t.Fatalf("expected 'jobs' in output, got: %s", out)
	}
}

func TestAgentDescribe_UnknownCommand(t *testing.T) {
	t.Parallel()

	root := newRootCommand()
	root.SetArgs([]string{"agent", "describe", "nonexistent-command-xyz"})

	captureCommandOutput(t, func() {
		err := root.Execute()
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not-found error, got: %v", err)
		}
	})
}

func TestAgentSkills_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newAgentCommand(state)
	cmd.SetArgs([]string{"skills"})

	out := captureCommandOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var skills []agentSkill
	if err := json.Unmarshal([]byte(out), &skills); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	// Skills list may be empty if skills/ dir is not in CWD — that's OK,
	// but it must be a valid JSON array.
}
