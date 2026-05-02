package main

import (
	"strings"
	"testing"
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
