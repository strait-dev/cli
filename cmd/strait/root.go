package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	cliauth "github.com/strait-dev/cli/internal/auth"
	cliconfig "github.com/strait-dev/cli/internal/config"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

type rootOptions struct {
	serverURL      string
	apiKey         string
	projectID      string
	outputFormat   string
	noHeaders      bool
	outputTpl      string
	outputPath     string
	noColor        bool
	quiet          bool
	verbose        bool
	contextName    string
	configPath     string
	timeout        time.Duration
	ciMode         bool
	nonInteractive bool
	debug          bool
}

type appState struct {
	opts       *rootOptions
	configPath string
	config     *cliconfig.File
	resolved   cliconfig.Resolved
	// stdout is where machine-readable output (printData, printQuietIDs,
	// renderAuditVerify, printLLMSManifest, etc.) is written. When nil it
	// defaults to os.Stdout. Tests inject a *bytes.Buffer to capture output
	// without swapping the global os.Stdout under a process-wide mutex,
	// which previously caused parallel-test races and flakes.
	stdout io.Writer
}

// out returns the writer for machine-readable output, defaulting to os.Stdout
// when no writer has been injected.
func (s *appState) out() io.Writer {
	if s.stdout == nil {
		return os.Stdout // printdata-ok: this method *is* the official accessor for stdout
	}
	return s.stdout
}

