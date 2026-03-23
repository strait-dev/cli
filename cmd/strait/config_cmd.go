package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	cliconfig "github.com/strait-dev/cli/internal/config"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newConfigCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}

	cmd.AddCommand(newConfigEditCommand(state))
	cmd.AddCommand(newConfigPathCommand(state))

	return cmd
}

func newConfigEditCommand(state *appState) *cobra.Command {
	var editor string

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open config file in your editor",
		Long: `Opens the CLI configuration file in your preferred editor.

Uses --editor flag, then $EDITOR environment variable, then falls back to vi.
After saving, validates the config file is still loadable.`,
		Example: `  strait config edit
  strait config edit --editor nano
  EDITOR=code strait config edit`,
		RunE: func(_ *cobra.Command, _ []string) error {
			configPath := state.configPath
			if configPath == "" {
				homePath, err := cliconfig.HomePath()
				if err != nil {
					return fmt.Errorf("resolve config path: %w", err)
				}
				configPath = homePath
			}

			// Ensure the config file exists.
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				dir := configPath[:strings.LastIndex(configPath, "/")]
				if err := os.MkdirAll(dir, 0o700); err != nil {
					return fmt.Errorf("create config directory: %w", err)
				}
				if err := os.WriteFile(configPath, []byte("# Strait CLI configuration\n"), 0o600); err != nil {
					return fmt.Errorf("create config file: %w", err)
				}
			}

			editorCmd := strings.TrimSpace(editor)
			if editorCmd == "" {
				editorCmd = strings.TrimSpace(os.Getenv("EDITOR"))
			}
			if editorCmd == "" {
				editorCmd = "vi"
			}

			cmd := exec.Command(editorCmd, configPath) //nolint:gosec // editor from $EDITOR env or --editor flag
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("editor exited with error: %w", err)
			}

			// Validate the config after editing.
			if _, loadErr := cliconfig.Load(configPath); loadErr != nil {
				fmt.Fprintf(os.Stderr, "%s\n", styles.Warn("Config file has errors after editing: "+loadErr.Error()))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&editor, "editor", "", "editor command (default: $EDITOR or vi)")

	return cmd
}

func newConfigPathCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		RunE: func(_ *cobra.Command, _ []string) error {
			configPath := state.configPath
			if configPath == "" {
				homePath, err := cliconfig.HomePath()
				if err != nil {
					return fmt.Errorf("resolve config path: %w", err)
				}
				configPath = homePath
			}
			fmt.Println(configPath)
			return nil
		},
	}
}
