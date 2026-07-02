package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

	out := captureStateOutput(t, state, func() {
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
	for _, want := range []string{"job_management", "run_management", "structured_output", "non_interactive_mode"} {
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

	out := captureStateOutput(t, state, func() {
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

	// describe walks cmd.Root(), so build a fresh root and inject a buffer
	// into its persistent appState via cmd.SetOut. We instead drive it through
	// the agent command using a state we control so output goes to state.out().
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	root := newRootCommand()
	root.AddCommand(newAgentDescribeCommand(state))
	root.SetArgs([]string{"describe", "jobs"})

	out := captureStateOutput(t, state, func() {
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

	// No output capture: the unknown-command path returns an error before
	// writing to stdout, so we just assert the error.
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got: %v", err)
	}
}

func TestAgentSkills_OutputIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	cmd := newAgentCommand(state)
	cmd.SetArgs([]string{"skills"})

	out := captureStateOutput(t, state, func() {
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

// runSkillsGenerate builds a fresh root command tree with the given state
// driving the skills-generate handler. Parenting it to root preserves the
// behavior of cmd.Root() walks performed by the handler.
func runSkillsGenerate(t *testing.T, state *appState, args ...string) string {
	t.Helper()
	root := newRootCommand()
	gen := newAgentSkillsGenerateCommand(state)
	root.AddCommand(gen)
	root.SetArgs(append([]string{"generate"}, args...))
	return captureStateOutput(t, state, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("skills generate %v: %v", args, err)
		}
	})
}

func TestAgentSkillsGenerate_CreatesFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	out := runSkillsGenerate(t, state, "--output-dir", dir)

	var results []skillFileResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one skill file to be generated")
	}

	for _, r := range results {
		if r.Status != "created" {
			t.Errorf("expected status=created for %q, got %q", r.Command, r.Status)
		}
		if r.File == "" {
			t.Errorf("result for %q has empty file path", r.Command)
		}
	}
}

func TestAgentSkillsGenerate_SkipsExistingWithoutOverwrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)

	runSkillsGenerate(t, state, "--output-dir", dir)
	out := runSkillsGenerate(t, state, "--output-dir", dir)

	var results []skillFileResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	for _, r := range results {
		if r.Status != "skipped" {
			t.Errorf("expected status=skipped on second run for %q, got %q", r.Command, r.Status)
		}
	}
}

func TestAgentSkillsGenerate_OverwriteReplacesFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)

	runSkillsGenerate(t, state, "--output-dir", dir)
	out := runSkillsGenerate(t, state, "--output-dir", dir, "--overwrite")

	var results []skillFileResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("expected results with --overwrite")
	}
	for _, r := range results {
		if r.Status != "created" {
			t.Errorf("expected status=created with --overwrite for %q, got %q", r.Command, r.Status)
		}
	}
}

func TestAgentSkillsGenerate_FileContainsSkillHeader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	runSkillsGenerate(t, state, "--output-dir", dir)

	content, err := os.ReadFile(dir + "/jobs.md")
	if err != nil {
		t.Fatalf("jobs.md not found: %v", err)
	}
	s := string(content)

	if !strings.HasPrefix(s, "# Skill: jobs") {
		t.Errorf("expected file to start with '# Skill: jobs', got: %.100s", s)
	}
	if !strings.Contains(s, "## Usage") {
		t.Error("expected ## Usage section")
	}
	if !strings.Contains(s, "## Agent notes") {
		t.Error("expected ## Agent notes section")
	}
}

func TestAgentSkillsGenerate_HiddenCommandsExcluded(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srv := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	state := newTestState(t, srv)
	runSkillsGenerate(t, state, "--output-dir", dir)

	for _, name := range []string{"help", "completion"} {
		if _, err := os.Stat(dir + "/" + name + ".md"); err == nil {
			t.Errorf("unexpected skill file generated for hidden command %q", name)
		}
	}
}

func TestBuildSkillMarkdown_ContainsSections(t *testing.T) {
	t.Parallel()

	root := newRootCommand()
	var jobsCmd *cobra.Command
	for _, sub := range root.Commands() {
		if sub.Name() == "jobs" {
			jobsCmd = sub
			break
		}
	}
	if jobsCmd == nil {
		t.Fatal("jobs command not found")
	}

	md := buildSkillMarkdown(jobsCmd)

	for _, want := range []string{
		"# Skill: jobs",
		"## Usage",
		"## Agent notes",
		"Exit codes",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in generated Markdown", want)
		}
	}
}
