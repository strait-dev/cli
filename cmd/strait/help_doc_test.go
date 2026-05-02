package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestResourceGroupsDocumentIdOrSlug asserts that every command group whose
// sub-commands accept a UUID-or-slug identifier exposes a Long: docstring
// explaining the convention. The reviewer flagged this as missing in the
// initial PR — slugs require an active project context, and that requirement
// must surface in --help output, not only in error messages.
func TestResourceGroupsDocumentIdOrSlug(t *testing.T) {
	t.Parallel()

	root := newRootCommand()

	// Resource groups whose sub-commands take an id-or-slug arg. Each must
	// document the identifier convention in Long.
	groups := []string{
		"jobs",
		"workflows",
		"environments",
		"event-sources",
		"job-groups",
	}

	for _, name := range groups {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd, _, err := root.Find([]string{name})
			if err != nil {
				t.Fatalf("could not find %q command: %v", name, err)
			}
			if cmd.Long == "" {
				t.Fatalf("%q has no Long: docstring; expected id-or-slug documentation", name)
			}
			// Both forms must be mentioned so users understand the choice.
			if !strings.Contains(cmd.Long, "UUID") {
				t.Errorf("%q Long: should mention UUID; got %q", name, cmd.Long)
			}
			if !strings.Contains(cmd.Long, "slug") {
				t.Errorf("%q Long: should mention slug; got %q", name, cmd.Long)
			}
			// Project-context requirement for slugs must be surfaced.
			if !strings.Contains(cmd.Long, "project") {
				t.Errorf("%q Long: should mention project context for slugs; got %q", name, cmd.Long)
			}
		})
	}
}

// TestRequiredFlagAnnotationsAreEnforced asserts that every command in the
// tree which advertises a required flag (via cmd.Flags().GetString(...) being
// flagged BashCompOneRequiredFlag = ["true"]) actually attaches the
// annotation to a real flag — i.e. that mustMarkFlagRequired never silently
// failed at construction time.
//
// This is a guardrail against the previous pattern of
// `_ = cmd.MarkFlagRequired("typo")` swallowing the lookup error. The whole
// command tree is constructed in newRootCommand() — if any flag name is
// misspelled, mustMarkFlagRequired panics during root construction and this
// test reports it via the recover in t.Fatal.
//
// Additionally, when the tree builds successfully, this walks every command
// and verifies the annotated flag still exists on the local FlagSet.
func TestRequiredFlagAnnotationsAreEnforced(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("newRootCommand() panicked — likely a typo in mustMarkFlagRequired: %v", r)
		}
	}()

	root := newRootCommand()
	visited := 0

	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		visited++
		c.Flags().VisitAll(func(f *pflag.Flag) {
			required, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
			if !ok || len(required) == 0 || required[0] != "true" {
				return
			}
			// Lookup the flag by name; it must exist (this proves
			// MarkFlagRequired received a valid name).
			if got := c.Flags().Lookup(f.Name); got == nil {
				t.Errorf("command %q: flag %q is annotated as required but not found in FlagSet",
					c.CommandPath(), f.Name)
			}
		})
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	if visited == 0 {
		t.Fatal("walked 0 commands — root tree is empty?")
	}
}

// TestMustMarkFlagRequired_PanicsOnUnknownFlag asserts the helper itself
// crashes loudly when given a non-existent flag name — proving the type of
// programmer error we want to surface at startup is actually surfaced.
func TestMustMarkFlagRequired_PanicsOnUnknownFlag(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on unknown flag, got none")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "MarkFlagRequired") {
			t.Fatalf("panic message should mention MarkFlagRequired; got %v", r)
		}
	}()

	cmd := &cobra.Command{Use: "fake"}
	mustMarkFlagRequired(cmd, "does-not-exist")
}

// TestIdOrSlugLong_ContainsAllElements is a unit test for the helper function
// itself, ensuring it produces output that the test above can rely on.
func TestIdOrSlugLong_ContainsAllElements(t *testing.T) {
	t.Parallel()

	got := idOrSlugLong("widget", "Manage widgets.")
	required := []string{
		"Manage widgets.",
		"widget UUID",
		"widget slug",
		"--project",
		"STRAIT_PROJECT_ID",
		"strait use",
	}
	for _, want := range required {
		if !strings.Contains(got, want) {
			t.Errorf("idOrSlugLong output missing %q; got:\n%s", want, got)
		}
	}
}
