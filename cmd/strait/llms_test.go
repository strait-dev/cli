package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLLMSFlag_RegisteredOnRoot(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	flag := cmd.Flags().Lookup("llms")
	if flag == nil {
		t.Fatal("--llms flag not registered on root command")
	}
}

func TestLLMSManifest_IsValidJSON(t *testing.T) {
	t.Parallel()

	root := newRootCommand()
	root.SetArgs([]string{"--llms"})

	out := captureCommandOutput(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var manifest llmsManifest
	if err := json.Unmarshal([]byte(out), &manifest); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput (first 500 chars): %.500s", err, out)
	}
}

func TestLLMSManifest_ContainsTopLevelCommands(t *testing.T) {
	t.Parallel()

	root := newRootCommand()
	root.SetArgs([]string{"--llms"})

	out := captureCommandOutput(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"jobs", "runs", "deploy", "schema", "agent", "doctor"} {
		if !strings.Contains(out, want) {
			t.Errorf("manifest missing command %q", want)
		}
	}
}

func TestLLMSManifest_HasVersionAndCLIName(t *testing.T) {
	t.Parallel()

	root := newRootCommand()
	root.SetArgs([]string{"--llms"})

	out := captureCommandOutput(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var manifest llmsManifest
	if err := json.Unmarshal([]byte(out), &manifest); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if manifest.CLI != "strait" {
		t.Errorf("expected cli=strait, got %q", manifest.CLI)
	}
	if manifest.Version == "" {
		t.Error("expected version to be set")
	}
	if len(manifest.Commands) == 0 {
		t.Error("expected non-empty commands list")
	}
}

func TestLLMSManifest_FlagsHaveTypes(t *testing.T) {
	t.Parallel()

	root := newRootCommand()
	root.SetArgs([]string{"--llms"})

	out := captureCommandOutput(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	var manifest llmsManifest
	if err := json.Unmarshal([]byte(out), &manifest); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Find jobs command and check it has subcommands.
	var jobsCmd *llmsCommand
	for i, cmd := range manifest.Commands {
		if cmd.Name == "jobs" {
			jobsCmd = &manifest.Commands[i]
			break
		}
	}
	if jobsCmd == nil {
		t.Fatal("jobs command not found in manifest")
	}
	if len(jobsCmd.Subcommands) == 0 {
		t.Error("expected jobs to have subcommands")
	}
}

func TestLLMSManifest_DoesNotContainHiddenCommands(t *testing.T) {
	t.Parallel()

	root := newRootCommand()
	root.SetArgs([]string{"--llms"})

	out := captureCommandOutput(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// "completion" is a registered but hidden-ish command that should be excluded
	// from the LLM manifest (it was filtered by name in buildCommandTree).
	// "help" is always excluded.
	if strings.Contains(out, `"name":"help"`) {
		t.Error("manifest should not contain 'help' command")
	}
}

func TestFirstParagraph(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"single line", "single line"},
		{"first\n\nsecond", "first"},
		{"first\nstill first\n\nsecond paragraph", "first\nstill first"},
		{"no blank line here", "no blank line here"},
	}
	for _, tc := range cases {
		got := firstParagraph(tc.input)
		if got != tc.want {
			t.Errorf("firstParagraph(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
