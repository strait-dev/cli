package main

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRootCommand_HasExpectedSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	expected := []string{
		"version", "completion", "context", "alias", "auth",
		"jobs", "runs", "workflows", "workflow-runs", "api-keys",
		"wait", "logs", "triggers", "secrets", "extension",
		"upgrade", "projects", "debug", "team", "config",
		"env", "webhooks", "event-sources", "log-drains",
		"usage", "analytics", "init", "migrate", "tui",
	}

	subs := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}

	for _, name := range expected {
		if !subs[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestRootCommand_PersistentFlags(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	flags := []string{
		"server", "api-key", "project", "format", "no-headers",
		"output-template", "output-jsonpath", "no-color", "quiet",
		"verbose", "context", "config", "timeout", "ci",
	}

	for _, name := range flags {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("missing persistent flag: --%s", name)
		}
	}
}

func TestRootCommand_FlagDefaults(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()

	tests := []struct {
		flag string
		want string
	}{
		{flag: "timeout", want: (30 * time.Second).String()},
		{flag: "ci", want: "false"},
		{flag: "quiet", want: "false"},
		{flag: "no-color", want: "false"},
		{flag: "verbose", want: "false"},
		{flag: "no-headers", want: "false"},
	}

	for _, tc := range tests {
		f := cmd.PersistentFlags().Lookup(tc.flag)
		if f == nil {
			t.Errorf("flag --%s not found", tc.flag)
			continue
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default: got %q, want %q", tc.flag, f.DefValue, tc.want)
		}
	}
}

func TestJobsCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	jobs := findSubcommand(t, cmd, "jobs")

	expected := []string{"list", "get", "create", "update", "delete", "clone", "trigger", "health", "versions", "dependencies", "batch"}
	assertSubcommands(t, jobs, expected)
}

func TestJobsListCommand_Flags(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	jobs := findSubcommand(t, cmd, "jobs")
	list := findSubcommand(t, jobs, "list")

	if list.Flags().Lookup("project") == nil {
		t.Error("jobs list missing --project flag")
	}
}

func TestRunsCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	runs := findSubcommand(t, cmd, "runs")

	expected := []string{"list", "get", "cancel", "logs", "watch", "replay"}
	assertSubcommands(t, runs, expected)
}

func TestRunsListCommand_Flags(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	runs := findSubcommand(t, cmd, "runs")
	list := findSubcommand(t, runs, "list")

	for _, name := range []string{"project", "status", "limit"} {
		if list.Flags().Lookup(name) == nil {
			t.Errorf("runs list missing --%s flag", name)
		}
	}

	limitFlag := list.Flags().Lookup("limit")
	if limitFlag.DefValue != "50" {
		t.Errorf("runs list --limit default: got %q, want 50", limitFlag.DefValue)
	}
}

func TestRunsCancelCommand_Flags(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	runs := findSubcommand(t, cmd, "runs")
	cancel := findSubcommand(t, runs, "cancel")

	for _, name := range []string{"all", "yes"} {
		if cancel.Flags().Lookup(name) == nil {
			t.Errorf("runs cancel missing --%s flag", name)
		}
	}
}

func TestWorkflowsCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	wf := findSubcommand(t, cmd, "workflows")

	expected := []string{"list", "get", "create", "update", "delete", "clone", "trigger", "dry-run", "plan", "simulate", "versions", "diff", "policy"}
	assertSubcommands(t, wf, expected)
}

func TestVersionCommand_Flags(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	version := findSubcommand(t, cmd, "version")

	for _, name := range []string{"short", "json", "check-server", "check-update"} {
		if version.Flags().Lookup(name) == nil {
			t.Errorf("version missing --%s flag", name)
		}
	}
}

func TestCIMode_Flag(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	ci := cmd.PersistentFlags().Lookup("ci")
	if ci == nil {
		t.Fatal("missing --ci flag")
	}
	if ci.DefValue != "false" {
		t.Errorf("--ci default: got %q, want false", ci.DefValue)
	}
}

func TestSecretsCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	secrets := findSubcommand(t, cmd, "secrets")

	expected := []string{"list", "create", "delete"}
	assertSubcommands(t, secrets, expected)
}

func TestTeamCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	team := findSubcommand(t, cmd, "team")

	expected := []string{"list", "add", "remove", "roles", "audit"}
	assertSubcommands(t, team, expected)
}

func TestTriggersCommand_HasStreamAndSendRaw(t *testing.T) {
	t.Parallel()
	cmd := newRootCommand()
	triggers := findSubcommand(t, cmd, "triggers")
	assertSubcommands(t, triggers, []string{"list", "get", "send", "stream", "purge"})

	send := findSubcommand(t, triggers, "send")
	if send.Flags().Lookup("raw") == nil {
		t.Error("triggers send missing --raw flag")
	}
	if send.Flags().Lookup("project") == nil {
		t.Error("triggers send missing --project flag")
	}
}

func TestAnalyticsCommand_HasPerformance(t *testing.T) {
	t.Parallel()
	cmd := newRootCommand()
	analytics := findSubcommand(t, cmd, "analytics")
	assertSubcommands(t, analytics, []string{"costs", "reliability", "top-failing", "performance"})
}

func TestMigrateCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	migrate := findSubcommand(t, cmd, "migrate")
	assertSubcommands(t, migrate, []string{"inngest", "trigger", "hatchet"})
}

func TestInitCommand_HasFlags(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	initCmd := findSubcommand(t, cmd, "init")
	for _, name := range []string{"template", "name", "force", "list"} {
		if initCmd.Flags().Lookup(name) == nil {
			t.Errorf("init missing --%s flag", name)
		}
	}
}

func TestDebugCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	debug := findSubcommand(t, cmd, "debug")

	expected := []string{"bundle", "request"}
	assertSubcommands(t, debug, expected)
}

func TestTeamAddCommand_UsesUserAndRoleIDs(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	team := findSubcommand(t, cmd, "team")
	add := findSubcommand(t, team, "add")

	if add.Use != "add <user-id>" {
		t.Fatalf("unexpected usage: %s", add.Use)
	}
	if add.Flags().Lookup("role-id") == nil {
		t.Fatal("team add missing --role-id flag")
	}
	if add.Flags().Lookup("role") != nil {
		t.Fatal("team add should not expose deprecated --role flag")
	}
}

// Test helpers.

func findSubcommand(t *testing.T, parent interface{ Commands() []*cobra.Command }, name string) *cobra.Command {
	t.Helper()
	for _, sub := range parent.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

func TestWorkflowsTriggerCommand_HasProjectFlag(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	wf := findSubcommand(t, cmd, "workflows")
	trigger := findSubcommand(t, wf, "trigger")

	for _, name := range []string{"project", "payload", "payload-file"} {
		if trigger.Flags().Lookup(name) == nil {
			t.Errorf("workflows trigger missing --%s flag", name)
		}
	}
}

func assertSubcommands(t *testing.T, parent interface{ Commands() []*cobra.Command }, expected []string) {
	t.Helper()
	subs := make(map[string]bool)
	for _, sub := range parent.Commands() {
		subs[sub.Name()] = true
	}
	for _, name := range expected {
		if !subs[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}
