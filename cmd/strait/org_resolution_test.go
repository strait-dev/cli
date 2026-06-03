package main

import (
	"testing"

	cliconfig "github.com/strait-dev/cli/internal/config"
)

func TestRequireOrgID(t *testing.T) {
	t.Parallel()

	t.Run("flag wins", func(t *testing.T) {
		t.Parallel()
		st := &appState{opts: &rootOptions{}, resolved: cliconfig.Resolved{OrgID: "resolved-org"}}
		got, err := requireOrgID(st, "flag-org")
		if err != nil || got != "flag-org" {
			t.Fatalf("got %q, err %v; want flag-org", got, err)
		}
	})

	t.Run("falls back to resolved org", func(t *testing.T) {
		t.Parallel()
		st := &appState{opts: &rootOptions{}, resolved: cliconfig.Resolved{OrgID: "env-org"}}
		got, err := requireOrgID(st, "")
		if err != nil || got != "env-org" {
			t.Fatalf("got %q, err %v; want env-org", got, err)
		}
	})

	t.Run("errors when none set", func(t *testing.T) {
		t.Parallel()
		st := &appState{opts: &rootOptions{}}
		if _, err := requireOrgID(st, ""); err == nil {
			t.Fatal("expected error when no org is set")
		}
	})
}

// TestResolveOrgFromEnvAndConfig verifies STRAIT_ORG, config default, and
// context org feed into Resolved.OrgID with the expected precedence.
func TestResolveOrgFromEnvAndConfig(t *testing.T) {
	t.Parallel()

	t.Run("env", func(t *testing.T) {
		t.Parallel()
		got := cliconfig.Resolve(cliconfig.ResolveInput{
			Env: map[string]string{"STRAIT_ORG": "env-org"},
		})
		if got.OrgID != "env-org" {
			t.Fatalf("OrgID = %q, want env-org", got.OrgID)
		}
	})

	t.Run("config default overrides env", func(t *testing.T) {
		t.Parallel()
		got := cliconfig.Resolve(cliconfig.ResolveInput{
			Env:    map[string]string{"STRAIT_ORG": "env-org"},
			Config: &cliconfig.File{DefaultOrg: "config-org"},
		})
		if got.OrgID != "config-org" {
			t.Fatalf("OrgID = %q, want config-org", got.OrgID)
		}
	})

	t.Run("context overrides config", func(t *testing.T) {
		t.Parallel()
		got := cliconfig.Resolve(cliconfig.ResolveInput{
			Config: &cliconfig.File{
				DefaultOrg:    "config-org",
				ActiveContext: "prod",
				Contexts:      map[string]cliconfig.Context{"prod": {Org: "ctx-org"}},
			},
		})
		if got.OrgID != "ctx-org" {
			t.Fatalf("OrgID = %q, want ctx-org", got.OrgID)
		}
	})
}
