package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/strait-dev/cli/internal/codedeploy"
	"github.com/strait-dev/cli/internal/pack"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

// formatUploadSize formats bytes as a human-readable string for upload progress display.
func formatUploadSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/mb)
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/kb)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func newDeploySourceCommand(state *appState) *cobra.Command {
	var (
		jobSlug    string
		runtime    string
		sourceDir  string
		ignoreFile string
		projectID  string
		dryRun     bool
		noStream   bool
	)

	cmd := &cobra.Command{
		Use:   "source",
		Short: "Deploy source code directly (code-first deployment)",
		Long: `Pack the source directory, upload it to Strait, and trigger a BuildKit build.

The build runs server-side. Build logs are streamed in real time.
On success the job switches to use the newly built image.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if jobSlug == "" {
				return fmt.Errorf("--job is required")
			}

			// Auto-detect runtime when not provided.
			if runtime == "" {
				detected, ok := codedeploy.DetectRuntime(sourceDir)
				if !ok {
					return fmt.Errorf("--runtime is required: could not detect runtime from %s (expected go.mod, package.json, requirements.txt, Cargo.toml, or Gemfile)", sourceDir)
				}
				runtime = detected
				if isTTYRich(state) {
					fmt.Fprintln(os.Stderr, styles.MutedStyle.Render(fmt.Sprintf("Detected runtime: %s", runtime)))
				}
			}

			resolvedProject, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			// Dry-run: list files without uploading.
			if dryRun {
				files, listErr := pack.ListContents(sourceDir, ignoreFile)
				if listErr != nil {
					return fmt.Errorf("list source contents: %w", listErr)
				}
				sort.Strings(files)
				normalised := codedeploy.NormalizeRuntime(runtime)
				if isTTYRich(state) {
					fmt.Fprintf(os.Stderr, "[dry-run] %d files would be packed from %s (runtime: %s)\n", len(files), sourceDir, normalised)
					for _, f := range files {
						fmt.Fprintf(os.Stderr, "  %s\n", styles.MutedStyle.Render(f))
					}
				} else {
					rows := make([]map[string]any, 0, len(files))
					for _, f := range files {
						rows = append(rows, map[string]any{"file": f})
					}
					return printData(state, rows)
				}
				return nil
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Info(fmt.Sprintf("Deploying source for job %s (runtime: %s)", jobSlug, codedeploy.NormalizeRuntime(runtime))))
			}

			var onLog func(string)
			if !noStream {
				onLog = func(chunk string) {
					fmt.Fprint(os.Stderr, chunk)
				}
			}

			// Upload progress: only in TTY-rich mode, printed inline using \r.
			var onUploadProgress func(read, total int64)
			if isTTYRich(state) && !state.opts.quiet {
				var lastPct int
				onUploadProgress = func(read, total int64) {
					if total <= 0 {
						return
					}
					pct := int(float64(read) / float64(total) * 100)
					if pct == lastPct && read < total {
						return // throttle: only print on 1% increments or completion
					}
					lastPct = pct
					fmt.Fprintf(os.Stderr, "\r  %s %s / %s (%d%%)     ",
						styles.MutedStyle.Render("Uploading..."),
						formatUploadSize(read),
						formatUploadSize(total),
						pct,
					)
					if read >= total {
						fmt.Fprintln(os.Stderr) // newline on completion
					}
				}
			}

			res, runErr := codedeploy.Run(cmd.Context(), cli, codedeploy.Options{
				ProjectID:        resolvedProject,
				JobSlug:          jobSlug,
				Runtime:          runtime,
				SourceDir:        sourceDir,
				IgnoreFile:       ignoreFile,
				OnUploadProgress: onUploadProgress,
				OnProgress: func(msg string) {
					if isTTYRich(state) && !state.opts.quiet {
						fmt.Fprintln(os.Stderr, styles.MutedStyle.Render(msg))
					}
				},
				OnLogChunk: onLog,
			})
			if runErr != nil {
				return runErr
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success(fmt.Sprintf(
					"Deployment %s ready", res.DeploymentID,
				)))
				if res.ImageURI != "" {
					fmt.Fprintln(os.Stderr, styles.KeyValue("Image", res.ImageURI))
				}
				return nil
			}
			return printData(state, map[string]any{
				"deployment_id": res.DeploymentID,
				"status":        res.Status,
				"image_uri":     res.ImageURI,
			})
		},
	}

	cmd.Flags().StringVar(&jobSlug, "job", "", "job slug to deploy (required)")
	cmd.Flags().StringVar(&runtime, "runtime", "", "language runtime: go, python, typescript (node/bun), ruby, rust (auto-detected when omitted)")
	cmd.Flags().StringVar(&sourceDir, "dir", ".", "source directory to pack (default: current directory)")
	cmd.Flags().StringVar(&ignoreFile, "ignore-file", "", "custom ignore file (default: .straitignore in source dir)")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID (overrides --project from root)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "pack and validate without uploading")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "disable real-time build log streaming")

	return cmd
}
