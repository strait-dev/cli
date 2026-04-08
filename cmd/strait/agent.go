package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newAgentCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent-oriented introspection and context commands",
		Long: `Commands designed for AI agents and automation scripts.
These subcommands expose CLI capabilities, environment context, and skill
references in a machine-readable format without requiring API connectivity.`,
		Example: `  strait agent capabilities
  strait agent context
  strait agent describe deploy source
  strait agent skills`,
	}

	cmd.AddCommand(newAgentCapabilitiesCommand(state))
	cmd.AddCommand(newAgentContextCommand(state))
	cmd.AddCommand(newAgentDescribeCommand(state))
	cmd.AddCommand(newAgentSkillsCommand(state))

	return cmd
}

// agentCapability describes a single CLI capability.
type agentCapability struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
	Flags       []string `json:"flags,omitempty"`
	EnvVars     []string `json:"env_vars,omitempty"`
}

func newAgentCapabilitiesCommand(_ *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "capabilities",
		Short: "List all agent-relevant CLI capabilities as JSON",
		RunE: func(_ *cobra.Command, _ []string) error {
			caps := []agentCapability{
				{
					Name:        "job_management",
					Description: "Create, list, get, update, pause, resume, and delete jobs",
					Commands:    []string{"jobs list", "jobs get", "jobs create", "jobs update", "jobs pause", "jobs resume", "jobs delete"},
					Flags:       []string{"--format json", "--project", "--yes"},
					EnvVars:     []string{"STRAIT_PROJECT"},
				},
				{
					Name:        "run_triggering",
					Description: "Trigger job runs with arbitrary JSON payloads",
					Commands:    []string{"trigger", "jobs trigger"},
					Flags:       []string{"--payload", "--payload-file", "--idempotency-key", "--format json"},
				},
				{
					Name:        "run_management",
					Description: "List, get, cancel, and wait for job runs",
					Commands:    []string{"runs list", "runs get", "runs cancel", "wait"},
					Flags:       []string{"--format json", "--status", "--job"},
				},
				{
					Name:        "code_deployment",
					Description: "Deploy source code and manage code-first deployments",
					Commands:    []string{"deploy source", "deployments list", "deployments get", "deployments logs", "deployments watch", "deployments rollback"},
					Flags:       []string{"--job", "--runtime", "--dir", "--dry-run", "--no-stream", "--format json"},
					EnvVars:     []string{},
				},
				{
					Name:        "schema_introspection",
					Description: "Inspect resource schemas and supported runtimes without API calls",
					Commands:    []string{"schema runtimes", "schema job", "schema deployment", "schema workflow", "schema run"},
					Flags:       []string{},
				},
				{
					Name:        "health_diagnosis",
					Description: "Run comprehensive health and configuration checks",
					Commands:    []string{"doctor", "health", "health --ready"},
					Flags:       []string{"--json", "--verbose", "--check-endpoints", "--check-manifests"},
				},
				{
					Name:        "workflow_management",
					Description: "Create and manage multi-step workflows",
					Commands:    []string{"workflows list", "workflows get", "workflows create", "workflow-runs list"},
					Flags:       []string{"--format json"},
				},
				{
					Name:        "non_interactive_mode",
					Description: "Suppress all interactive prompts for use in CI and agent contexts",
					Commands:    []string{"(any command)"},
					Flags:       []string{"--non-interactive", "--yes", "--ci"},
					EnvVars:     []string{"STRAIT_NON_INTERACTIVE", "CI", "STRAIT_CI"},
				},
				{
					Name:        "structured_output",
					Description: "Emit machine-readable JSON on stdout for any command",
					Commands:    []string{"(any command)"},
					Flags:       []string{"--format json"},
					EnvVars:     []string{"STRAIT_FORMAT=json", "NO_COLOR"},
				},
				{
					Name:        "authentication",
					Description: "Authenticate with the Strait server using an API key",
					Commands:    []string{"login", "whoami", "api-keys list"},
					Flags:       []string{"--api-key", "--token"},
					EnvVars:     []string{"STRAIT_API_KEY"},
				},
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(caps)
		},
	}
}

