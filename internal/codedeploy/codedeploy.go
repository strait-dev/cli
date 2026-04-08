// Package codedeploy orchestrates the code-first deployment pipeline:
// pack source → presign → upload tarball → confirm → stream build logs.
package codedeploy

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/pack"
)

// maxSourceBytes is the maximum tarball size accepted by the server (256 MB).
const maxSourceBytes = 256 * 1024 * 1024

// serverRuntimes is the set of runtime values the server accepts.
var serverRuntimes = map[string]bool{
	"go": true, "python": true, "typescript": true, "ruby": true, "rust": true,
}

// NormalizeRuntime maps CLI-friendly aliases (node, bun, js) to the canonical
// server runtime names. Returns the original value unchanged if no mapping exists.
func NormalizeRuntime(r string) string {
	switch r {
	case "node", "bun", "js":
		return "typescript"
	}
	return r
}

// DetectRuntime inspects dir for well-known project files and returns the
// best-matching server runtime. Returns ("", false) when nothing is recognised.
func DetectRuntime(dir string) (runtime string, ok bool) {
	markers := []struct {
		file    string
		runtime string
	}{
		{"go.mod", "go"},
		{"Cargo.toml", "rust"},
		{"Gemfile", "ruby"},
		{"requirements.txt", "python"},
		{"pyproject.toml", "python"},
		{"setup.py", "python"},
		{"package.json", "typescript"},
		{"bun.lockb", "typescript"},
	}
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(dir, m.file)); err == nil {
			return m.runtime, true
		}
	}
	return "", false
}

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
	// OnProgress is called with status updates; may be nil.
	OnProgress func(msg string)
	// OnUploadProgress is called during tarball upload with bytes read and total size.
	// May be nil. Called frequently — throttle rendering on the caller side if needed.
	OnUploadProgress func(read, total int64)
	// OnLogChunk is called with each raw log line from the build; may be nil.
	OnLogChunk func(chunk string)
}

