// Package codedeploy orchestrates the code-first deployment pipeline:
// pack source → presign → upload tarball → confirm → stream build logs.
package codedeploy

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/pack"
)

// Options controls a code-first deployment.
type Options struct {
	// ProjectID is required.
	ProjectID string
	// JobSlug is the job slug; the job ID is resolved via the API.
	JobSlug string
	// Runtime is the language runtime: go, python, typescript, ruby, rust.
	Runtime string
	// SourceDir is the directory to pack. Defaults to ".".
	SourceDir string
	// IgnoreFile overrides the default .straitignore discovery.
	IgnoreFile string
	// DryRun stops after packing and prints what would be uploaded.
	DryRun bool
	// OnProgress is called with status updates; may be nil.
	OnProgress func(msg string)
	// OnLogChunk is called with each raw log line from the build; may be nil.
	OnLogChunk func(chunk string)
}

// Result is returned on success.
type Result struct {
	DeploymentID string
	Status       string
	ImageURI     string
}

// Run executes the full code-first deployment pipeline.
func Run(ctx context.Context, cli *client.Client, opts Options) (*Result, error) {
	if opts.SourceDir == "" {
		opts.SourceDir = "."
	}

	progress := opts.OnProgress
	if progress == nil {
		progress = func(string) {}
	}

	// 1. Pack source code.
	progress("Packing source directory...")
	packed, err := pack.Pack(opts.SourceDir, opts.IgnoreFile)
	if err != nil {
		return nil, fmt.Errorf("pack source: %w", err)
	}
	defer os.Remove(packed.Path)

	progress(fmt.Sprintf("Packed %s (SHA-256: %s, size: %s)",
		opts.SourceDir, packed.Hash[:12]+"...", formatSize(packed.Size)))

	if opts.DryRun {
		progress(fmt.Sprintf("[dry-run] would upload %s (%d bytes, hash: %s) for job %s/%s runtime %s",
			opts.SourceDir, packed.Size, packed.Hash, opts.ProjectID, opts.JobSlug, opts.Runtime))
		return &Result{}, nil
	}

	// 2. Resolve job ID from slug.
	progress(fmt.Sprintf("Looking up job %q...", opts.JobSlug))
	job, err := cli.GetJobBySlug(ctx, opts.ProjectID, opts.JobSlug)
	if err != nil {
		return nil, fmt.Errorf("look up job: %w", err)
	}

	// 3. Create deployment record and get presigned URL.
	progress("Creating deployment record...")
	createResp, err := cli.CreateCodeDeployment(ctx, job.ID, client.CreateCodeDeploymentRequest{
		ProjectID:       opts.ProjectID,
		JobID:           job.ID,
		Runtime:         opts.Runtime,
		SourceHash:      packed.Hash,
		SourceSizeBytes: packed.Size,
	})
	if err != nil {
		return nil, fmt.Errorf("create deployment: %w", err)
	}
	deploymentID := createResp.Deployment.ID
	progress(fmt.Sprintf("Created deployment %s", deploymentID))

	// 4. Upload tarball to presigned URL.
	progress("Uploading source tarball...")
	f, err := os.Open(packed.Path)
	if err != nil {
		return nil, fmt.Errorf("open packed archive: %w", err)
	}
	defer f.Close()

	if err := cli.UploadFile(ctx, createResp.UploadURL, f, packed.Size); err != nil {
		return nil, fmt.Errorf("upload tarball: %w", err)
	}
	progress("Upload complete")

	// 5. Confirm deployment (triggers build).
	progress("Confirming deployment (triggering build)...")
	d, err := cli.ConfirmCodeDeployment(ctx, job.ID, deploymentID, client.ConfirmCodeDeploymentRequest{
		ProjectID: opts.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("confirm deployment: %w", err)
	}
	progress(fmt.Sprintf("Build started (deployment %s, status: %s)", d.ID, d.Status))

	// 6. Stream build logs.
	if opts.OnLogChunk != nil {
		progress("Streaming build logs...")
		onChunk := opts.OnLogChunk
		streamErr := cli.StreamDeploymentLogs(ctx, job.ID, deploymentID, func(chunk string) error {
			onChunk(chunk)
			return nil
		})
		if streamErr != nil && ctx.Err() == nil {
			// Non-fatal: logs stream ended early; fall through to polling.
			progress(fmt.Sprintf("Log stream ended: %v", streamErr))
		}
	}

	// 7. Poll until terminal status.
	progress("Waiting for build to complete...")
	final, err := pollUntilTerminal(ctx, cli, job.ID, deploymentID, progress)
	if err != nil {
		return nil, err
	}

	switch final.Status {
	case "ready":
		return &Result{
			DeploymentID: final.ID,
			Status:       final.Status,
			ImageURI:     final.BuiltImageURI,
		}, nil
	case "timed_out":
		return nil, fmt.Errorf("build timed out (deployment %s)", final.ID)
	default:
		msg := final.ErrorMessage
		if msg == "" {
			msg = "build failed"
		}
		return nil, fmt.Errorf("deployment %s failed: %s", final.ID, msg)
	}
}

// pollUntilTerminal polls GetCodeDeployment with exponential backoff until
// the deployment reaches a terminal state (ready, failed, timed_out).
func pollUntilTerminal(ctx context.Context, cli *client.Client, jobID, deploymentID string, progress func(string)) (*client.CodeDeployment, error) {
	const (
		maxWait     = 30 * time.Minute
		minInterval = 3 * time.Second
		maxInterval = 30 * time.Second
	)

	deadline := time.Now().Add(maxWait)
	interval := minInterval

	for {
		d, err := cli.GetCodeDeployment(ctx, jobID, deploymentID)
		if err != nil {
			return nil, fmt.Errorf("poll deployment status: %w", err)
		}

		switch d.Status {
		case "ready", "failed", "timed_out":
			return d, nil
		case "building", "pending":
			// continue polling
		default:
			return nil, fmt.Errorf("unexpected deployment status: %s", d.Status)
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("deployment %s did not reach terminal status within %s", deploymentID, maxWait)
		}

		progress(fmt.Sprintf("  status: %s (polling in %s)", d.Status, interval))

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		interval *= 2
		if interval > maxInterval {
			interval = maxInterval
		}
	}
}

// formatSize returns a human-readable file size.
func formatSize(bytes int64) string {
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

// GetBuildLogs retrieves stored build logs for a terminal deployment.
func GetBuildLogs(ctx context.Context, cli *client.Client, jobID, deploymentID string) (string, error) {
	d, err := cli.GetCodeDeployment(ctx, jobID, deploymentID)
	if err != nil {
		return "", err
	}
	return d.BuildLogs, nil
}

// ReadLogsNonStreaming fetches the logs body as plain text via GET without streaming.
func ReadLogsNonStreaming(ctx context.Context, httpClient interface {
	GetCodeDeployment(context.Context, string, string) (*client.CodeDeployment, error)
}, jobID, deploymentID string) (string, error) {
	d, err := httpClient.GetCodeDeployment(ctx, jobID, deploymentID)
	if err != nil {
		return "", err
	}
	_ = io.Discard
	return d.BuildLogs, nil
}
