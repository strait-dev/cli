package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/tunnel"

	"github.com/spf13/cobra"
)

// devStartTunnel is the hook used to launch a tunnel and obtain a public URL.
// Tests swap it with a stub that yields a deterministic URL and a no-op
// shutdown function.
var devStartTunnel = startCloudflaredTunnel

func newDevCommand(state *appState) *cobra.Command {
	var port int
	var manifestPath string
	var dir string
	var keepEndpoint bool

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Run a development tunnel and auto-register SDK jobs",
		Long: `Launches a Cloudflare Quick Tunnel pointing at a local serve handler,
reads strait.deploy.json, and updates each job's endpoint_url to
<tunnel-url>/<job-slug>.

Press Ctrl-C to shut down. On shutdown the previously-registered
endpoint URLs are restored (pass --keep-endpoint to leave them
pointing at the tunnel).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectID := state.opts.projectID
			if strings.TrimSpace(projectID) == "" {
				return fmt.Errorf("project is required (set --project, STRAIT_PROJECT, or context)")
			}

			path := manifestPath
			if path == "" {
				root := dir
				if root == "" {
					root = "."
				}
				path = filepath.Join(root, "strait.deploy.json")
			}
			manifest, err := loadDeployManifest(path)
			if err != nil {
				return err
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			tunnelURL, stopTunnel, err := devStartTunnel(ctx, port)
			if err != nil {
				return fmt.Errorf("start tunnel: %w", err)
			}
			defer func() { _ = stopTunnel() }()

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Tunnel ready"))
				fmt.Fprintln(os.Stderr, styles.KeyValue("URL", tunnelURL))
			}

			previous, err := registerDevEndpoints(ctx, cli, projectID, manifest, tunnelURL)
			if err != nil {
				return err
			}
			if isTTYRich(state) {
				for slug, endpoint := range previous {
					fmt.Fprintln(os.Stderr, styles.KeyValue(slug, endpoint))
				}
				fmt.Fprintln(os.Stderr, styles.MutedStyle.Render("Press Ctrl-C to stop"))
			}

			<-ctx.Done()

			if !keepEndpoint {
				restoreCtx, cancelRestore := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancelRestore()
				restoreDevEndpoints(restoreCtx, cli, manifest, previous)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 3000, "local port that the serve handler is listening on")
	cmd.Flags().StringVar(&dir, "dir", "", "project root containing strait.deploy.json (default: current dir)")
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "explicit path to a manifest file (overrides --dir)")
	cmd.Flags().BoolVar(&keepEndpoint, "keep-endpoint", false, "do not restore previous endpoint URLs on shutdown")
	return cmd
}

// startCloudflaredTunnel spawns `cloudflared tunnel --url http://localhost:<port>`,
// parses its stderr for the public URL, and returns a shutdown function.
func startCloudflaredTunnel(ctx context.Context, port int) (string, func() error, error) {
	bin := tunnel.DetectCloudflared()
	if bin == "" {
		return "", nil, fmt.Errorf("cloudflared not found")
	}
	target := fmt.Sprintf("http://localhost:%d", port)
	cmd := exec.CommandContext(ctx, bin, "tunnel", "--url", target, "--no-autoupdate") //nolint:gosec // args are CLI-controlled

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", nil, fmt.Errorf("attach stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("start cloudflared: %w", err)
	}

	urlCh := make(chan string, 1)
	var once sync.Once
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if url, err := tunnel.ParseTunnelURL(line); err == nil {
				once.Do(func() { urlCh <- url })
			}
		}
	}()

	stop := func() error {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-done
		}
		_, _ = io.Copy(io.Discard, stderr)
		return nil
	}

	select {
	case url := <-urlCh:
		return url, stop, nil
	case <-ctx.Done():
		_ = stop()
		return "", nil, ctx.Err()
	case <-time.After(60 * time.Second):
		_ = stop()
		return "", nil, fmt.Errorf("cloudflared did not report a tunnel URL within 60s")
	}
}

// registerDevEndpoints updates each manifest job's endpoint_url to point at
// the tunnel. It returns a map of slug -> previous endpoint so the caller can
// restore them on shutdown.
func registerDevEndpoints(ctx context.Context, cli *client.Client, projectID string, manifest *DeployManifest, tunnelURL string) (map[string]string, error) {
	existing, err := cli.ListJobs(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	bySlug := make(map[string]string, len(existing))
	previousEndpoint := make(map[string]string, len(existing))
	for _, j := range existing {
		bySlug[j.Slug] = j.ID
		previousEndpoint[j.Slug] = j.EndpointURL
	}

	base := strings.TrimRight(tunnelURL, "/")
	for _, job := range manifest.Jobs {
		endpoint := base + "/" + job.Slug
		id, ok := bySlug[job.Slug]
		if !ok {
			req := jobCreateRequest(projectID, job)
			req.EndpointURL = endpoint
			created, err := cli.CreateJob(ctx, req, "")
			if err != nil {
				return previousEndpoint, fmt.Errorf("create %s: %w", job.Slug, err)
			}
			previousEndpoint[job.Slug] = ""
			_ = created
			continue
		}
		_, err := cli.UpdateJob(ctx, id, client.UpdateJobRequest{EndpointURL: &endpoint})
		if err != nil {
			return previousEndpoint, fmt.Errorf("update %s: %w", job.Slug, err)
		}
	}
	return previousEndpoint, nil
}

// restoreDevEndpoints reverts each manifest job's endpoint_url to its
// pre-dev-session value. Failures are logged but not fatal; the caller is in
// shutdown.
func restoreDevEndpoints(ctx context.Context, cli *client.Client, manifest *DeployManifest, previous map[string]string) {
	existing, err := cli.ListJobs(ctx, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "restore endpoints: list jobs failed: %v\n", err)
		return
	}
	bySlug := make(map[string]string, len(existing))
	for _, j := range existing {
		bySlug[j.Slug] = j.ID
	}
	for _, job := range manifest.Jobs {
		id, ok := bySlug[job.Slug]
		if !ok {
			continue
		}
		prev := previous[job.Slug]
		if prev == "" {
			continue
		}
		_, _ = cli.UpdateJob(ctx, id, client.UpdateJobRequest{EndpointURL: &prev})
	}
}
