package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/strait-dev/cli/internal/client"
	cliconfig "github.com/strait-dev/cli/internal/config"
	"github.com/strait-dev/cli/internal/validate"
)

var stdoutIsTTYFunc = func() bool {
	fi, err := os.Stdout.Stat() // printdata-ok: TTY detection on the actual fd, not a writer
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// mustMarkFlagRequired panics if MarkFlagRequired returns an error — which
// only happens when the flag name doesn't exist on the command. That is a
// programmer error caught at command construction (i.e. process startup),
// not a user-facing condition. Panicking guarantees we don't ship a binary
// where a "required" flag is silently optional because of a typo.
func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Sprintf("strait: MarkFlagRequired(%q) on %q: %v", name, cmd.Use, err))
	}
}

// idOrSlugLong returns a standardized Long: docstring for resource command
// groups whose sub-commands accept either a UUID or a slug as the identifier
// argument. Slugs require an active project context; UUIDs do not.
func idOrSlugLong(resource, shortDesc string) string {
	return fmt.Sprintf(`%s

Sub-commands that take an identifier argument accept either:
  - The %s UUID (resolved directly), or
  - The %s slug (resolved within the active project — set via
    --project, STRAIT_PROJECT_ID, or 'strait use <project>')`, shortDesc, resource, resource)
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

// resolveJobIdentifier accepts either a UUID or a slug and returns the
// canonical identifier the API expects (UUIDs pass through; slugs are
// resolved against the active project).
func resolveJobIdentifier(ctx context.Context, cli *client.Client, state *appState, idOrSlug string) (string, error) {
	if err := validate.SlugOrID(idOrSlug); err != nil {
		return "", fmt.Errorf("invalid job identifier: %w", err)
	}
	if validate.IsUUID(idOrSlug) {
		return idOrSlug, nil
	}
	_, err := cli.GetJob(ctx, idOrSlug)
	if err == nil {
		return idOrSlug, nil
	}
	if !client.IsNotFound(err) {
		return "", fmt.Errorf("resolving job %q: %w", idOrSlug, err)
	}

	projectID := state.opts.projectID
	if projectID == "" {
		return "", fmt.Errorf("project is required to resolve slug %q", idOrSlug)
	}

	jobs, lerr := cli.ListJobs(ctx, projectID)
	if lerr != nil {
		return "", fmt.Errorf("resolving job %q: %w", idOrSlug, lerr)
	}
	for _, job := range jobs {
		if job.Slug == idOrSlug {
			return job.ID, nil
		}
	}

	return "", fmt.Errorf("job %q not found", idOrSlug)
}

// parsePerfPeriodHours converts a human-friendly period string to hours for
// the analytics performance API.
func parsePerfPeriodHours(raw string) (int, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "24h":
		return 24, nil
	case "72h":
		return 72, nil
	case "7d":
		return 24 * 7, nil
	case "30d":
		return 24 * 30, nil
	case "90d":
		return 24 * 90, nil
	default:
		return 0, fmt.Errorf("invalid --period %q: use one of 24h, 72h, 7d, 30d, 90d", raw)
	}
}

// openBrowserFunc is a package-level indirection so tests can stub the
// browser-open behavior without spawning real processes.
var openBrowserFunc = openBrowser

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) //nolint:gosec // URL is derived from configured server URL
	case "linux":
		cmd = exec.Command("xdg-open", url) //nolint:gosec // URL is derived from configured server URL
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url) //nolint:gosec // URL is derived from configured server URL
	default:
		return fmt.Errorf("unsupported platform for browser open; visit: %s", url)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r // swallow panic from detached browser process
			}
		}()
		_ = cmd.Wait()
	}()
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

// requireOrgID resolves the organization ID from the --org flag, falling back
// to the resolved config/env value (STRAIT_ORG, config "org", or the active
// context's "org"). Org-scoped commands (billing, some usage views) use it so
// the org need only be set once rather than passed on every invocation.
func requireOrgID(state *appState, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if state.resolved.OrgID != "" {
		return state.resolved.OrgID, nil
	}
	return "", fmt.Errorf("organization ID is required (use --org or set STRAIT_ORG)")
}
