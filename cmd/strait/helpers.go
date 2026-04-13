package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/client"
	cliconfig "github.com/strait-dev/cli/internal/config"
)

var stdoutIsTTYFunc = func() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// debugTransport wraps an http.RoundTripper and logs method, URL, status, and
// latency to stderr for each request. It is activated by --debug.
type debugTransport struct {
	next http.RoundTripper
}

func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := d.next.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "debug: %s %s → error: %v (%s)\n", req.Method, req.URL, err, time.Since(start).Round(time.Millisecond))
		return resp, err
	}
	fmt.Fprintf(os.Stderr, "debug: %s %s → %d (%s)\n", req.Method, req.URL, resp.StatusCode, time.Since(start).Round(time.Millisecond))
	return resp, err
}

func loadConfigForWrite(state *appState) (*cliconfig.File, string, error) {
	path := state.configPath
	if path == "" {
		loaded, err := cliconfig.Load("")
		if err != nil {
			return nil, "", err
		}
		path = loaded.Path
		state.config = loaded.Data
		state.configPath = loaded.Path
	}

	loaded, err := cliconfig.Load(path)
	if err != nil {
		return nil, "", err
	}
	if loaded.Data == nil {
		return nil, "", fmt.Errorf("unable to load config")
	}

	state.config = loaded.Data
	state.configPath = loaded.Path

	return loaded.Data, loaded.Path, nil
}

func stdoutIsTTY() bool {
	return stdoutIsTTYFunc()
}

// isTTYRich returns true when styled/rich output should be used.
// False when an explicit machine-readable format is requested (json, yaml, csv, etc.)
// or when stdout is not a TTY (piped).
func isTTYRich(state *appState) bool {
	if !stdoutIsTTY() {
		return false
	}
	f := state.opts.outputFormat
	return f == "" || f == "table"
}

func newAPIClient(state *appState) (*client.Client, error) {
	c, err := client.New(state.opts.serverURL, state.opts.apiKey, state.opts.timeout)
	if err != nil {
		return nil, err
	}
	if state.opts.debug {
		c.SetTransport(&debugTransport{next: http.DefaultTransport})
	}
	return c, nil
}

// requireConfirmation checks CI/non-interactive mode and prompts interactively if needed.
// Pass yes=true when the user provided --yes flag.
func requireConfirmation(state *appState, msg string, yes bool) error {
	if yes {
		return nil
	}
	if state.opts.nonInteractive || state.opts.ciMode {
		return fmt.Errorf("interactive prompt blocked in non-interactive mode; use --yes to confirm")
	}
	if !stdoutIsTTY() {
		return fmt.Errorf("non-interactive terminal detected; use --yes to confirm")
	}
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", msg)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("operation cancelled")
	}
	return nil
}

// requireProjectID resolves the project ID from the flag value or appState default.
func requireProjectID(state *appState, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if state.opts.projectID != "" {
		return state.opts.projectID, nil
	}
	return "", fmt.Errorf("project ID is required (use --project)")
}