func newRootCommand() *cobra.Command {
	opts := &rootOptions{}
	state := &appState{opts: opts}
	var llms bool

	cmd := &cobra.Command{
		Use:           "strait",
		Short:         "Strait CLI — manage jobs, runs, and workflows",
		Long:          "Strait CLI manages jobs, runs, workflows, and other resources via the Strait REST API.",
		SilenceUsage:  true,
		SilenceErrors: true,
		// RunE handles `strait --llms` (no subcommand).
		RunE: func(cmd *cobra.Command, _ []string) error {
			if llms {
				return printLLMSManifest(state.out(), cmd)
			}
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			loaded, err := cliconfig.Load(opts.configPath)
			if err != nil {
				return err
			}

			if loaded.IsLocal && loaded.Exists {
				if fields := cliconfig.HasSensitiveLocalFields(loaded.Data); len(fields) > 0 {
					fmt.Fprintln(os.Stderr, styles.Warn("local config "+loaded.Path+" overrides: "+strings.Join(fields, ", ")))
				}
			}

			resolved := cliconfig.Resolve(cliconfig.ResolveInput{
				Flags: map[string]string{
					"server":  opts.serverURL,
					"api-key": opts.apiKey,
					"project": opts.projectID,
					"format":  opts.outputFormat,
					"context": opts.contextName,
				},
				BoolFlags: map[string]bool{
					"no-color": opts.noColor,
					"quiet":    opts.quiet,
					"verbose":  opts.verbose,
				},
				DurationFlags: map[string]string{
					"timeout": opts.timeout.String(),
				},
				Changed: map[string]bool{
					"server":   cmd.Flags().Changed("server"),
					"api-key":  cmd.Flags().Changed("api-key"),
					"project":  cmd.Flags().Changed("project"),
					"format":   cmd.Flags().Changed("format"),
					"context":  cmd.Flags().Changed("context"),
					"no-color": cmd.Flags().Changed("no-color"),
					"quiet":    cmd.Flags().Changed("quiet"),
					"verbose":  cmd.Flags().Changed("verbose"),
					"timeout":  cmd.Flags().Changed("timeout"),
				},
				Config:          loaded.Data,
				Env:             cliEnv(),
				ContextOverride: opts.contextName,
			})

			if resolved.Credential == "" {
				if key, keyErr := cliauth.LoadAPIKey(resolved.ContextName); keyErr == nil {
					resolved.Credential = key
				}
			}

			timeout, parseErr := time.ParseDuration(resolved.Timeout)
			if parseErr != nil {
				return fmt.Errorf("invalid timeout %q: %w", resolved.Timeout, parseErr)
			}

			opts.serverURL = resolved.ServerURL
			opts.apiKey = resolved.Credential
			opts.projectID = resolved.ProjectID
			opts.outputFormat = resolved.Format
			opts.contextName = resolved.ContextName
			opts.noColor = resolved.NoColor
			opts.quiet = resolved.Quiet
			opts.verbose = resolved.Verbose
			opts.timeout = timeout
			opts.configPath = loaded.Path

			state.configPath = loaded.Path
			state.config = loaded.Data
			state.resolved = resolved

			if opts.ciMode || os.Getenv("STRAIT_CI") == "true" || os.Getenv("CI") == "true" {
				opts.ciMode = true
				opts.noColor = true
			}

			if opts.nonInteractive || os.Getenv("STRAIT_NON_INTERACTIVE") == "1" || os.Getenv("STRAIT_NON_INTERACTIVE") == "true" {
				opts.nonInteractive = true
			}

			// CI mode implies non-interactive.
			if opts.ciMode {
				opts.nonInteractive = true
			}

			if opts.noColor {
				styles.ForceNoColor()
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&opts.serverURL, "server", "", "server URL")
	cmd.PersistentFlags().StringVar(&opts.apiKey, "api-key", "", "API key")
	cmd.PersistentFlags().StringVar(&opts.projectID, "project", "", "default project ID")
	cmd.PersistentFlags().StringVarP(&opts.outputFormat, "format", "o", "", "output format")
	cmd.PersistentFlags().BoolVar(&opts.noHeaders, "no-headers", false, "omit headers for table output")
	cmd.PersistentFlags().StringVar(&opts.outputTpl, "output-template", "", "go template for --format go-template")
	cmd.PersistentFlags().StringVar(&opts.outputPath, "output-jsonpath", "", "jsonpath for --format jsonpath")
	cmd.PersistentFlags().BoolVar(&opts.noColor, "no-color", false, "disable color output")
	cmd.PersistentFlags().BoolVarP(&opts.quiet, "quiet", "q", false, "minimal output")
	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")
	cmd.PersistentFlags().StringVar(&opts.contextName, "context", "", "context name override")
	cmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "config file path")
	cmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", 30*time.Second, "API request timeout")
	cmd.PersistentFlags().BoolVar(&opts.ciMode, "ci", false, "enable CI mode (no color, no prompts)")
	cmd.PersistentFlags().BoolVar(&opts.nonInteractive, "non-interactive", false, "disable interactive prompts (also set via STRAIT_NON_INTERACTIVE=1)")
	cmd.PersistentFlags().BoolVar(&opts.debug, "debug", false, "print HTTP request/response details to stderr")
	cmd.Flags().BoolVar(&llms, "llms", false, "print full CLI command manifest as JSON for LLM consumption and exit")

	// Canonical CLI surface (orchestration-only).
	cmd.AddCommand(newVersionCommand(state))
	cmd.AddCommand(newCompletionCommand(state, cmd))
	cmd.AddCommand(newContextCommand(state))
	cmd.AddCommand(newAliasCommand(state))
	cmd.AddCommand(newAuthCommand(state))
	cmd.AddCommand(newJobsCommand(state))
	cmd.AddCommand(newRunsCommand(state))
	cmd.AddCommand(newWorkflowsCommand(state))
	cmd.AddCommand(newWorkflowRunsCommand(state))
	cmd.AddCommand(newAPIKeysCommand(state))
	cmd.AddCommand(newWaitCommand(state))
	cmd.AddCommand(newLogsCommand(state))
	cmd.AddCommand(newTriggersCommand(state))
	cmd.AddCommand(newSecretsCommand(state))
	cmd.AddCommand(newExtensionCommand(state))
	cmd.AddCommand(newUpgradeCommand(state))
	cmd.AddCommand(newProjectsCommand(state))
	cmd.AddCommand(newDebugCommand(state))
	cmd.AddCommand(newTeamCommand(state))
	cmd.AddCommand(newConfigCommand(state))
	cmd.AddCommand(newEnvCommand(state))
	cmd.AddCommand(newWebhooksCommand(state))
	cmd.AddCommand(newEventSourcesCommand(state))
	cmd.AddCommand(newLogDrainsCommand(state))
	cmd.AddCommand(newUsageCommand(state))
	cmd.AddCommand(newAnalyticsCommand(state))
	cmd.AddCommand(newInitCommand(state))
	cmd.AddCommand(newMigrateCommand(state))
	cmd.AddCommand(newTUICommand(state))

	// Migration stubs for high-traffic legacy command names. These exit
	// with a non-zero status and a styled error pointing the user at the
	// canonical replacement. Hidden from --help.
	for _, stub := range legacyMigrationStubs() {
		cmd.AddCommand(newDeprecatedStubCommand(stub.name, stub.message))
	}

	rawArgs := os.Args[1:]
	configPath := extractConfigPath(rawArgs)
	rawArgs = expandAliasArgs(rawArgs, configPath)
	cmd.SetArgs(rawArgs)

	registerRootCompletions(cmd)

	return cmd
}

func registerRootCompletions(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("format", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "jsonl", "compact", "yaml", "csv", "wide", "go-template", "jsonpath"}, cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.RegisterFlagCompletionFunc("context", func(c *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		configPath := ""
		if flag := c.Flag("config"); flag != nil {
			configPath = strings.TrimSpace(flag.Value.String())
		}
		loaded, err := cliconfig.Load(configPath)
		if err != nil || loaded == nil || loaded.Data == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		results := make([]string, 0, len(loaded.Data.Contexts))
		for name := range loaded.Data.Contexts {
			results = append(results, name)
		}
		return results, cobra.ShellCompDirectiveNoFileComp
	})
}

func extractConfigPath(args []string) string {
	for i := range len(args) {
		if args[i] == "--config" && i+1 < len(args) {
			return strings.TrimSpace(args[i+1])
		}
		if value, ok := strings.CutPrefix(args[i], "--config="); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func newVersionCommand(state *appState) *cobra.Command {
	var short bool
	var asJSON bool
	var checkServer bool
	var checkUpdate bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print CLI version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			w := state.out()
			if short {
				fmt.Fprintln(w, version)
				return nil
			}

			info := map[string]string{
				"version": version,
				"commit":  commit,
				"date":    date,
				"go":      runtime.Version(),
				"os_arch": fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			}

			if checkServer {
				serverStatus := "unreachable"
				client := &http.Client{Timeout: state.opts.timeout}
				resp, err := client.Get(strings.TrimRight(state.opts.serverURL, "/") + "/health")
				if err == nil {
					_ = resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						serverStatus = "reachable"
					} else {
						serverStatus = fmt.Sprintf("http_%d", resp.StatusCode)
					}
				}
				info["server"] = serverStatus
			}

			if asJSON {
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}

			if isTTYRich(state) {
				fmt.Fprintln(w, styles.TitleStyle.Render("Strait CLI"))
				fmt.Fprintln(w, styles.KeyValue("Version", info["version"]))
				fmt.Fprintln(w, styles.KeyValue("Commit", info["commit"]))
				fmt.Fprintln(w, styles.KeyValue("Date", info["date"]))
				fmt.Fprintln(w, styles.KeyValue("Go", info["go"]))
				fmt.Fprintln(w, styles.KeyValue("OS/Arch", info["os_arch"]))
				if checkServer {
					fmt.Fprintln(w, styles.KeyValue("Server", info["server"]))
				}
			} else {
				fmt.Fprintf(w, "version: %s\n", info["version"])
				fmt.Fprintf(w, "commit: %s\n", info["commit"])
				fmt.Fprintf(w, "date: %s\n", info["date"])
				fmt.Fprintf(w, "go: %s\n", info["go"])
				fmt.Fprintf(w, "os/arch: %s\n", info["os_arch"])
				if checkServer {
					fmt.Fprintf(w, "server: %s\n", info["server"])
				}
			}

			if checkUpdate {
				latest, cached := getCachedUpdate()
				if !cached {
					latest = checkForUpdate()
					if latest != "" {
						setCachedUpdate(latest)
					}
				}
				if latest != "" {
					current := strings.TrimPrefix(version, "v")
					if current == latest {
						fmt.Fprintln(w, "update: up to date")
					} else {
						fmt.Fprintf(w, "update: v%s available (current: v%s)\n", latest, current)
					}
				} else {
					fmt.Fprintln(w, "update: check failed")
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "print only the version number")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print version information as JSON")
	cmd.Flags().BoolVar(&checkServer, "check-server", false, "check configured server health endpoint")
	cmd.Flags().BoolVar(&checkUpdate, "check-update", false, "check for newer CLI version")

	return cmd
}

func newCompletionCommand(state *appState, root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			w := state.out()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(w)
			case "zsh":
				return root.GenZshCompletion(w)
			case "fish":
				return root.GenFishCompletion(w, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(w)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}

	return cmd
}

func cliEnv() map[string]string {
	return map[string]string{
		"STRAIT_SERVER":          strings.TrimSpace(os.Getenv("STRAIT_SERVER")),
		"STRAIT_API_KEY":         strings.TrimSpace(os.Getenv("STRAIT_API_KEY")),
		"STRAIT_PROJECT":         strings.TrimSpace(os.Getenv("STRAIT_PROJECT")),
		"STRAIT_FORMAT":          strings.TrimSpace(os.Getenv("STRAIT_FORMAT")),
		"STRAIT_CONTEXT":         strings.TrimSpace(os.Getenv("STRAIT_CONTEXT")),
		"NO_COLOR":               strings.TrimSpace(os.Getenv("NO_COLOR")),
		"STRAIT_CI":              strings.TrimSpace(os.Getenv("STRAIT_CI")),
		"STRAIT_NON_INTERACTIVE": strings.TrimSpace(os.Getenv("STRAIT_NON_INTERACTIVE")),
	}
}