// agentContext describes the current resolved environment for an agent.
type agentContext struct {
	ServerURL      string `json:"server_url"`
	ProjectID      string `json:"project_id"`
	APIKeySet      bool   `json:"api_key_set"`
	OutputFormat   string `json:"output_format"`
	NonInteractive bool   `json:"non_interactive"`
	CIMode         bool   `json:"ci_mode"`
	CLIVersion     string `json:"cli_version"`
	GoVersion      string `json:"go_version"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
}

func newAgentContextCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "context",
		Short: "Print the resolved CLI environment as JSON",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx := agentContext{
				ServerURL:      state.opts.serverURL,
				ProjectID:      state.opts.projectID,
				APIKeySet:      state.opts.apiKey != "",
				OutputFormat:   state.opts.outputFormat,
				NonInteractive: state.opts.nonInteractive,
				CIMode:         state.opts.ciMode,
				CLIVersion:     version,
				GoVersion:      runtime.Version(),
				OS:             runtime.GOOS,
				Arch:           runtime.GOARCH,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(ctx)
		},
	}
}

func newAgentDescribeCommand(_ *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "describe <command> [subcommand]",
		Short: "Describe a CLI command's flags and usage as JSON",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Walk the root command tree to find the target command.
			root := cmd.Root()
			target := root
			for _, arg := range args {
				found := false
				for _, sub := range target.Commands() {
					if sub.Name() == arg {
						target = sub
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("command %q not found under %q", arg, target.CommandPath())
				}
			}

			type flagDesc struct {
				Name      string `json:"name"`
				Shorthand string `json:"shorthand,omitempty"`
				Type      string `json:"type"`
				Default   string `json:"default"`
				Usage     string `json:"usage"`
			}

			type commandDesc struct {
				Command     string     `json:"command"`
				Use         string     `json:"use"`
				Short       string     `json:"short"`
				Long        string     `json:"long,omitempty"`
				Example     string     `json:"example,omitempty"`
				Subcommands []string   `json:"subcommands,omitempty"`
				Flags       []flagDesc `json:"flags,omitempty"`
			}

			desc := commandDesc{
				Command: target.CommandPath(),
				Use:     target.Use,
				Short:   target.Short,
			}
			if target.Long != "" {
				desc.Long = strings.TrimSpace(target.Long)
			}
			if target.Example != "" {
				desc.Example = strings.TrimSpace(target.Example)
			}
			for _, sub := range target.Commands() {
				if !sub.Hidden {
					desc.Subcommands = append(desc.Subcommands, sub.Name())
				}
			}
			appendFlags := func(fs *pflag.FlagSet) {
				fs.VisitAll(func(f *pflag.Flag) {
					desc.Flags = append(desc.Flags, flagDesc{
						Name:      f.Name,
						Shorthand: f.Shorthand,
						Type:      f.Value.Type(),
						Default:   f.DefValue,
						Usage:     f.Usage,
					})
				})
			}
			appendFlags(target.LocalFlags())
			appendFlags(target.InheritedFlags())

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(desc)
		},
	}
}

// agentSkill summarises one skill file.
type agentSkill struct {
	Name     string `json:"name"`
	File     string `json:"file"`
	Abstract string `json:"abstract"`
}

func newAgentSkillsCommand(_ *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "skills",
		Short: "List available agent skill files as JSON",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Locate the skills/ directory relative to the binary or CWD.
			candidates := []string{
				filepath.Join(execDir(), "skills"),
				"skills",
			}

			var skillsDir string
			for _, c := range candidates {
				if info, err := os.Stat(c); err == nil && info.IsDir() {
					skillsDir = c
					break
				}
			}

			skills := make([]agentSkill, 0)

			if skillsDir != "" {
				entries, err := os.ReadDir(skillsDir)
				if err == nil {
					for _, e := range entries {
						if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
							continue
						}
						path := filepath.Join(skillsDir, e.Name())
						abstract := extractSkillAbstract(path)
						name := strings.TrimSuffix(e.Name(), ".md")
						skills = append(skills, agentSkill{
							Name:     name,
							File:     path,
							Abstract: abstract,
						})
					}
				}
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(skills)
		},
	}
}

// extractSkillAbstract reads the first non-heading, non-empty line from a
// Markdown file to use as a brief description. The path is constructed from the
// skills directory listing and is trusted (not user input).
func extractSkillAbstract(path string) string {
	data, err := os.ReadFile(path) //nolint:gosec // path is from os.ReadDir of a known skills directory
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

// execDir returns the directory containing the running binary.
func execDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}
