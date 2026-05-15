package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newDebugCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debugging tools",
	}

	cmd.AddCommand(newDebugBundleCommand(state))
	cmd.AddCommand(newDebugRequestCommand(state))
	cmd.AddCommand(newDebugProfileCommand(state))

	return cmd
}

func newDebugProfileCommand(state *appState) *cobra.Command {
	var projectID, period string
	var iterations int

	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Probe API performance (round-trip timings + server performance analytics)",
		Long: `Run a small set of authenticated probes against the configured server, time
each one, and (if --project is set) include the server's performance analytics
snapshot.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if iterations < 1 {
				return fmt.Errorf("--iterations must be >= 1")
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			probes := []struct {
				name string
				fn   func() error
			}{
				{"health", func() error {
					_, err := cli.Health(cmd.Context())
					return err
				}},
			}

			pid, _ := requireProjectID(state, projectID)
			if pid != "" {
				probes = append(probes,
					struct {
						name string
						fn   func() error
					}{"jobs.list", func() error {
						_, err := cli.ListJobs(cmd.Context(), pid)
						return err
					}},
				)
			}

			type probeResult struct {
				Name     string `json:"name"`
				AvgMS    int64  `json:"avg_ms"`
				MinMS    int64  `json:"min_ms"`
				MaxMS    int64  `json:"max_ms"`
				Failures int    `json:"failures"`
			}
			results := make([]probeResult, 0, len(probes))

			for _, p := range probes {
				r := probeResult{Name: p.name, MinMS: -1}
				var totalNS int64
				for range iterations {
					start := time.Now()
					if err := p.fn(); err != nil {
						r.Failures++
						continue
					}
					d := time.Since(start).Milliseconds()
					totalNS += d
					if r.MinMS < 0 || d < r.MinMS {
						r.MinMS = d
					}
					if d > r.MaxMS {
						r.MaxMS = d
					}
				}
				succeeded := iterations - r.Failures
				if succeeded > 0 {
					r.AvgMS = totalNS / int64(succeeded)
				}
				if r.MinMS < 0 {
					r.MinMS = 0
				}
				results = append(results, r)
			}

			var perf *struct {
				PeriodHours int     `json:"period_hours"`
				SuccessRate float64 `json:"success_rate"`
				QueueDepth  int     `json:"queue_depth"`
			}
			if pid != "" {
				hours, err := parsePerfPeriodHours(period)
				if err != nil {
					return err
				}
				snap, err := cli.GetPerformanceAnalytics(cmd.Context(), pid, hours)
				if err == nil {
					perf = &struct {
						PeriodHours int     `json:"period_hours"`
						SuccessRate float64 `json:"success_rate"`
						QueueDepth  int     `json:"queue_depth"`
					}{
						PeriodHours: snap.Throughput.PeriodHours,
						SuccessRate: snap.HealthSummary.SuccessRate,
						QueueDepth:  snap.HealthSummary.QueueDepth,
					}
				}
			}

			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.SectionHeader("API probes", len(results)))
				for _, r := range results {
					fmt.Fprintf(os.Stderr, "  %s  avg=%dms  min=%dms  max=%dms  failures=%d\n",
						styles.Bold.Render(r.Name), r.AvgMS, r.MinMS, r.MaxMS, r.Failures)
				}
				if perf != nil {
					fmt.Fprint(os.Stderr, styles.DetailBox("Server health", []string{
						styles.DetailLine("Period (hours)", fmt.Sprintf("%d", perf.PeriodHours)),
						styles.DetailLine("Success rate", fmt.Sprintf("%.2f%%", perf.SuccessRate*100)),
						styles.DetailLine("Queue depth", fmt.Sprintf("%d", perf.QueueDepth)),
					}))
				}
				return nil
			}

			return printData(state, map[string]any{
				"probes":       results,
				"iterations":   iterations,
				"server_perf":  perf,
				"server_url":   state.opts.serverURL,
				"completed_at": time.Now().UTC(),
			})
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID (enables jobs.list probe + server analytics)")
	cmd.Flags().StringVar(&period, "period", "24h", "analytics period for server health (24h, 72h, 7d, 30d, 90d)")
	cmd.Flags().IntVar(&iterations, "iterations", 3, "probe iterations per call")
	return cmd
}

func newDebugRequestCommand(state *appState) *cobra.Command {
	var body string

	cmd := &cobra.Command{
		Use:   "request <METHOD> <PATH>",
		Short: "Send an authenticated HTTP request to the API and print the response",
		Long: `Send a raw authenticated HTTP request to the Strait API server.

METHOD is the HTTP verb (GET, POST, PATCH, DELETE, etc.).
PATH is relative to the configured server URL (e.g. /v1/jobs).

The response body is printed as indented JSON when parseable, otherwise as text.
Use --debug on the root command to also log timing and status code.`,
		Example: `  strait debug request GET /v1/jobs
  strait debug request GET /v1/jobs?project_id=proj-1
  strait debug request POST /v1/jobs --body '{"name":"test","slug":"test","project_id":"p","endpoint_url":"http://x"}'
  strait debug request DELETE /v1/jobs/job-1`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			urlPath := args[1]

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			return cli.RawRequest(cmd.Context(), method, urlPath, body, state.out())
		},
	}

	cmd.Flags().StringVar(&body, "body", "", "JSON request body")

	return cmd
}

func newDebugBundleCommand(state *appState) *cobra.Command {
	var outputPath string
	var noEvents bool

	cmd := &cobra.Command{
		Use:   "bundle <run-id>",
		Short: "Collect diagnostics into a shareable archive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID := args[0]

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			run, err := cli.GetRun(cmd.Context(), runID)
			if err != nil {
				return fmt.Errorf("fetch run: %w", err)
			}

			job, jobErr := cli.GetJob(cmd.Context(), run.JobID)
			if jobErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not fetch job %s: %v\n", run.JobID, jobErr)
			}

			var events any
			if !noEvents {
				evts, evtErr := cli.ListRunEvents(cmd.Context(), runID, "", "")
				if evtErr == nil {
					events = evts
				}
			}

			env := map[string]string{
				"go_version":  runtime.Version(),
				"os_arch":     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
				"cli_version": version,
				"server_url":  state.opts.serverURL,
				"api_key":     maskBundleKey(state.opts.apiKey),
				"project_id":  state.opts.projectID,
			}

			if outputPath == "" {
				outputPath = fmt.Sprintf("strait-debug-%s-%d.zip", runID, time.Now().Unix())
			}

			f, err := os.Create(outputPath) //nolint:gosec // user-controlled output path for debug bundle
			if err != nil {
				return fmt.Errorf("create zip: %w", err)
			}

			w := zip.NewWriter(f)

			var writeErr error
			if err := writeJSON(w, "run.json", run); err != nil {
				writeErr = fmt.Errorf("write run.json: %w", err)
			}
			if writeErr == nil && job != nil {
				if err := writeJSON(w, "job.json", job); err != nil {
					writeErr = fmt.Errorf("write job.json: %w", err)
				}
			}
			if writeErr == nil && events != nil {
				if err := writeJSON(w, "events.json", events); err != nil {
					writeErr = fmt.Errorf("write events.json: %w", err)
				}
			}
			if writeErr == nil {
				if err := writeJSON(w, "env.json", env); err != nil {
					writeErr = fmt.Errorf("write env.json: %w", err)
				}
			}

			// Close zip writer before file to ensure data is flushed.
			if closeErr := w.Close(); closeErr != nil && writeErr == nil {
				writeErr = fmt.Errorf("finalize zip: %w", closeErr)
			}
			if closeErr := f.Close(); closeErr != nil && writeErr == nil {
				writeErr = fmt.Errorf("close file: %w", closeErr)
			}

			if writeErr != nil {
				_ = os.Remove(outputPath) // Clean up partial file.
				return writeErr
			}

			absPath, _ := filepath.Abs(outputPath)
			if isTTYRich(state) {
				fmt.Fprintln(os.Stderr, styles.Success("Debug bundle created"))
				fmt.Fprintln(os.Stderr, styles.KeyValue("Path", styles.FilePath(absPath)))
				fmt.Fprintln(os.Stderr, styles.KeyValue("Run", runID))
				return nil
			}
			return printData(state, map[string]any{
				"bundle": absPath,
				"run_id": runID,
			})
		},
	}

	cmd.Flags().StringVar(&outputPath, "output", "", "output file path")
	cmd.Flags().BoolVar(&noEvents, "no-events", false, "skip event collection")

	return cmd
}

func writeJSON(w *zip.Writer, name string, data any) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func maskBundleKey(key string) string {
	if len(key) <= 4 {
		return "***"
	}
	return "..." + key[len(key)-4:]
}
