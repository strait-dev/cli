package main

import (
	"fmt"
	"os"

	"github.com/strait-dev/cli/internal/codedeploy"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

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
			if runtime == "" {
				return fmt.Errorf("--runtime is required (go, python, typescript, ruby, rust)")
			}

			resolvedProject, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			if !dryRun && isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Info(fmt.Sprintf("Deploying source for job %s (runtime: %s)", jobSlug, runtime)))
			}

			var onLog func(string)
			if !noStream {
				onLog = func(chunk string) {
					fmt.Fprint(os.Stderr, chunk)
				}
			}

			res, runErr := codedeploy.Run(cmd.Context(), cli, codedeploy.Options{
				ProjectID:  resolvedProject,
				JobSlug:    jobSlug,
				Runtime:    runtime,
				SourceDir:  sourceDir,
				IgnoreFile: ignoreFile,
				DryRun:     dryRun,
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

			if dryRun {
				return nil
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
	cmd.Flags().StringVar(&runtime, "runtime", "", "language runtime: go, python, typescript, ruby, rust (required)")
	cmd.Flags().StringVar(&sourceDir, "dir", ".", "source directory to pack (default: current directory)")
	cmd.Flags().StringVar(&ignoreFile, "ignore-file", "", "custom ignore file (default: .straitignore in source dir)")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID (overrides --project from root)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "pack and validate without uploading")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "disable real-time build log streaming")

	return cmd
}
