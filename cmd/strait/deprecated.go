package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/strait-dev/cli/internal/styles"
)

// legacyStub is a single removed top-level command and the migration message
// shown when a user invokes it. The stub command exits non-zero so CI scripts
// don't silently keep running with an incorrect assumption.
type legacyStub struct {
	name    string
	message string
}

// legacyMigrationStubs returns the list of removed top-level command names
// that get rendered as hidden cobra commands returning a styled migration
// error. Strait pivoted to orchestration-only — managed-compute commands and
// the long tail of non-canonical commands were removed in this minor.
func legacyMigrationStubs() []legacyStub {
	return []legacyStub{
		// Managed-compute removals.
		{"deploy", "managed-mode `deploy` was removed; use `strait deploy push` to upsert SDK-defined jobs"},
		{"build", "managed-mode `build` was removed; SDK definitions are deployed via `strait deploy push`"},
		{"verify", "managed-mode `verify` was removed; use `strait endpoint verify <slug>` to verify a serve endpoint"},
		{"deployments", "managed-mode `deployments` was removed; use `strait deploy push` to upsert jobs"},

		// Replaced surfaces.
		{"dev", "the `dev` command is replaced by orchestration mode (`strait dev` from this minor)"},

		// Diagnostics surfaces collapsed.
		{"doctor", "removed; use `strait debug bundle` for diagnostics"},
		{"diagnose", "removed; use `strait debug bundle` for diagnostics"},
		{"check", "removed; use `strait debug bundle` for diagnostics"},
		{"status", "removed; use `strait debug bundle` for diagnostics"},
		{"health", "removed; use `strait debug bundle` for diagnostics"},
		{"api", "removed; use the appropriate canonical subcommand (jobs, runs, workflows, …)"},

		// Dashboard-only surfaces.
		{"top", "removed; use the dashboard"},
		{"tui", "removed; use the dashboard"},
		{"agent", "removed; use the dashboard"},

		// Declarative was rolled into dry-run/--dry-run.
		{"validate", "removed; use `strait workflows dry-run` / `strait jobs update --dry-run`"},
		{"apply", "removed; use `strait workflows dry-run` / `strait jobs update --dry-run`"},
		{"diff", "removed; use `strait workflows dry-run` / `strait jobs update --dry-run`"},

		// Performance/analytics consolidation.
		{"stats", "use `strait analytics performance`"},
		{"perf", "use `strait analytics performance`"},
		{"profile", "use `strait analytics performance`"},

		// Eventing top-level → triggers / runs / worker.
		{"send", "use `strait triggers send <key>`"},
		{"listen", "use `strait triggers stream`"},
		{"drain", "use `strait worker drain`"},
		{"events", "use `strait runs events`"},
		{"trigger", "use `strait jobs trigger`"},

		// Auth top-level → auth subcommands.
		{"whoami", "use `strait auth whoami`"},
		{"login", "use `strait auth login`"},
		{"logout", "use `strait auth logout`"},

		// Audit top-level → team audit.
		{"audit", "use `strait team audit`"},

		// Long tail removals (not in canonical surface).
		{"schema", "removed; not in canonical surface"},
		{"export", "use `strait projects export`"},
		{"backup", "removed; not in canonical surface"},
		{"cleanup", "removed; not in canonical surface"},
		{"trace", "removed; not in canonical surface"},
		{"ci", "removed; not in canonical surface"},
		{"open", "removed; not in canonical surface"},
		{"run", "use `strait jobs trigger` or `strait workflows trigger`"},
		{"create", "use `strait jobs create` or `strait workflows create`"},
		{"docs", "removed; not in canonical surface"},
		{"update", "removed; not in canonical surface"},
		{"fixtures", "removed; not in canonical surface"},
		{"job-groups", "removed; not in canonical surface"},
		{"notifications", "removed; not in canonical surface"},
	}
}

// newDeprecatedStubCommand returns a hidden cobra command that prints a
// styled migration error to stderr and exits with a non-zero status when
// invoked. It accepts and ignores any arguments and flags so users get a
// consistent message regardless of what they pass.
func newDeprecatedStubCommand(name, message string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                name,
		Short:              fmt.Sprintf("Removed: %s", message),
		Hidden:             true,
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), styles.Warn(fmt.Sprintf("`strait %s` was removed: %s", name, message)))
			return fmt.Errorf("`strait %s` was removed", name)
		},
	}
	return cmd
}