// progressReader wraps an io.Reader and calls onProgress with cumulative bytes read.
type progressReader struct {
	r          io.Reader
	total      int64
	read       int64
	onProgress func(read, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.read += int64(n)
		p.onProgress(p.read, p.total)
	}
	return n, err
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

	// 1. Resolve and validate runtime.
	runtime := NormalizeRuntime(opts.Runtime)
	if !serverRuntimes[runtime] {
		return nil, fmt.Errorf("unsupported runtime %q: must be one of go, python, typescript, ruby, rust", opts.Runtime)
	}

	// 2. Pack source code.
	progress("Packing source directory...")
	packed, err := pack.Pack(opts.SourceDir, opts.IgnoreFile)
	if err != nil {
		return nil, fmt.Errorf("pack source: %w", err)
	}
	defer os.Remove(packed.Path)

	// 3. Client-side size gate (server enforces 256 MB; fail fast before upload).
	if packed.Size > maxSourceBytes {
		return nil, fmt.Errorf(
			"packed archive is %.1f MB — exceeds the 256 MB server limit; add more entries to .straitignore",
			float64(packed.Size)/1024/1024,
		)
	}

	progress(fmt.Sprintf("Packed %s (SHA-256: %s, size: %s)",
		opts.SourceDir, packed.Hash[:12]+"...", formatSize(packed.Size)))

	// 4. Resolve job ID from slug.
	progress(fmt.Sprintf("Looking up job %q...", opts.JobSlug))
	job, err := cli.GetJobBySlug(ctx, opts.ProjectID, opts.JobSlug)
	if err != nil {
		return nil, fmt.Errorf("look up job: %w", err)
	}

	// 5. Create deployment record and get presigned URL.
	progress("Creating deployment record...")
	createResp, err := cli.CreateCodeDeployment(ctx, job.ID, client.CreateCodeDeploymentRequest{
		ProjectID:       opts.ProjectID,
		JobID:           job.ID,
		Runtime:         runtime,
		SourceHash:      packed.Hash,
		SourceSizeBytes: packed.Size,
	})
	if err != nil {
		return nil, fmt.Errorf("create deployment: %w", err)
	}
	deploymentID := createResp.Deployment.ID
	progress(fmt.Sprintf("Created deployment %s", deploymentID))

	// 6. Upload tarball to presigned URL.
	progress("Uploading source tarball...")
	f, err := os.Open(packed.Path)
	if err != nil {
		return nil, fmt.Errorf("open packed archive: %w", err)
	}
	defer f.Close()

	var uploadReader io.Reader = f
	if opts.OnUploadProgress != nil {
		uploadReader = &progressReader{
			r:          f,
			total:      packed.Size,
			onProgress: opts.OnUploadProgress,
		}
	}
	if err := cli.UploadFile(ctx, createResp.UploadURL, uploadReader, packed.Size); err != nil {
		return nil, fmt.Errorf("upload tarball: %w", err)
	}
	progress("Upload complete")

	// 7. Confirm deployment (triggers build).
	progress("Confirming deployment (triggering build)...")
	d, err := cli.ConfirmCodeDeployment(ctx, job.ID, deploymentID, client.ConfirmCodeDeploymentRequest{
		ProjectID: opts.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("confirm deployment: %w", err)
	}
	progress(fmt.Sprintf("Build started (deployment %s, status: %s)", d.ID, d.Status))

	// 8. Stream build logs; a successful stream (done sentinel received) means
	// the build has reached a terminal state — skip polling in that case.
	streamCompleted := false
	if opts.OnLogChunk != nil {
		progress("Streaming build logs...")
		onChunk := opts.OnLogChunk
		streamErr := cli.StreamDeploymentLogs(ctx, job.ID, deploymentID, func(chunk string) error {
			onChunk(chunk)
			return nil
		})
		if streamErr == nil {
			streamCompleted = true
		} else if ctx.Err() == nil {
			// Non-fatal: stream ended early; fall through to polling.
			progress(fmt.Sprintf("Log stream ended: %v", streamErr))
		}
	}

	// 9. Fetch final status (single GET if stream completed; polling otherwise).
	var final *client.CodeDeployment
	if streamCompleted {
		final, err = cli.GetCodeDeployment(ctx, job.ID, deploymentID)
		if err != nil {
			return nil, fmt.Errorf("fetch final deployment status: %w", err)
		}
	} else {
		progress("Waiting for build to complete...")
		final, err = pollUntilTerminal(ctx, cli, job.ID, deploymentID, progress)
		if err != nil {
			return nil, err
		}
	}

	return resultFromDeployment(final)
}

func resultFromDeployment(d *client.CodeDeployment) (*Result, error) {
	switch d.Status {
	case "ready":
		return &Result{
			DeploymentID: d.ID,
			Status:       d.Status,
			ImageURI:     d.BuiltImageURI,
		}, nil
	case "timed_out":
		return nil, fmt.Errorf("build timed out (deployment %s)", d.ID)
	default:
		msg := d.ErrorMessage
		if msg == "" {
			msg = "build failed"
		}
		return nil, fmt.Errorf("deployment %s failed: %s", d.ID, msg)
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

// WatchUntilTerminal polls GetCodeDeployment with exponential backoff and calls
// onTick with the current status and elapsed time on each poll iteration.
// The context deadline is respected (use context.WithTimeout for a custom ceiling).
func WatchUntilTerminal(ctx context.Context, cli *client.Client, jobID, deploymentID string, onTick func(status string, elapsed time.Duration)) (*client.CodeDeployment, error) {
	const (
		minInterval = 3 * time.Second
		maxInterval = 30 * time.Second
	)

	start := time.Now()
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

		if onTick != nil {
			onTick(d.Status, time.Since(start))
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("watch cancelled after %s: %w", time.Since(start).Round(time.Second), ctx.Err())
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
