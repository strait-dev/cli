package main

import (
	"fmt"
	"os"

	cliauth "github.com/strait-dev/cli/internal/auth"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newWhoamiCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show authenticated user and context info",
		Long: `Display the current authentication state, active context, server URL,
project ID, and API key status. If authenticated, verifies connectivity.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			contextName := state.opts.contextName
			if contextName == "" {
				contextName = "default"
			}

			info := map[string]any{
				"context":     contextName,
				"server":      state.opts.serverURL,
				"project":     state.opts.projectID,
				"config_path": state.configPath,
			}

			authenticated := state.opts.apiKey != ""
			info["authenticated"] = authenticated
			if authenticated {
				info["key_prefix"] = cliauth.MaskAPIKey(state.opts.apiKey)
			}

			// If authenticated, verify server connectivity.
			if authenticated {
				cli, err := newAPIClient(state)
				if err == nil {
					_, healthErr := cli.Health(cmd.Context())
					if healthErr == nil {
						info["server_status"] = "reachable"
					} else {
						info["server_status"] = "unreachable"
					}
				}
			}

			if isTTYRich(state) {
				lines := []string{
					styles.DetailLine("Context", contextName),
					styles.DetailLine("Server", state.opts.serverURL),
					styles.DetailLine("Project", state.opts.projectID),
				}
				if authenticated {
					lines = append(lines, styles.DetailLine("API Key", cliauth.MaskAPIKey(state.opts.apiKey)))
					if status, ok := info["server_status"].(string); ok {
						lines = append(lines, styles.DetailLine("Server Status", status))
					}
				} else {
					lines = append(lines, styles.DetailLine("API Key", "not set"))
				}
				lines = append(lines, styles.DetailLine("Config", state.configPath))
				fmt.Fprint(os.Stderr, styles.DetailBox("Who Am I", lines))
				return nil
			}
			return printData(state, info)
		},
	}

	return cmd
}
