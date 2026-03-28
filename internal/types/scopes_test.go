package types

import (
	"testing"
)

func TestValidateScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		scopes  []string
		wantErr bool
	}{
		{"nil scopes", nil, false},
		{"empty scopes", []string{}, false},
		{"single valid scope", []string{"jobs:read"}, false},
		{"wildcard scope", []string{"*"}, false},
		{"multiple valid scopes", []string{"jobs:read", "jobs:write", "runs:read"}, false},
		{"all valid scopes", []string{
			ScopeAll, ScopeJobsRead, ScopeJobsWrite, ScopeJobsTrigger,
			ScopeRunsRead, ScopeRunsWrite, ScopeWorkflowsRead, ScopeWorkflowsWrite,
			ScopeWorkflowsTrigger, ScopeSecretsRead, ScopeSecretsWrite,
			ScopeAPIKeysManage, ScopeRBACManage, ScopeStatsRead,
			ScopeProjectsRead, ScopeProjectsWrite, ScopeProjectsManage,
		}, false},
		{"unknown scope", []string{"unknown:scope"}, true},
		{"valid then invalid", []string{"jobs:read", "bad:scope"}, true},
		{"empty string scope", []string{""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateScopes(tt.scopes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScopes(%v) error = %v, wantErr %v", tt.scopes, err, tt.wantErr)
			}
		})
	}
}

func TestHasScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		scopes   []string
		required string
		want     bool
	}{
		{"empty scopes grants all", []string{}, "jobs:read", true},
		{"nil scopes grants all", nil, "jobs:read", true},
		{"exact match", []string{"jobs:read", "jobs:write"}, "jobs:read", true},
		{"no match", []string{"jobs:read", "jobs:write"}, "runs:read", false},
		{"wildcard grants any", []string{"*"}, "runs:write", true},
		{"wildcard among others", []string{"jobs:read", "*"}, "secrets:write", true},
		{"single scope match", []string{"stats:read"}, "stats:read", true},
		{"single scope no match", []string{"stats:read"}, "jobs:read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := HasScope(tt.scopes, tt.required); got != tt.want {
				t.Errorf("HasScope(%v, %q) = %v, want %v", tt.scopes, tt.required, got, tt.want)
			}
		})
	}
}

func TestValidScopes_Completeness(t *testing.T) {
	t.Parallel()

	// Every scope constant should be in ValidScopes.
	constants := []string{
		ScopeAll, ScopeJobsRead, ScopeJobsWrite, ScopeJobsTrigger,
		ScopeRunsRead, ScopeRunsWrite, ScopeWorkflowsRead, ScopeWorkflowsWrite,
		ScopeWorkflowsTrigger, ScopeSecretsRead, ScopeSecretsWrite,
		ScopeAPIKeysManage, ScopeRBACManage, ScopeStatsRead,
		ScopeProjectsRead, ScopeProjectsWrite, ScopeProjectsManage,
	}

	for _, scope := range constants {
		if !ValidScopes[scope] {
			t.Errorf("scope constant %q is not in ValidScopes map", scope)
		}
	}

	// ValidScopes should have exactly the same count as constants.
	if len(ValidScopes) != len(constants) {
		t.Errorf("ValidScopes has %d entries, but %d scope constants exist", len(ValidScopes), len(constants))
	}
}
