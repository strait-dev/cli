package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/strait-dev/cli/internal/styles"
)

var version = "0.2.0-dev"
var commit = "none"
var date = "unknown"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	code := run(ctx)
	cancel()
	os.Exit(code)
}

func run(ctx context.Context) (exitCode int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "fatal: %v\n", r)
			exitCode = ExitPanic
		}
	}()

	if err := newRootCommand().ExecuteContext(ctx); err != nil {
		code := exitCodeFromError(err)
		if isJSONOutputMode() {
			enc := json.NewEncoder(os.Stdout) // printdata-ok: top-level fatal handler runs without appState in scope
			out := map[string]any{
				"error":     err.Error(),
				"exit_code": code,
				"code":      exitCodeName(code),
			}
			if s := errorSuggestion(code); s != "" {
				out["suggestion"] = s
			}
			if u := errorDocsURL(code); u != "" {
				out["docs_url"] = u
			}
			_ = enc.Encode(out)
		} else {
			fmt.Fprintln(os.Stderr, formatCLIError(err))
		}
		return code
	}
	return ExitOK
}

// isJSONOutputMode reports whether the current invocation is requesting JSON or
// JSONL output. It checks the STRAIT_FORMAT env var and scans os.Args for
// --format / -o flags without depending on cobra flag parsing (which runs after
// this point when an error occurs).
func isJSONOutputMode() bool {
	if f := strings.TrimSpace(os.Getenv("STRAIT_FORMAT")); f == "json" || f == "jsonl" || f == "compact" {
		return true
	}
	args := os.Args[1:]
	for i, arg := range args {
		if (arg == "--format" || arg == "-o") && i+1 < len(args) {
			f := strings.TrimSpace(args[i+1])
			return f == "json" || f == "jsonl" || f == "compact"
		}
		for _, prefix := range []string{"--format=", "-o="} {
			if f, ok := strings.CutPrefix(arg, prefix); ok {
				return f == "json" || f == "jsonl" || f == "compact"
			}
		}
	}
	return false
}

// formatCLIError turns a raw Go error into a human-friendly styled message.
func formatCLIError(err error) string {
	msg := err.Error()

	// Parse "request failed (409): message" pattern from API client.
	if strings.HasPrefix(msg, "request failed (") {
		parts := strings.SplitN(msg, ": ", 2)
		if len(parts) == 2 {
			code := strings.TrimPrefix(parts[0], "request failed ")
			detail := parts[1]

			hint := ""
			switch {
			case strings.Contains(detail, "already exists") || strings.Contains(detail, "conflict"):
				hint = "\n  " + styles.MutedStyle.Render("Use a different name/slug, or update the existing resource.")
			case strings.Contains(detail, "not found"):
				hint = "\n  " + styles.MutedStyle.Render("Check the ID or slug with `strait jobs list` or `strait runs list`.")
			case strings.Contains(detail, "invalid or missing"):
				hint = "\n  " + styles.MutedStyle.Render("Run `strait login` to authenticate or check your API key.")
			case strings.Contains(detail, "unauthorized") || strings.Contains(detail, "permission"):
				hint = "\n  " + styles.MutedStyle.Render("Your API key may lack the required scope. Check with `strait api-keys list`.")
			}

			return styles.Err(detail+" "+styles.MutedStyle.Render(code)) + hint
		}
	}

	// Parse "resolving job ..." wrapper.
	if strings.HasPrefix(msg, "resolving job ") || strings.HasPrefix(msg, "resolving workflow ") {
		inner := msg
		if idx := strings.Index(msg, ": "); idx > 0 {
			inner = msg[idx+2:]
		}
		return styles.Err(inner) + "\n  " + styles.MutedStyle.Render("Check the slug exists with `strait jobs list`.")
	}

	// Generic: "project ID is required".
	if strings.Contains(msg, "project ID is required") {
		return styles.Err("No project specified.") + "\n  " + styles.MutedStyle.Render("Set STRAIT_PROJECT or use --project <id>.")
	}

	return styles.Err(msg)
}
